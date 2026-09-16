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

	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"github.com/alexandremahdhaoui/forge/pkg/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/mcputil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateInput represents the input for the create tool.
type CreateInput struct {
	Stage string `json:"stage"`
}

// DeleteInput represents the input for the delete tool.
type DeleteInput struct {
	TestID string `json:"testID"`
}

// runMCPServer starts the MCP server.
func runMCPServer() error {
	server := mcpserver.New("testenv", Version)
	commands := newTestenvCommands()

	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "create",
		Description: "Create a test environment for a given stage",
	}, commands.handleCreateTool)

	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "delete",
		Description: "Delete a test environment by ID",
	}, commands.handleDeleteTool)

	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "config-validate",
		Description: "Validate testenv configuration and recursively validate subengines",
	}, commands.handleConfigValidate)

	if err := enginedocs.RegisterDocsTools(server, *docsConfig); err != nil {
		return err
	}

	return server.RunDefault()
}

func (c *testenvCommands) handleCreateTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input CreateInput,
) (*mcp.CallToolResult, any, error) {
	log.Printf("Creating test environment: stage=%s", input.Stage)

	if result := mcputil.ValidateRequiredWithPrefix("Create failed", map[string]string{
		"stage": input.Stage,
	}); result != nil {
		return result, nil, nil
	}

	testID, err := c.cmdCreate(input.Stage)
	if err != nil {
		return mcputil.ErrorResult(fmt.Sprintf("Create failed: %v", err)), nil, nil
	}

	result, returnedArtifact := mcputil.SuccessResultWithArtifact(
		fmt.Sprintf("Created test environment: %s", testID),
		map[string]string{"testID": testID},
	)
	return result, returnedArtifact, nil
}

func (c *testenvCommands) handleDeleteTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input DeleteInput,
) (*mcp.CallToolResult, any, error) {
	log.Printf("Deleting test environment: testID=%s", input.TestID)

	if result := mcputil.ValidateRequiredWithPrefix("Delete failed", map[string]string{
		"testID": input.TestID,
	}); result != nil {
		return result, nil, nil
	}

	if err := c.cmdDelete(input.TestID); err != nil {
		return mcputil.ErrorResult(fmt.Sprintf("Delete failed: %v", err)), nil, nil
	}

	return mcputil.SuccessResult(fmt.Sprintf("Deleted test environment: %s", input.TestID)), nil, nil
}
