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
	"log"
	"os"

	"github.com/alexandremahdhaoui/forge/internal/cli"
	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/mcputil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Version information (set via ldflags during build)
var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
)

func main() {
	// Check if user needs help
	if len(os.Args) >= 2 && (os.Args[1] == "help" || os.Args[1] == "--help" || os.Args[1] == "-h") {
		printUsage()
		return
	}

	// Otherwise, use standard cli.Bootstrap for MCP mode and version handling
	cli.Bootstrap(cli.Config{
		Name:           "ci-orchestrator",
		Version:        Version,
		CommitSHA:      CommitSHA,
		BuildTimestamp: BuildTimestamp,
		RunMCP:         runMCPServer,
	})
}

func printUsage() {
	fmt.Print(`ci-orchestrator - Orchestrate CI pipelines (not yet implemented)

Usage:
  ci-orchestrator --mcp           Run as MCP server (not yet implemented)
  ci-orchestrator version         Show version information
  ci-orchestrator help            Show this help message

Description:
  ci-orchestrator is a placeholder for future CI pipeline orchestration
  functionality. Currently not implemented.
`)
}

// RunInput represents the input for the run tool.
type RunInput struct {
	Pipeline string `json:"pipeline"`
}

func runMCPServer() error {
	server := mcpserver.New("ci-orchestrator", Version)

	// Register run tool (not yet implemented)
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "run",
		Description: "Run CI pipeline (not yet implemented)",
	}, handleRunTool)

	return server.RunDefault()
}

func handleRunTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input RunInput,
) (*mcp.CallToolResult, any, error) {
	log.Printf("Run called (not yet implemented): pipeline=%s", input.Pipeline)
	return mcputil.ErrorResult("ci-orchestrator: not yet implemented"), nil, nil
}
