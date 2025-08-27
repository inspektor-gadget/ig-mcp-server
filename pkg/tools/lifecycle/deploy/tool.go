package deploy

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func GetTool(toolRefresher func()) server.ServerTool {
	return server.ServerTool{
		Tool:    lifecycleTool(),
		Handler: lifecycleHandler(toolRefresher),
	}
}

func lifecycleTool() mcp.Tool {
	return mcp.NewTool(
		toolName,
		mcp.WithDescription("Manage the deployment of Inspektor Gadget on target system"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithString("action",
			mcp.Description("Lifecycle action to perform: "+
				actionDeployIG+"(deploy Inspektor Gadget), "+
				actionUndeployIG+"(undeploy Inspektor Gadget), "+
				actionUpgradeIG+"(upgrade Inspektor Gadget), "+
				actionIsDeployed+"(check if Inspektor Gadget is deployed)"),
			mcp.Enum(actions...),
		),
		mcp.WithString("chart_version", mcp.Description("Version of the Inspektor Gadget Helm chart to deploy, only set if user explicitly specifies a version")),
	)
}
