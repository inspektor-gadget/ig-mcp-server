package _default

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"text/template"
	"time"

	"github.com/inspektor-gadget/inspektor-gadget/pkg/gadget-service/api"
	metadatav1 "github.com/inspektor-gadget/inspektor-gadget/pkg/metadata/v1"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gopkg.in/yaml.v3"

	"github.com/inspektor-gadget/ig-mcp-server/pkg/discoverer"
	"github.com/inspektor-gadget/ig-mcp-server/pkg/gadgetmanager"
)

func GetTools(ctx context.Context, mgr gadgetmanager.GadgetManager, env string, gadgets []discoverer.Gadget) []server.ServerTool {
	sem := make(chan struct{}, 10) // Limit concurrency to 10
	var wg sync.WaitGroup
	resultsChan := make(chan struct {
		img  string
		info *api.GadgetInfo
		err  error
	}, len(gadgets))

	for _, g := range gadgets {
		wg.Add(1)
		sem <- struct{}{}
		go func(image string) {
			defer func() {
				wg.Done()
				<-sem
			}()
			for i := 0; i < 3; i++ {
				info, err := mgr.GetInfo(ctx, image)
				if err == nil {
					resultsChan <- struct {
						img  string
						info *api.GadgetInfo
						err  error
					}{img: image, info: info, err: nil}
					return
				}
				log.Warn("Failed to get gadget info, retrying", "image", image, "attempt", i+1, "error", err)
				time.Sleep(2 * time.Second)
			}
			info, err := mgr.GetInfo(ctx, image)
			resultsChan <- struct {
				img  string
				info *api.GadgetInfo
				err  error
			}{img: g.Image, info: info, err: err}
		}(g.Image)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	var tools []server.ServerTool
	for result := range resultsChan {
		if result.err != nil {
			log.Warn("Skipping gadget image due to error", "image", result.img, "error", result.err)
			continue
		}
		info := result.info
		t, err := gadgetsTool(env, info)
		if err != nil {
			log.Warn("Skipping gadget due to error creating tool", "image", result.img, "error", err)
			continue
		}
		h := gadgetHandler(mgr, info)
		st := server.ServerTool{
			Tool:    t,
			Handler: h,
		}
		tools = append(tools, st)
	}

	return tools
}

func gadgetsTool(env string, info *api.GadgetInfo) (mcp.Tool, error) {
	var tool mcp.Tool
	var metadata *metadatav1.GadgetMetadata
	err := yaml.Unmarshal(info.Metadata, &metadata)
	if err != nil {
		return tool, fmt.Errorf("unmarshalling gadget metadata: %w", err)
	}
	tmpl, err := template.ParseFS(templates, "templates/toolDescription.tmpl")
	if err != nil {
		return tool, fmt.Errorf("parsing template: %w", err)
	}
	var fields []FieldData
	if len(info.DataSources) > 0 {
		for _, field := range info.DataSources[0].Fields {
			fields = append(fields, FieldData{
				Name:           field.FullName,
				Description:    field.Annotations[metadatav1.DescriptionAnnotation],
				PossibleValues: field.Annotations[metadatav1.ValueOneOfAnnotation],
			})
		}
	}
	var out bytes.Buffer
	td := ToolData{
		Name:        normalizeToolName(metadata.Name),
		Description: metadata.Description,
		Environment: env,
		Fields:      fields,
	}
	if err = tmpl.Execute(&out, td); err != nil {
		return tool, fmt.Errorf("executing template for gadget %s: %w", info.ImageName, err)
	}
	params := make(map[string]interface{})
	for _, p := range info.Params {
		params[p.Prefix+p.Key] = map[string]interface{}{
			"type":        "string",
			"description": p.Description,
		}
	}

	opts := []mcp.ToolOption{
		mcp.WithDescription(out.String()),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithObject("params",
			mcp.Required(),
			mcp.Description("key-value pairs of parameters to pass to the gadget"),
			mcp.Properties(params),
		),
		mcp.WithNumber("duration",
			mcp.Description("Duration in seconds to run the gadget. Use 0 to run  in background/continuously."),
		),
	}
	tool = mcp.NewTool(
		normalizeToolName(metadata.Name),
		opts...,
	)
	return tool, nil
}
