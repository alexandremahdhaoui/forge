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

const Name = "go-dependency-detector"

// Version information (set via ldflags during build)
var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
)

// docsConfig is the configuration for the docs subcommand.
var docsConfig = &enginedocs.Config{
	EngineName:   Name,
	LocalDir:     "cmd/go-dependency-detector/docs",
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

// run executes the main logic of the go-dependency-detector tool in direct CLI mode.
// This mode is for standalone execution (not via MCP).
func run() error {
	// Read environment variables to get filepath and funcName
	// For direct CLI execution, we expect these to be passed via env vars or flags
	// For now, this is a stub as the primary interface is MCP
	_, _ = fmt.Fprintln(os.Stderr, "⚠ Direct CLI execution not yet implemented")
	_, _ = fmt.Fprintln(os.Stderr, "   Use --mcp mode or call via forge")
	return fmt.Errorf("direct CLI execution not yet implemented")
}

// ----------------------------------------------------- MCP -------------------------------------------------------- //

// runMCPServer starts the MCP server for dependency detection.
func runMCPServer() error {
	server := mcpserver.New(Name, Version)

	// Register detectDependencies tool
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "detectDependencies",
		Description: "Detect all dependencies (local files and external packages) for a Go function",
	}, handleDetectDependencies)

	if err := enginedocs.RegisterDocsTools(server, *docsConfig); err != nil {
		return err
	}

	return server.RunDefault()
}

// handleDetectDependencies handles the "detect-dependencies" tool call from MCP clients.
func handleDetectDependencies(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input mcptypes.DetectDependenciesInput,
) (*mcp.CallToolResult, any, error) {
	log.Printf("Detecting dependencies: filepath=%s funcName=%s", input.FilePath, input.FuncName)

	// Validate required inputs
	if result := mcputil.ValidateRequiredWithPrefix("Dependency detection failed", map[string]string{
		"filePath": input.FilePath,
		"funcName": input.FuncName,
	}); result != nil {
		return result, nil, nil
	}

	// Call DetectDependencies to do the actual work
	output, err := DetectDependencies(input)
	if err != nil {
		return mcputil.ErrorResult(fmt.Sprintf("Dependency detection failed: %v", err)), nil, nil
	}

	// Return success with the dependencies
	result, artifact := mcputil.SuccessResultWithArtifact(
		fmt.Sprintf("Detected %d dependencies for %s in %s",
			len(output.Dependencies), input.FuncName, input.FilePath),
		output,
	)
	return result, artifact, nil
}

// ----------------------------------------------------- PRINT HELPERS ----------------------------------------------- //

func printSuccess() {
	_, _ = fmt.Fprintln(os.Stdout, "✅ Dependency detection completed successfully")
}

func printFailure(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "❌ Error detecting dependencies\n%s\n", err.Error())
}
