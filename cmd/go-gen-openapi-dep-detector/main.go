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
	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/alexandremahdhaoui/forge/pkg/mcputil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const Name = "go-gen-openapi-dep-detector"

// Version information (set via ldflags during build)
var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
)

// docsConfig is the configuration for the docs subcommand.
var docsConfig = &enginedocs.Config{
	EngineName:   Name,
	LocalDir:     "cmd/go-gen-openapi-dep-detector/docs",
	BaseURL:      "https://raw.githubusercontent.com/alexandremahdhaoui/forge/refs/heads/main",
	RequiredDocs: []string{"usage", "schema"},
}

// ----------------------------------------------------- MAIN ------------------------------------------------------- //

func main() {
	cli.Bootstrap(cli.Config{
		Name:           Name,
		Version:        Version,
		CommitSHA:      CommitSHA,
		BuildTimestamp: BuildTimestamp,
		RunCLI:         run,
		RunMCP:         runMCPServer,
		SuccessHandler: printSuccess,
		FailureHandler: printFailure,
		DocsConfig:     docsConfig,
	})
}

// ----------------------------------------------------- RUN -------------------------------------------------------- //

// run executes the main logic of the go-gen-openapi-dep-detector tool in direct CLI mode.
// This mode is for standalone execution (not via MCP).
func run() error {
	_, _ = fmt.Fprintln(os.Stderr, "Direct CLI execution not yet implemented")
	_, _ = fmt.Fprintln(os.Stderr, "   Use --mcp mode or call via forge")
	return fmt.Errorf("direct CLI execution not yet implemented")
}

// ----------------------------------------------------- MCP -------------------------------------------------------- //

// runMCPServer starts the MCP server for OpenAPI dependency detection.
func runMCPServer() error {
	server := mcpserver.New(Name, Version)

	// Register detectDependencies tool
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "detectDependencies",
		Description: "Detect dependencies for OpenAPI code generation (spec files)",
	}, handleDetectDependencies)

	if err := enginedocs.RegisterDocsTools(server, *docsConfig); err != nil {
		return err
	}

	return server.RunDefault()
}

// handleDetectDependencies handles the "detectDependencies" tool call from MCP clients.
func handleDetectDependencies(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input mcptypes.DetectOpenAPIDependenciesInput,
) (*mcp.CallToolResult, any, error) {
	log.Printf("Detecting OpenAPI dependencies: %d spec files", len(input.SpecSources))

	output, err := DetectOpenAPIDependencies(input)
	if err != nil {
		return mcputil.ErrorResult(fmt.Sprintf("OpenAPI dependency detection failed: %v", err)), nil, nil
	}

	result, artifact := mcputil.SuccessResultWithArtifact(
		fmt.Sprintf("Detected %d dependencies for OpenAPI generation", len(output.Dependencies)),
		output,
	)
	return result, artifact, nil
}

// ----------------------------------------------------- PRINT HELPERS ----------------------------------------------- //

func printSuccess() {
	_, _ = fmt.Fprintln(os.Stdout, "OpenAPI dependency detection completed successfully")
}

func printFailure(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "Error detecting OpenAPI dependencies\n%s\n", err.Error())
}
