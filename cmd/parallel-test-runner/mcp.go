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

package main

import (
	"context"
	"fmt"

	"github.com/alexandremahdhaoui/forge/pkg/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// runMCPServer starts the parallel-test-runner MCP server.
// This custom implementation registers the recursive config-validate handler.
func runMCPServer() error {
	// Use generated MCP server setup (registers run and config-validate tools)
	server, err := SetupMCPServer(Name, Version, Run)
	if err != nil {
		return fmt.Errorf("setting up MCP server: %w", err)
	}

	// Register docs MCP tools (docs-list, docs-get)
	if err := RegisterDocsMCPTools(server); err != nil {
		return fmt.Errorf("registering docs MCP tools: %w", err)
	}

	// Override config-validate tool with custom recursive validation handler
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "config-validate",
		Description: "Validate parallel-test-runner configuration and recursively validate sub-runners",
	}, handleRecursiveConfigValidate)

	if err := server.Run(context.Background()); err != nil {
		return fmt.Errorf("running MCP server: %w", err)
	}

	return nil
}
