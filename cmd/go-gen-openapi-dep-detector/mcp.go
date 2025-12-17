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
		Description: "Detect dependencies for OpenAPI code generation",
	}, handleDetectDependencies)
}

// handleDetectDependencies handles the "detectDependencies" tool call from MCP clients.
func handleDetectDependencies(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input mcptypes.DetectOpenAPIDependenciesInput,
) (*mcp.CallToolResult, any, error) {
	log.Printf("Detecting OpenAPI dependencies: specSources=%v", input.SpecSources)

	// Call DetectOpenAPIDependencies to do the actual work
	output, err := DetectOpenAPIDependencies(input)
	if err != nil {
		return mcputil.ErrorResult(fmt.Sprintf("OpenAPI dependency detection failed: %v", err)), nil, nil
	}

	// Return success with the dependencies
	result, artifact := mcputil.SuccessResultWithArtifact(
		fmt.Sprintf("Detected %d dependencies for OpenAPI generation", len(output.Dependencies)),
		output,
	)
	return result, artifact, nil
}
