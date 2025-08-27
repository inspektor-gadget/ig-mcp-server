package gadgets

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/inspektor-gadget/ig-mcp-server/pkg/gadgetmanager"
)

var log = slog.Default().With("component", "lifecycle_gadgets_tool")

func lifecycleHandler(mgr gadgetmanager.GadgetManager) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		action := request.GetString("action", "")
		if action == "" {
			return mcp.NewToolResultText("No action specified, must be one of: " + strings.Join(gadgetActions, ", ")), nil
		}
		if !slices.Contains(gadgetActions, action) {
			return mcp.NewToolResultError("Invalid action specified, must be one of: " + strings.Join(gadgetActions, ", ")), nil
		}

		gadgetID := request.GetString("gadget_id", "")
		if gadgetID == "" && (action == actionGetResults || action == actionStopGadget) {
			return mcp.NewToolResultError("A gadget_id must be specified for " + action), nil
		}

		switch action {
		case actionListGadgets:
			return handleListGadgets(ctx, mgr)
		case actionGetResults:
			return handleGetGadgetResults(ctx, mgr, gadgetID)
		case actionStopGadget:
			return handleStopGadget(ctx, mgr, gadgetID)
		}

		return mcp.NewToolResultText("Action not implemented"), nil
	}
}

func handleListGadgets(ctx context.Context, mgr gadgetmanager.GadgetManager) (*mcp.CallToolResult, error) {
	log.Debug("Listing gadgets")
	gadgets, err := mgr.ListGadgets(ctx)
	if err != nil {
		return mcp.NewToolResultError("Failed to list gadgets: " + err.Error()), nil
	}
	if len(gadgets) == 0 {
		return mcp.NewToolResultText("No running gadgets found"), nil
	}

	JSONData, err := json.Marshal(gadgets)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal gadgets to JSON: " + err.Error()), nil
	}

	return mcp.NewToolResultText(string(JSONData)), nil
}

func handleGetGadgetResults(_ context.Context, mgr gadgetmanager.GadgetManager, gadgetID string) (*mcp.CallToolResult, error) {
	log.Debug("Getting gadget results", "gadget_id", gadgetID)
	result, err := mgr.GetResults(gadgetID)
	if err != nil {
		return mcp.NewToolResultError("Failed to get gadget results: " + err.Error()), nil
	}
	return mcp.NewToolResultText(result), nil
}

func handleStopGadget(_ context.Context, mgr gadgetmanager.GadgetManager, gadgetID string) (*mcp.CallToolResult, error) {
	log.Debug("Stopping gadget", "gadget_id", gadgetID)
	err := mgr.Stop(gadgetID)
	if err != nil {
		return mcp.NewToolResultError("Failed to stop gadget: " + err.Error()), nil
	}
	return mcp.NewToolResultText("Gadget with ID " + gadgetID + " has been stopped"), nil
}
