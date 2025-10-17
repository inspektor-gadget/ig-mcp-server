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

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
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
	remoteAddr  string
}

// NewGadgetManager creates a new GadgetManager instance.
func NewGadgetManager(env string, linuxRemoteAddress string, k8sConfig *genericclioptions.ConfigFlags) (GadgetManager, error) {
	if env != "kubernetes" && env != "linux" {
		return nil, fmt.Errorf("unsupported gadget manager environment: %s", env)
	}
	if env == "linux" && linuxRemoteAddress == "" {
		return nil, fmt.Errorf("linuxRemoteAddress must be set when environment is linux")
	}
	return &gadgetManager{
		k8sConfig:  k8sConfig,
		env:        env,
		remoteAddr: linuxRemoteAddress,
	}, nil
}

func (g *gadgetManager) Run(image string, params map[string]string, timeout time.Duration) (string, error) {
	var res strings.Builder
	gadgetCtx := gadgetcontext.New(
		context.Background(),
		image,
		gadgetcontext.WithDataOperators(
			g.outputOperator(func(buf []byte) {
				res.Write(buf)
				res.WriteByte('\n')
			}),
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
	return truncateResults(res.String(), false), nil
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
	var res strings.Builder
	to, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	gadgetCtx := gadgetcontext.New(
		to,
		id,
		gadgetcontext.WithDataOperators(
			g.outputOperator(func(buf []byte) {
				res.Write(buf)
				res.WriteByte('\n')
			}),
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
	return truncateResults(res.String(), true), nil
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
		environment.Environment = environment.Kubernetes
		rt := grpcruntime.New(grpcruntime.WithConnectUsingK8SProxy)
		if err := rt.Init(rt.GlobalParamDescs().ToParams()); err != nil {
			return nil, fmt.Errorf("initializing gadget runtime: %w", err)
		}

		restConfig, err := g.k8sConfig.ToRESTConfig()
		if err != nil {
			return nil, fmt.Errorf("creating REST config: %w", err)
		}
		rt.SetRestConfig(restConfig)

		return rt, nil
	}
	if g.env == "linux" {
		environment.Environment = environment.Local
		rt := grpcruntime.New()
		gp := rt.GlobalParamDescs().ToParams()
		err := gp.Set(grpcruntime.ParamRemoteAddress, g.remoteAddr)
		if err != nil {
			return nil, fmt.Errorf("setting remote address: %w", err)
		}
		if err = rt.Init(gp); err != nil {
			return nil, fmt.Errorf("initializing gadget runtime: %w", err)
		}
		return rt, nil
	}
	return nil, fmt.Errorf("unsupported gadget manager environment: %s", g.env)
}

func (g *gadgetManager) outputOperator(cb func(buf []byte)) operators.DataOperator {
	const opPriority = 50000
	return simple.New("outputOperator",
		simple.OnInit(func(gadgetCtx operators.GadgetContext) error {
			for _, d := range gadgetCtx.GetDataSources() {
				// skip data sources that have the annotation "cli.default-output-mode"
				if m, ok := d.Annotations()["cli.default-output-mode"]; ok && m == "none" {
					continue
				}

				// handle adding a raw string field for certain content types
				restField := d.Annotations()["ebpf.rest.name"]
				var restAcc datasource.FieldAccessor
				var restStrAcc datasource.FieldAccessor
				var err error
				if restField != "" {
					restAcc = d.GetField(restField)
					ct, ok := restAcc.Annotations()["content-type"]
					if ok && ct == "application/x-raw-packet" {
						restStrAcc, err = d.AddField(restField+"_string", api.Kind_String)
						if err != nil {
							return fmt.Errorf("adding raw string field accessor: %w", err)
						}
					}
				}

				jsonFormatter, _ := igjson.New(d,
					igjson.WithShowAll(true),
				)

				d.Subscribe(func(source datasource.DataSource, data datasource.Data) error {
					g.formatterMu.Lock()
					defer g.formatterMu.Unlock()
					if restAcc != nil && restStrAcc != nil {
						pktStr := gopacket.NewPacket(restAcc.Get(data), layers.LinkTypeEthernet, gopacket.Default).String()
						err = restStrAcc.Set(data, []byte(pktStr))
						if err != nil {
							return fmt.Errorf("setting raw string field: %w", err)
						}
					}
					jsonData := jsonFormatter.Marshal(data)
					cb(jsonData)
					return nil
				}, opPriority)
			}
			return nil
		}),
	)
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
