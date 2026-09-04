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
	"os"
	"os/exec"
	"time"

	"github.com/alexandremahdhaoui/forge/internal/engineresolver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const mcpToolCallTimeout = 15 * time.Minute

type engineInvocation struct {
	command string
	args    []string
	dir     string
}

func callMCPEngine(engine engineInvocation, toolName string, params interface{}) (interface{}, error) {
	cmdArgs := append(append([]string{}, engine.args...), "--mcp")
	cmd := exec.Command(engine.command, cmdArgs...)
	cmd.Env = os.Environ()
	cmd.Dir = engine.dir
	cmd.Stderr = os.Stderr

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "testenv-client",
		Version: "v1.0.0",
	}, nil)

	transport := &mcp.CommandTransport{
		Command: cmd,
	}

	ctx := context.Background()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server %s %v: %w", engine.command, engine.args, err)
	}
	defer func() { _ = session.Close() }()

	arguments, ok := params.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("calling %s: params must be a map[string]any, got %T", toolName, params)
	}

	toolCtx, cancel := context.WithTimeout(context.Background(), mcpToolCallTimeout)
	defer cancel()

	result, err := session.CallTool(toolCtx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: arguments,
	})
	if err != nil {
		if toolCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("MCP tool call timed out after %s: %w", mcpToolCallTimeout, err)
		}
		return nil, fmt.Errorf("MCP tool call failed: %w", err)
	}

	if result.IsError {
		errMsg := "unknown error"
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
				errMsg = textContent.Text
			}
		}
		return nil, fmt.Errorf("operation failed: %s", errMsg)
	}

	if result.StructuredContent != nil {
		return result.StructuredContent, nil
	}

	return nil, nil
}

func resolveEngineURI(engineURI string) (engineInvocation, error) {
	inv, err := engineresolver.ResolveForgeURI(engineURI, getVersion())
	if err != nil {
		return engineInvocation{}, fmt.Errorf("resolving engine %s: %w", engineURI, err)
	}

	return engineInvocation{command: inv.Command, args: inv.Args, dir: inv.Dir}, nil
}
