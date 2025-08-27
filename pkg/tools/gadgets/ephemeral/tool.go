package ephemeral

import (
	"fmt"
	"strings"

	"github.com/distribution/reference"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/inspektor-gadget/ig-mcp-server/pkg/discoverer"
)

func GetTools(gadgets []discoverer.Gadget) []server.ServerTool {
	var tools []server.ServerTool
	for _, g := range gadgets {
		n, err := extractNameFrom(g.Image)
		if err != nil {
			log.Warn("Failed to extract tool name from image", "image", g.Image, "error", err)
			continue
		}

		t := mcp.NewTool(
			"gadget_"+n,
			mcp.WithDescription(g.Description),
		)

		tools = append(tools, server.ServerTool{
			Tool:    t,
			Handler: ephemeralHandler(),
		})
	}
	return tools
}

func extractNameFrom(image string) (string, error) {
	ref, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", fmt.Errorf("parsing image reference %s: %w", image, err)
	}
	parts := strings.Split(reference.TrimNamed(ref).Name(), "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid image reference: %s", image)
	}
	return normalizeToolName(parts[len(parts)-1]), nil
}

func normalizeToolName(name string) string {
	// Normalize tool name to lowercase and replace spaces with dashes
	return strings.ReplaceAll(name, " ", "_")
}
