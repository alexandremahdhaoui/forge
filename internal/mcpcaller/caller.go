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

// Package mcpcaller provides shared MCP calling functionality.
// It provides reusable MCP calling for parallel-builder and parallel-test-runner,
// encapsulating process spawning and MCP protocol.
package mcpcaller

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/alexandremahdhaoui/forge/internal/engineresolver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPCaller is a function type for calling MCP engines.
// Matches signature from internal/orchestrate/orchestrate.go:24
type MCPCaller func(command string, args []string, toolName string, params interface{}) (interface{}, error)

// EngineResolver is a function type for resolving engine URIs to command and args.
// Matches signature from internal/orchestrate/orchestrate.go:29
type EngineResolver func(engineURI string) (command string, args []string, err error)

// Caller provides MCP calling and engine resolution functionality.
type Caller struct {
	forgeVersion string
}

// NewCaller creates a new Caller with the specified forge version.
func NewCaller(forgeVersion string) *Caller {
	return &Caller{forgeVersion: forgeVersion}
}

// CallMCP implements MCPCaller - spawns MCP server process and calls tool.
// This is adapted from cmd/forge/mcp_client.go:callMCPEngine (lines 31-102)
func (c *Caller) CallMCP(command string, args []string, toolName string, params interface{}) (interface{}, error) {
	// Create command to spawn MCP server
	// Append --mcp flag to the args
	cmdArgs := append(args, "--mcp")
	cmd := exec.Command(command, cmdArgs...)

	// Inherit environment variables from parent process
	cmd.Env = os.Environ()

	// Forward stderr from the MCP server to show build logs
	cmd.Stderr = os.Stderr

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "forge-parallel",
		Version: c.forgeVersion,
	}, nil)

	// Create command transport
	transport := &mcp.CommandTransport{
		Command: cmd,
	}

	// Connect to the MCP server
	ctx := context.Background()
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server %s %v: %w", command, args, err)
	}
	defer func() { _ = session.Close() }()

	// Convert params to map[string]any for CallTool
	var arguments map[string]any
	switch p := params.(type) {
	case map[string]any:
		arguments = p
	default:
		arguments = params.(map[string]any)
	}

	// Call the tool
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: arguments,
	})
	if err != nil {
		return nil, fmt.Errorf("MCP tool call failed: %w", err)
	}

	// Check if result indicates an error
	if result.IsError {
		errMsg := "unknown error"
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
				errMsg = textContent.Text
			}
		}
		return nil, fmt.Errorf("tool call failed: %s", errMsg)
	}

	// Return the structured content if available
	if result.StructuredContent != nil {
		return result.StructuredContent, nil
	}

	return nil, nil
}

// ResolveEngine implements EngineResolver - parses engine URI to command and args.
// Note: This only handles go:// URIs. alias:// URIs are NOT supported here
// because alias resolution requires forge.yaml spec access.
func (c *Caller) ResolveEngine(engineURI string) (string, []string, error) {
	engineType, command, args, err := engineresolver.ParseEngineURI(engineURI, c.forgeVersion)
	if err != nil {
		return "", nil, err
	}

	if engineType == engineresolver.EngineTypeAlias {
		return "", nil, fmt.Errorf("alias:// URIs not supported in parallel engines; use resolved go:// URIs")
	}

	return command, args, nil
}

// GetMCPCaller returns a MCPCaller function that can be used with orchestrators.
func (c *Caller) GetMCPCaller() MCPCaller {
	return c.CallMCP
}

// GetEngineResolver returns an EngineResolver function that can be used with orchestrators.
func (c *Caller) GetEngineResolver() EngineResolver {
	return c.ResolveEngine
}
