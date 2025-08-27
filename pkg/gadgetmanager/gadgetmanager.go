// Copyright 2025 The Inspektor Gadget authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gadgetmanager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/inspektor-gadget/inspektor-gadget/pkg/environment"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/inspektor-gadget/inspektor-gadget/pkg/datasource"
	igjson "github.com/inspektor-gadget/inspektor-gadget/pkg/datasource/formatters/json"
	gadgetcontext "github.com/inspektor-gadget/inspektor-gadget/pkg/gadget-context"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/gadget-service/api"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/operators"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/operators/simple"
	grpcruntime "github.com/inspektor-gadget/inspektor-gadget/pkg/runtime/grpc"
)

const maxResultLen = 64 * 1024 // 64kb

// GadgetManager is an interface for managing gadgets.
type GadgetManager interface {
	// Run starts a gadget with the given image and parameters, returning the output as a string.
	Run(image string, params map[string]string, timeout time.Duration) (string, error)
	// RunDetached starts a gadget with the given image and parameters in the background, returning its ID.
	RunDetached(image string, params map[string]string) (string, error)
	// GetResults returns the stored result buffer from a gadget
	GetResults(id string) (string, error)
	// Stop stops a gadget
	Stop(id string) error
	// GetInfo retrieves information about a gadget image via runtime.
	GetInfo(ctx context.Context, image string) (*api.GadgetInfo, error)
	// GetVersion retrieves the version of Inspektor Gadget installed in the cluster
	GetVersion() (string, error)
	// ListGadgets lists all running gadget instances
	ListGadgets(ctx context.Context) ([]*GadgetInstance, error)
}

// GadgetInstance represents a running gadget instance
type GadgetInstance struct {
	ID          string `json:"id"`
	GadgetImage string `json:"gadgetImage"`
	Params      string `json:"params"`
	CreatedBy   string `json:"createdBy,omitempty"`
	StartedAt   string `json:"startedAt,omitempty"`
}

type gadgetManager struct {
	k8sConfig   *genericclioptions.ConfigFlags
	formatterMu sync.Mutex
	env         string
}

// NewGadgetManager creates a new GadgetManager instance.
func NewGadgetManager(env string, k8sConfig *genericclioptions.ConfigFlags) (GadgetManager, error) {
	if env != "kubernetes" {
		return nil, fmt.Errorf("unsupported gadget manager environment: %s", env)
	}
	// Set the environment to Kubernetes for the gadget runtime
	environment.Environment = environment.Kubernetes
	return &gadgetManager{
		k8sConfig: k8sConfig,
		env:       env,
	}, nil
}

func (g *gadgetManager) Run(image string, params map[string]string, timeout time.Duration) (string, error) {
	const opPriority = 50000
	var jsonBuffer []byte
	myOperator := simple.New("myOperator",
		simple.OnInit(func(gadgetCtx operators.GadgetContext) error {
			for _, d := range gadgetCtx.GetDataSources() {
				jsonFormatter, _ := igjson.New(d,
					igjson.WithShowAll(true),
				)

				// skip data sources that have the annotation "cli.default-output-mode"
				// set to "none"Add commentMore actions
				if m, ok := d.Annotations()["cli.default-output-mode"]; ok && m == "none" {
					continue
				}

				d.Subscribe(func(source datasource.DataSource, data datasource.Data) error {
					g.formatterMu.Lock()
					defer g.formatterMu.Unlock()
					jsonData := jsonFormatter.Marshal(data)
					jsonBuffer = append(jsonBuffer, jsonData...)
					jsonBuffer = append(jsonBuffer, '\n')
					return nil
				}, opPriority)
			}
			return nil
		}),
	)

	gadgetCtx := gadgetcontext.New(
		context.Background(),
		image,
		gadgetcontext.WithDataOperators(
			myOperator,
		),
		gadgetcontext.WithTimeout(timeout),
	)

	runtime, err := g.getRuntime()
	if err != nil {
		return "", fmt.Errorf("getting runtime: %w", err)
	}

	if err = runtime.RunGadget(gadgetCtx, runtime.ParamDescs().ToParams(), params); err != nil {
		return "", fmt.Errorf("running gadget: %w", err)
	}
	return truncateResults(string(jsonBuffer), false), nil
}

func (g *gadgetManager) RunDetached(image string, params map[string]string) (string, error) {
	gadgetCtx := gadgetcontext.New(
		context.Background(),
		image,
	)
	runtime, err := g.getRuntime()
	if err != nil {
		return "", fmt.Errorf("getting runtime: %w", err)
	}

	p := runtime.ParamDescs().ToParams()

	newID := make([]byte, 16)
	rand.Read(newID)
	idString := hex.EncodeToString(newID)

	p.Set(grpcruntime.ParamTags, "createdBy=ig-mcp-server")
	p.Set(grpcruntime.ParamID, idString)
	p.Set(grpcruntime.ParamDetach, "true")
	if err = runtime.RunGadget(gadgetCtx, p, params); err != nil {
		return "", fmt.Errorf("running gadget: %w", err)
	}
	return idString, nil
}

func (g *gadgetManager) Stop(id string) error {
	runtime, err := g.getRuntime()
	if err != nil {
		return fmt.Errorf("getting runtime: %w", err)
	}
	if err = runtime.RemoveGadgetInstance(context.Background(), runtime.ParamDescs().ToParams(), id); err != nil {
		return fmt.Errorf("stopping to gadget: %w", err)
	}
	return nil
}

func (g *gadgetManager) GetResults(id string) (string, error) {
	const opPriority = 50000
	var jsonBuffer []byte
	myOperator := simple.New("myOperator",
		simple.OnInit(func(gadgetCtx operators.GadgetContext) error {
			for _, d := range gadgetCtx.GetDataSources() {
				jsonFormatter, _ := igjson.New(d,
					igjson.WithShowAll(true),
				)

				// skip data sources that have the annotation "cli.default-output-mode"
				// set to "none"Add commentMore actions
				if m, ok := d.Annotations()["cli.default-output-mode"]; ok && m == "none" {
					continue
				}

				d.Subscribe(func(source datasource.DataSource, data datasource.Data) error {
					g.formatterMu.Lock()
					defer g.formatterMu.Unlock()
					jsonData := jsonFormatter.Marshal(data)
					jsonBuffer = append(jsonBuffer, jsonData...)
					jsonBuffer = append(jsonBuffer, '\n')
					return nil
				}, opPriority)
			}
			return nil
		}),
	)

	to, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	gadgetCtx := gadgetcontext.New(
		to,
		id,
		gadgetcontext.WithDataOperators(
			myOperator,
		),
		gadgetcontext.WithID(id),
		gadgetcontext.WithUseInstance(true),
		gadgetcontext.WithTimeout(time.Second),
	)

	runtime, err := g.getRuntime()
	if err != nil {
		return "", fmt.Errorf("getting runtime: %w", err)
	}

	if err = runtime.RunGadget(gadgetCtx, runtime.ParamDescs().ToParams(), map[string]string{}); err != nil {
		return "", fmt.Errorf("attaching to gadget: %w", err)
	}
	return truncateResults(string(jsonBuffer), true), nil
}

func (g *gadgetManager) GetInfo(ctx context.Context, image string) (*api.GadgetInfo, error) {
	gadgetCtx := gadgetcontext.New(
		ctx,
		image,
	)

	runtime, err := g.getRuntime()
	if err != nil {
		return nil, fmt.Errorf("getting runtime: %w", err)
	}

	info, err := runtime.GetGadgetInfo(gadgetCtx, runtime.ParamDescs().ToParams(), nil)
	if err != nil {
		return nil, fmt.Errorf("get gadget info: %w", err)
	}
	return info, nil
}

func (g *gadgetManager) ListGadgets(ctx context.Context) ([]*GadgetInstance, error) {
	rt, err := g.getRuntime()
	if err != nil {
		return nil, fmt.Errorf("getting runtime: %w", err)
	}

	instances, err := rt.GetGadgetInstances(ctx, rt.ParamDescs().ToParams())
	if err != nil {
		return nil, fmt.Errorf("listing gadgets: %w", err)
	}

	var gadgetInstances []*GadgetInstance
	for _, instance := range instances {
		inst := gadgetInstanceFromAPI(instance)
		if inst != nil {
			gadgetInstances = append(gadgetInstances, inst)
		}
	}
	return gadgetInstances, nil
}

func (g *gadgetManager) GetVersion() (string, error) {
	rt, err := g.getRuntime()
	if err != nil {
		return "", fmt.Errorf("getting runtime: %w", err)
	}

	info, err := rt.GetInfo()
	if err != nil {
		return "", fmt.Errorf("getting info: %w", err)
	}
	return info.ServerVersion, nil
}

func truncateResults(results string, latest bool) string {
	if len(results) <= maxResultLen {
		return fmt.Sprintf("\n<results>%s</results>\n", results)
	}

	var truncated string
	if latest {
		truncated = results[len(results)-maxResultLen:]
	} else {
		truncated = results[:maxResultLen] + "…"
	}

	return fmt.Sprintf("\n<isTruncated>true</isTruncated>\n<results>%s</results>\n", truncated)
}

func (g *gadgetManager) getRuntime() (*grpcruntime.Runtime, error) {
	if g.env == "kubernetes" {
		rt := grpcruntime.New(grpcruntime.WithConnectUsingK8SProxy)
		if err := rt.Init(nil); err != nil {
			return nil, fmt.Errorf("initializing gadget runtime: %w", err)
		}

		restConfig, err := g.k8sConfig.ToRESTConfig()
		if err != nil {
			return nil, fmt.Errorf("creating REST config: %w", err)
		}
		rt.SetRestConfig(restConfig)

		return rt, nil
	}
	return nil, fmt.Errorf("unsupported gadget manager environment: %s", g.env)
}

func gadgetInstanceFromAPI(instance *api.GadgetInstance) *GadgetInstance {
	if instance == nil {
		return nil
	}

	var createdBy string
	for _, tag := range instance.Tags {
		if strings.HasPrefix(tag, "createdBy=") {
			createdBy = strings.TrimPrefix(tag, "createdBy=")
			break
		}
	}

	var params []string
	for k, v := range instance.GadgetConfig.ParamValues {
		if v == "" {
			continue
		}
		params = append(params, fmt.Sprintf("%s=%q", k, v))
	}

	return &GadgetInstance{
		ID:          instance.Id,
		Params:      strings.Join(params, ","),
		GadgetImage: instance.GadgetConfig.ImageName,
		CreatedBy:   createdBy,
		StartedAt:   time.Unix(instance.TimeCreated, 0).Format(time.RFC3339),
	}
}
