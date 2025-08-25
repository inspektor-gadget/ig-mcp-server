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

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/inspektor-gadget/ig-mcp-server/pkg/discoverer"
	"github.com/inspektor-gadget/ig-mcp-server/pkg/gadgetmanager"
	"github.com/inspektor-gadget/ig-mcp-server/pkg/server"
	"github.com/inspektor-gadget/ig-mcp-server/pkg/tools"
)

// This variable is used by the "version" command and is set during build
var version = "undefined"

var (
	// MCP server configuration
	readOnly      = flag.Bool("read-only", false, "run the server in read-only mode")
	transport     = flag.String("transport", "stdio", fmt.Sprintf("transport to use (%s)", strings.Join(server.SupportedTransports, ", ")))
	transportHost = flag.String("transport-host", "localhost", "host for the transport")
	transportPort = flag.String("transport-port", "8080", "port for the transport")
	// Inspektor Gadget configuration
	environment                   = flag.String("environment", "kubernetes", "environment to use (currently only 'kubernetes' is supported)")
	gadgetImages                  = flag.String("gadget-images", "", "comma-separated list of gadget images to use (e.g. 'trace_dns:latest,trace_open:latest')")
	gadgetDiscoverer              = flag.String("gadget-discoverer", "artifacthub", "gadget discoverer to use (artifacthub)")
	artifactHubDiscovererOfficial = flag.Bool("artifacthub-official", true, "use only official gadgets from Artifact Hub")
	// Server configuration
	logLevel    = flag.String("log-level", "", "log level (debug, info, warn, error)")
	versionFlag = flag.Bool("version", false, "print version and exit")
	// Kubernetes configuration
	k8sConfig = genericclioptions.NewConfigFlags(false)
)

var log = slog.Default().With("component", "ig-mcp-server")

func init() {
	if k8sConfig.KubeConfig != nil {
		flag.StringVar(k8sConfig.KubeConfig, "kubeconfig", "", "Path to the kubeconfig file to use")
	}
	if k8sConfig.Context != nil {
		flag.StringVar(k8sConfig.Context, "context", "", "The name of the kubeconfig context to use")
	}
	if k8sConfig.AuthInfoName != nil {
		flag.StringVar(k8sConfig.AuthInfoName, "user", "", "The name of the kubeconfig user to use")
	}
	if k8sConfig.BearerToken != nil {
		flag.StringVar(k8sConfig.BearerToken, "token", "", "Bearer token to use for authentication")
	}
}

func main() {
	flag.Parse()

	if *versionFlag {
		log.Info("Inspektor Gadget MCP Server", "version", version)
		os.Exit(0)
	}

	if *environment != "kubernetes" {
		logFatal("unsupported environment, only 'kubernetes' is supported")
	}

	if *logLevel != "" {
		l, err := parseLogLevel(*logLevel)
		if err != nil {
			logFatal("invalid log level", "error", err)
		}
		slog.SetLogLoggerLevel(l)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mgr, err := gadgetmanager.NewGadgetManager(*environment, k8sConfig)
	if err != nil {
		logFatal("failed to create gadget manager", "error", err)
	}
	defer mgr.Close()
	var dis discoverer.Discoverer
	if *gadgetDiscoverer != "" {
		dis, err = discoverer.New(*gadgetDiscoverer, discoverer.WithArtifactHubOfficialOnly(*artifactHubDiscovererOfficial))
		if err != nil {
			logFatal("failed to create gadget discoverer", "error", err)
		}
	}
	registry := tools.NewToolRegistry(mgr, *environment, k8sConfig, dis, *readOnly)
	srv := server.New(version, registry)

	images := strings.Split(*gadgetImages, ",")
	if err = registry.Prepare(ctx, images); err != nil {
		logFatal("failed to prepare tool registry", "error", err)
	}

	go func() {
		defer stop()
		if err = srv.Start(*transport, *transportHost, *transportPort); err != nil {
			log.Error("failed to start server", "error", err)
		}
	}()

	<-ctx.Done()
	log.Info("Received shutdown signal, shutting down server")
	if err = srv.Shutdown(ctx); err != nil {
		logFatal("failed to shutdown server", "error", err)
	}
}

func logFatal(msg string, args ...any) {
	log.Error(msg, args...)
	os.Exit(1)
}

func parseLogLevel(level string) (slog.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	}
	return 0, fmt.Errorf("invalid log level: %s", level)
}
