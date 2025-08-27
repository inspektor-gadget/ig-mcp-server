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

package tools

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/mark3labs/mcp-go/server"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/inspektor-gadget/ig-mcp-server/pkg/discoverer"
	"github.com/inspektor-gadget/ig-mcp-server/pkg/gadgetmanager"
	gadgetsdefault "github.com/inspektor-gadget/ig-mcp-server/pkg/tools/gadgets/default"
	gadgetsephemeral "github.com/inspektor-gadget/ig-mcp-server/pkg/tools/gadgets/ephemeral"
	lifecycledeploy "github.com/inspektor-gadget/ig-mcp-server/pkg/tools/lifecycle/deploy"
	lifecyclegadgets "github.com/inspektor-gadget/ig-mcp-server/pkg/tools/lifecycle/gadgets"
)

var log = slog.Default().With("component", "tools")

type ToolRegistryCallback func(tool ...server.ServerTool)

// GadgetToolRegistry is a simple registry for server tools based on gadgets.
type GadgetToolRegistry struct {
	tools     map[string]server.ServerTool
	mu        sync.Mutex
	callbacks []ToolRegistryCallback
	readonly  bool

	gadgetMgr  gadgetmanager.GadgetManager
	k8sConfig  *genericclioptions.ConfigFlags
	discoverer discoverer.Discoverer
	env        string
}

// NewToolRegistry creates a new GadgetToolRegistry instance.
func NewToolRegistry(manager gadgetmanager.GadgetManager, env string, k8sConfig *genericclioptions.ConfigFlags, discoverer discoverer.Discoverer, readonly bool) *GadgetToolRegistry {
	return &GadgetToolRegistry{
		tools:      make(map[string]server.ServerTool),
		gadgetMgr:  manager,
		env:        env,
		k8sConfig:  k8sConfig,
		discoverer: discoverer,
		readonly:   readonly,
	}
}

func (r *GadgetToolRegistry) all() []server.ServerTool {
	tools := make([]server.ServerTool, 0, len(r.tools))
	for _, tool := range r.tools {
		if r.readonly && tool.Tool.Annotations.ReadOnlyHint != nil && !*tool.Tool.Annotations.ReadOnlyHint {
			// If the registry is in read-only mode, skip tools that do not have the read-only hint annotation
			continue
		}
		tools = append(tools, tool)
	}
	return tools
}

func (r *GadgetToolRegistry) RegisterTools(tools ...server.ServerTool) {
	for _, tool := range tools {
		log.Debug("Registering tool", "name", tool.Tool.Name)
		r.tools[tool.Tool.Name] = tool
	}

	for _, callback := range r.callbacks {
		log.Debug("Invoking tool registry callback", "tools_count", len(r.tools))
		callback(r.all()...)
	}
}

func (r *GadgetToolRegistry) RegisterCallback(callback ToolRegistryCallback) {
	r.callbacks = append(r.callbacks, callback)
}

func (r *GadgetToolRegistry) Prepare(ctx context.Context, images []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	gadgets := discoverer.FromImages(images)

	var err error
	if len(gadgets) == 0 && r.discoverer != nil {
		gadgets, err = r.discoverer.ListGadgets()
		if err != nil {
			log.Warn("listing gadgets from discoverer", "error", err)
		}
	}

	// if still no gadgets, fall back to builtin discoverer
	if len(gadgets) == 0 {
		dis := discoverer.NewBuiltinDiscoverer()
		gadgets, err = dis.ListGadgets()
		if err != nil {
			return fmt.Errorf("listing gadgets from builtin discoverer: %w", err)
		}
	}

	var tools []server.ServerTool
	// Register gadgets lifecycle tools and environment-specific tools
	if r.env == "kubernetes" {
		tools = append(tools, r.getK8sTools(ctx, gadgets)...)
	}

	// Register all tools in the registry
	r.RegisterTools(tools...)

	return nil
}

func (r *GadgetToolRegistry) getK8sTools(ctx context.Context, gadgets []discoverer.Gadget) []server.ServerTool {
	var tools []server.ServerTool
	// Register Gadget lifecycle tool
	tools = append(tools, lifecyclegadgets.GetTool(r.gadgetMgr))
	// Register Inspektor Gadget lifecycle tool since we are in Kubernetes
	toolRefresher := func() {
		go func() {
			tools = r.getK8sTools(ctx, gadgets)
			r.RegisterTools(tools...)
		}()
	}
	tools = append(tools, lifecycledeploy.GetTool(toolRefresher))
	deployed, _, err := lifecycledeploy.IsInspektorGadgetDeployed(ctx)
	if err != nil {
		log.Warn("Failed to check if Inspektor Gadget is deployed, skipping Inspektor Gadget lifecycle tool", "error", err)
	}
	// Register tools based on gadgets only if Inspektor Gadget is deployed
	if deployed {
		tools = append(tools, gadgetsdefault.GetTools(ctx, r.gadgetMgr, r.env, gadgets)...)
	} else {
		tools = append(tools, gadgetsephemeral.GetTools(gadgets)...)
	}
	return tools
}
