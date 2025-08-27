package deploy

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var log = slog.Default().With("component", "deploy")

func lifecycleHandler(toolRefresher func()) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		action := request.GetString("action", "")
		if action == "" {
			return mcp.NewToolResultText("No action specified, must be one of: " + strings.Join(actions, ", ")), nil
		}
		if !slices.Contains(actions, action) {
			return mcp.NewToolResultError("Invalid action specified, must be one of: " + strings.Join(actions, ", ")), nil
		}

		chartVersion := request.GetString("chart_version", "")

		deployed, _, err := IsInspektorGadgetDeployed(ctx)
		if err != nil {
			return nil, fmt.Errorf("check if Inspektor Gadget is deployed: %w", err)
		}

		hc, err := newHelmClient(false)
		if err != nil {
			return nil, fmt.Errorf("create helm client: %w", err)
		}

		switch action {
		case actionDeployIG:
			if deployed {
				return mcp.NewToolResultError("Inspektor Gadget is already deployed"), nil
			}
			return handleDeploy(hc, toolRefresher, chartVersion)
		case actionUndeployIG:
			if !deployed {
				return mcp.NewToolResultError("Inspektor Gadget is not deployed"), nil
			}
			return handleUndeploy(hc)
		case actionUpgradeIG:
			if !deployed {
				return mcp.NewToolResultError("Inspektor Gadget is not deployed, cannot upgrade"), nil
			}
			return handleUpgrade(hc, chartVersion)
		case actionIsDeployed:
			if deployed {
				return mcp.NewToolResultText("Inspektor Gadget is deployed"), nil
			} else {
				return mcp.NewToolResultText("Inspektor Gadget is not deployed"), nil
			}
		}

		return mcp.NewToolResultText("Action not implemented"), nil
	}
}

func handleDeploy(hc *helmClient, toolRefresher func(), chartVersion string) (*mcp.CallToolResult, error) {
	var chartUrl string
	if chartVersion != "" {
		chartUrl = fmt.Sprintf("%s:%s", defaultChartUrl, chartVersion)
	} else {
		chartUrl = fmt.Sprintf("%s:%s", defaultChartUrl, getChartVersion())
	}

	resp, err := hc.InstallChart(chartUrl, defaultReleaseName, defaultNamespace)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to deploy Inspektor Gadget: %v", err)), nil
	}

	// refresh the tool with gadget information after deployment
	toolRefresher()

	return mcp.NewToolResultText(resp), nil
}

func handleUndeploy(hc *helmClient) (*mcp.CallToolResult, error) {
	resp, err := hc.UninstallChart(defaultReleaseName, defaultNamespace)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to undeploy Inspektor Gadget: %v", err)), nil
	}

	return mcp.NewToolResultText(resp), nil
}

func handleUpgrade(hc *helmClient, chartVersion string) (*mcp.CallToolResult, error) {
	var chartUrl string
	if chartVersion != "" {
		chartUrl = fmt.Sprintf("%s:%s", defaultChartUrl, chartVersion)
	} else {
		chartUrl = fmt.Sprintf("%s:%s", defaultChartUrl, getChartVersion())
	}

	err := hc.CheckRelease(defaultReleaseName, defaultNamespace)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("cannot upgrade Inspektor Gadget: helm release %s in namespace %s does not exist. Did you deploy it manually?", defaultReleaseName, defaultNamespace)), nil
	}

	resp, err := hc.UpgradeChart(chartUrl, defaultReleaseName, defaultNamespace)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to upgrade Inspektor Gadget: %v", err)), nil
	}

	return mcp.NewToolResultText(resp), nil
}
