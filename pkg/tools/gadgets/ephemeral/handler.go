package ephemeral

import (
	"context"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var log = slog.Default().With("component", "ephemeral_tool")

func ephemeralHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultError(
			"Inspektor Gadget is not deployed, please deploy it using the ig_deploy tool first" +
				"or if you just deployed it, please restart the ig-mcp-server or MCP gateway to refresh the tool list",
		), nil
	}
}
