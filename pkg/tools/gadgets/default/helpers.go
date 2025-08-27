package _default

import (
	"strings"

	"github.com/inspektor-gadget/inspektor-gadget/pkg/gadget-service/api"
)

func defaultParamsFromGadgetInfo(info *api.GadgetInfo) map[string]string {
	params := make(map[string]string)
	for _, p := range info.Params {
		params[p.Prefix+p.Key] = p.DefaultValue
	}
	return params
}

func normalizeToolName(name string) string {
	// Normalize tool name to lowercase and replace spaces with dashes
	return "gadget_" + strings.ReplaceAll(name, " ", "_")
}
