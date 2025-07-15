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

package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/inspektor-gadget/ig-mcp-server/pkg/tools"
)

const (
	StdioTransport          = "stdio"
	SSETransport            = "sse"
	StreamableHTTPTransport = "streamable-http"
)

var log = slog.Default().With("component", "sever")

var SupportedTransports = []string{StdioTransport, SSETransport, StreamableHTTPTransport}

// Server is the main server for the Inspektor Gadget MCP server.
type Server struct {
	mcpServer   *server.MCPServer
	sseSever    *server.SSEServer
	httpServer  *server.StreamableHTTPServer
	stdioCancel func()
}

// New creates a new instance of the Inspektor Gadget MCP server.
func New(version string, registry *tools.GadgetToolRegistry) *Server {
	ms := server.NewMCPServer(
		"ig-mcp-server",
		version,
		server.WithLogging(),
		server.WithRecovery(),
	)

	// Register callback to register tools
	registry.RegisterCallback(func(tools ...server.ServerTool) {
		ms.SetTools(tools...)
	})

	return &Server{
		mcpServer: ms,
	}
}

// Start starts the MCP server and listens for incoming connections based on transport.
func (s *Server) Start(transport, host, port string) error {
	switch transport {
	case StdioTransport:
		log.Info("Starting MCP server", "transport", transport)
		var ctx context.Context
		ctx, s.stdioCancel = context.WithCancel(context.Background())
		return server.NewStdioServer(s.mcpServer).Listen(ctx, os.Stdin, os.Stdout)
	case SSETransport:
		log.Info("Starting MCP server", "transport", transport, "host", host, "port", port)
		s.sseSever = server.NewSSEServer(s.mcpServer)
		return s.sseSever.Start(net.JoinHostPort(host, port))
	case StreamableHTTPTransport:
		log.Info("Starting MCP server", "transport", transport, "host", host, "port", port)
		s.httpServer = server.NewStreamableHTTPServer(s.mcpServer)
		return s.httpServer.Start(net.JoinHostPort(host, port))
	}
	return fmt.Errorf("unsupported transport: %s", transport)
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Info("Shutting down MCP server")
	if s.stdioCancel != nil {
		s.stdioCancel()
	}
	if s.sseSever != nil {
		if err := s.sseSever.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutting down SSE server: %w", err)
		}
	}
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutting down HTTP server: %w", err)
		}
	}
	return nil
}
