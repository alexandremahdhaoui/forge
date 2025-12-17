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

	"github.com/alexandremahdhaoui/forge/pkg/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/alexandremahdhaoui/forge/pkg/mcputil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// init assigns the registerDetectDependenciesTool function.
func init() {
	registerDetectDependenciesTool = doRegisterDetectDependenciesTool
}

// doRegisterDetectDependenciesTool registers the detectDependencies MCP tool.
func doRegisterDetectDependenciesTool(server *mcpserver.Server) {
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "detectDependencies",
		Description: "Detect all dependencies (local files and external packages) for a Go function",
	}, handleDetectDependencies)
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
