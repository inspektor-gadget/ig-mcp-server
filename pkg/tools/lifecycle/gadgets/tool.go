package gadgets

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/inspektor-gadget/ig-mcp-server/pkg/gadgetmanager"
)

func GetTool(mgr gadgetmanager.GadgetManager) server.ServerTool {
	return server.ServerTool{
		Tool:    lifecycleTool(),
		Handler: lifecycleHandler(mgr),
	}
}

func lifecycleTool() mcp.Tool {
	return mcp.NewTool(
		toolName,
		mcp.WithDescription("Manage running gadgets"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithString("action",
			mcp.Description("Lifecycle action to perform: "+
				actionListGadgets+"(list running gadgets), "+
				actionStopGadget+"(stop a running gadget using its ID), "+
				actionGetResults+"(get results of a running gadget using its ID, only available before stopping it)"),
			mcp.Enum(gadgetActions...),
		),
		mcp.WithString("gadget_id", mcp.Description("ID of the gadget to stop or get results from, required for "+actionStopGadget+" and "+actionGetResults)),
	)
}
