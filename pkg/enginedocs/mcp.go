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

package enginedocs

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/alexandremahdhaoui/forge/pkg/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/mcputil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DocsListInput represents the input parameters for the docs-list tool.
type DocsListInput struct {
	// No parameters - lists all docs
}

// DocsGetInput represents the input parameters for the docs-get tool.
type DocsGetInput struct {
	Name string `json:"name"`
}

// DocsValidateInput represents the input parameters for the docs-validate tool.
type DocsValidateInput struct {
	// No parameters - validates all docs
}

// DocsListResult represents the result of listing documentation.
type DocsListResult struct {
	Docs   []DocEntry `json:"docs"`
	Engine string     `json:"engine"`
	Count  int        `json:"count"`
}

// DocsValidateResult represents the result of documentation validation.
type DocsValidateResult struct {
	Engine string   `json:"engine"`
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// RegisterDocsTools registers the documentation MCP tools with the server.
// This registers three tools:
//  1. docs-list - lists all available documentation entries
//  2. docs-get - retrieves specific documentation content by name
//  3. docs-validate - validates documentation completeness (NEW functionality)
func RegisterDocsTools(server *mcpserver.Server, cfg Config) error {
	// Register docs-list tool
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "docs-list",
		Description: fmt.Sprintf("List all available documentation for %s", cfg.EngineName),
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DocsListInput) (*mcp.CallToolResult, any, error) {
		return handleDocsListTool(ctx, req, input, cfg)
	})

	// Register docs-get tool
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "docs-get",
		Description: fmt.Sprintf("Get a specific documentation by name for %s", cfg.EngineName),
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DocsGetInput) (*mcp.CallToolResult, any, error) {
		return handleDocsGetTool(ctx, req, input, cfg)
	})

	// Register docs-validate tool (NEW functionality)
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "docs-validate",
		Description: fmt.Sprintf("Validate documentation completeness for %s", cfg.EngineName),
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DocsValidateInput) (*mcp.CallToolResult, any, error) {
		return handleDocsValidateTool(ctx, req, input, cfg)
	})

	return nil
}

// handleDocsListTool handles the "docs-list" tool call from MCP clients.
func handleDocsListTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input DocsListInput,
	cfg Config,
) (*mcp.CallToolResult, any, error) {
	log.Printf("Listing documentation for %s", cfg.EngineName)

	docs, err := DocsList(cfg)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to list documentation: %v", err)},
			},
			IsError: true,
		}, nil, nil
	}

	result := DocsListResult{
		Docs:   docs,
		Engine: cfg.EngineName,
		Count:  len(docs),
	}

	mcpResult, artifact := mcputil.SuccessResultWithArtifact(
		fmt.Sprintf("Found %d documentation entries for %s", len(docs), cfg.EngineName),
		result,
	)
	return mcpResult, artifact, nil
}

// handleDocsGetTool handles the "docs-get" tool call from MCP clients.
func handleDocsGetTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input DocsGetInput,
	cfg Config,
) (*mcp.CallToolResult, any, error) {
	log.Printf("Getting documentation: %s for %s", input.Name, cfg.EngineName)

	content, err := DocsGet(cfg, input.Name)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to get documentation: %v", err)},
			},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: content},
		},
	}, nil, nil
}

// handleDocsValidateTool handles the "docs-validate" tool call from MCP clients.
func handleDocsValidateTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input DocsValidateInput,
	cfg Config,
) (*mcp.CallToolResult, any, error) {
	log.Printf("Validating documentation for %s", cfg.EngineName)

	errs := Validate(cfg)

	result := DocsValidateResult{
		Engine: cfg.EngineName,
		Valid:  len(errs) == 0,
		Errors: make([]string, len(errs)),
	}

	for i, err := range errs {
		result.Errors[i] = err.Error()
	}

	if len(errs) > 0 {
		mcpResult, artifact := mcputil.ErrorResultWithArtifact(
			fmt.Sprintf("Validation failed for %s: %d error(s) found", cfg.EngineName, len(errs)),
			result,
		)
		// Also include detailed error messages in content
		mcpResult.Content = append(mcpResult.Content, &mcp.TextContent{
			Text: "Errors:\n- " + strings.Join(result.Errors, "\n- "),
		})
		return mcpResult, artifact, nil
	}

	mcpResult, artifact := mcputil.SuccessResultWithArtifact(
		fmt.Sprintf("Validation passed for %s: documentation is complete", cfg.EngineName),
		result,
	)
	return mcpResult, artifact, nil
}
