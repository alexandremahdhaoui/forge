// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mcpserver

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps the MCP server with common functionality.
type Server struct {
	server *mcp.Server
}

// New creates a new MCP server with the given name and version.
func New(name, version string) *Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    name,
		Version: version,
	}, nil)

	return &Server{
		server: server,
	}
}

// RegisterTool registers a tool with the MCP server.
// The handler must be a function with signature:
// func(ctx context.Context, req *mcp.CallToolRequest, input In) (*mcp.CallToolResult, any, error)
func RegisterTool[In any](s *Server, tool *mcp.Tool, handler func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, any, error)) {
	mcp.AddTool(s.server, tool, handler)
}

// Run starts the MCP server with stdio transport.
// It reads JSON-RPC requests from stdin and writes responses to stdout.
// All logs should go to stderr only to avoid corrupting the JSON-RPC stream.
func (s *Server) Run(ctx context.Context) error {
	if err := s.server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Printf("MCP server failed: %v", err)
		return err
	}
	return nil
}

// RunDefault starts the MCP server with a background context.
func (s *Server) RunDefault() error {
	return s.Run(context.Background())
}
