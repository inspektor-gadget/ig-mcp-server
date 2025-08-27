package _default

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"time"

	"github.com/inspektor-gadget/inspektor-gadget/pkg/gadget-service/api"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/inspektor-gadget/ig-mcp-server/pkg/gadgetmanager"
)

//go:embed templates
var templates embed.FS

var log = slog.Default().With("component", "gadgets_tool")

type ToolData struct {
	Name        string
	Description string
	Environment string
	Fields      []FieldData
}

type FieldData struct {
	Name           string
	Description    string
	PossibleValues string
}

func gadgetHandler(mgr gadgetmanager.GadgetManager, info *api.GadgetInfo) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		duration := 10 * time.Second
		params := defaultParamsFromGadgetInfo(info)
		args := request.GetArguments()
		background := false
		if args != nil {
			if t, ok := args["duration"].(float64); ok {
				duration = time.Duration(t) * time.Second
			}
			if duration == 0 {
				background = true
			}
			// set map-fetch-interval to half of the duration to limit the volume of data fetched
			if _, ok := params["operator.oci.ebpf.map-fetch-interval"]; ok && !background {
				params["operator.oci.ebpf.map-fetch-interval"] = (duration / 2).String()
			}
			// If params is provided, merge it with the default parameters
			if p, ok := args["params"].(map[string]interface{}); ok {
				for k, v := range p {
					if strVal, ok := v.(string); ok {
						params[k] = strVal
					} else {
						return nil, fmt.Errorf("invalid type for parameter %s: expected string, got %T", k, v)
					}
				}
			}
		}

		if background {
			id, err := mgr.RunDetached(info.ImageName, params)
			if err != nil {
				return nil, fmt.Errorf("running gadget: %w", err)
			}
			return mcp.NewToolResultText(fmt.Sprintf("The gadget has been started with ID %s.", id)), nil
		}

		log.Debug("Running gadget", "image", info.ImageName, "params", params, "duration", duration)
		resp, err := mgr.Run(info.ImageName, params, duration)
		if err != nil {
			return nil, fmt.Errorf("starting gadget %s: %w", info.ImageName, err)
		}
		return mcp.NewToolResultText(resp), nil
	}
}
