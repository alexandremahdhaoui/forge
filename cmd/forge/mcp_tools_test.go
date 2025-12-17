//go:build unit

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
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcputil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestAllMCPToolsReturnObjects verifies that ALL forge MCP tools return
// structured content as objects (not bare arrays or null), which is required
// by the MCP specification and Claude Code's MCP client.
func TestAllMCPToolsReturnObjects(t *testing.T) {
	// Setup: Create a temporary directory with a valid forge.yaml
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	// Create forge.yaml
	forgeYAML := `name: test-project
artifactStorePath: .ignore.artifact-store.yaml

build:
  - name: test-artifact
    src: ./test-src
    dest: ./test-dest
    engine: go://go-build

test:
  - name: unit
    runner: go://go-test
`
	if err := os.WriteFile(filepath.Join(tmpDir, "forge.yaml"), []byte(forgeYAML), 0o644); err != nil {
		t.Fatalf("Failed to write forge.yaml: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create artifact store
	storeYAML := `artifacts: []
testEnvironments: []
testReports: []
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".ignore.artifact-store.yaml"), []byte(storeYAML), 0o644); err != nil {
		t.Fatalf("Failed to write artifact store: %v", err)
	}

	tests := []struct {
		name        string
		toolName    string
		handler     func(context.Context, *mcp.CallToolRequest, interface{}) (*mcp.CallToolResult, any, error)
		input       interface{}
		expectError bool // Some tools are expected to fail in test environment
	}{
		{
			name:     "config-validate",
			toolName: "config-validate",
			handler: func(ctx context.Context, req *mcp.CallToolRequest, input interface{}) (*mcp.CallToolResult, any, error) {
				return handleConfigValidateTool(ctx, req, ConfigValidateInput{ConfigPath: "forge.yaml"})
			},
			input:       ConfigValidateInput{ConfigPath: "forge.yaml"},
			expectError: false,
		},
		{
			name:     "test-list",
			toolName: "test-list",
			handler: func(ctx context.Context, req *mcp.CallToolRequest, input interface{}) (*mcp.CallToolResult, any, error) {
				return handleTestListTool(ctx, req, TestListInput{Stage: "unit"})
			},
			input:       TestListInput{Stage: "unit"},
			expectError: false,
		},
		{
			name:     "docs-list",
			toolName: "docs-list",
			handler: func(ctx context.Context, req *mcp.CallToolRequest, input interface{}) (*mcp.CallToolResult, any, error) {
				return handleDocsListTool(ctx, req, DocsListInput{})
			},
			input:       DocsListInput{},
			expectError: false,
		},
		{
			name:     "docs-get",
			toolName: "docs-get",
			handler: func(ctx context.Context, req *mcp.CallToolRequest, input interface{}) (*mcp.CallToolResult, any, error) {
				return handleDocsGetTool(ctx, req, DocsGetInput{Name: "forge-usage"})
			},
			input:       DocsGetInput{Name: "forge-usage"},
			expectError: false,
		},
		// Note: build, test-create, test-get, test-delete, test-run, test-all require actual build/test infrastructure
		// and are covered by integration tests. Here we verify the wrapper types exist.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := &mcp.CallToolRequest{}

			result, artifact, err := tt.handler(ctx, req, tt.input)

			// Check for unexpected errors
			if err != nil && !tt.expectError {
				t.Fatalf("Unexpected error from %s: %v", tt.toolName, err)
			}

			// Verify result structure
			if result == nil {
				t.Fatalf("%s returned nil result", tt.toolName)
			}

			// Verify Content field exists
			if result.Content == nil || len(result.Content) == 0 {
				t.Errorf("%s returned empty Content field", tt.toolName)
			}

			// If artifact is returned, verify it's a valid object (not array or null)
			if artifact != nil {
				// Marshal to JSON
				jsonBytes, err := json.Marshal(artifact)
				if err != nil {
					t.Fatalf("%s: artifact failed to marshal to JSON: %v", tt.toolName, err)
				}

				// Verify it's a JSON object (starts with '{'), not array (starts with '[')
				if len(jsonBytes) > 0 {
					firstChar := jsonBytes[0]
					if firstChar == '[' {
						t.Errorf("%s: structured content is a bare array (starts with '['), must be object (start with '{')\nJSON: %s",
							tt.toolName, string(jsonBytes))
					} else if firstChar != '{' && string(jsonBytes) != "null" {
						t.Errorf("%s: structured content is not a valid JSON object or null\nJSON: %s",
							tt.toolName, string(jsonBytes))
					}
				}

				// Additional validation: try to unmarshal as map to verify it's an object
				if string(jsonBytes) != "null" {
					var objMap map[string]interface{}
					if err := json.Unmarshal(jsonBytes, &objMap); err != nil {
						t.Errorf("%s: structured content cannot be unmarshaled as object: %v\nJSON: %s",
							tt.toolName, err, string(jsonBytes))
					}
				}
			}
		})
	}
}

// TestMCPWrapperTypes verifies that all MCP wrapper types are valid JSON objects
func TestMCPWrapperTypes(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "BuildResult",
			data: BuildResult{
				Artifacts: []forge.Artifact{
					{Name: "test", Type: "binary", Location: "./test"},
				},
				Summary: "Built 1 artifact",
			},
		},
		{
			name: "TestListResult",
			data: TestListResult{
				Reports: []forge.TestReport{
					{ID: "test-1", Stage: "unit", Status: "passed"},
				},
				Stage: "unit",
				Count: 1,
			},
		},
		{
			name: "TestAllResult",
			data: TestAllResult{
				BuildArtifacts: []forge.Artifact{
					{Name: "test", Type: "binary", Location: "./test"},
				},
				TestReports: []forge.TestReport{
					{ID: "test-1", Stage: "unit", Status: "passed"},
				},
				Summary: "1 artifact built, 1 test passed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			jsonBytes, err := json.Marshal(tt.data)
			if err != nil {
				t.Fatalf("Failed to marshal %s to JSON: %v", tt.name, err)
			}

			// Verify it's a JSON object (not array)
			if len(jsonBytes) == 0 {
				t.Fatalf("%s marshaled to empty JSON", tt.name)
			}

			if jsonBytes[0] != '{' {
				t.Errorf("%s is not a JSON object (doesn't start with '{'): %s", tt.name, string(jsonBytes))
			}

			// Verify it can be unmarshaled as an object
			var objMap map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &objMap); err != nil {
				t.Errorf("Failed to unmarshal %s as object: %v", tt.name, err)
			}

			t.Logf("%s JSON: %s", tt.name, string(jsonBytes))
		})
	}
}

// TestBatchResultWrapper verifies the BatchResult wrapper from mcputil
func TestBatchResultWrapper(t *testing.T) {
	// Import the BatchResult type via reflection since it's in mcputil
	// This test verifies that FormatBatchResult returns a proper object

	artifacts := []any{"artifact1", "artifact2"}
	errorMsgs := []string{}

	result, returnedArtifact := mcputil.FormatBatchResult("binaries", artifacts, errorMsgs)

	// Verify result
	if result == nil {
		t.Fatal("FormatBatchResult returned nil result")
	}

	if result.IsError {
		t.Error("Expected IsError to be false for success case")
	}

	// Verify artifact is not nil
	if returnedArtifact == nil {
		t.Fatal("FormatBatchResult returned nil artifact")
	}

	// Marshal to JSON and verify it's an object
	jsonBytes, err := json.Marshal(returnedArtifact)
	if err != nil {
		t.Fatalf("Failed to marshal BatchResult: %v", err)
	}

	if jsonBytes[0] != '{' {
		t.Errorf("BatchResult is not a JSON object (doesn't start with '{'): %s", string(jsonBytes))
	}

	// Verify it has expected fields
	var objMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &objMap); err != nil {
		t.Fatalf("Failed to unmarshal BatchResult: %v", err)
	}

	// Check for expected fields
	if _, ok := objMap["artifacts"]; !ok {
		t.Error("BatchResult missing 'artifacts' field")
	}
	if _, ok := objMap["summary"]; !ok {
		t.Error("BatchResult missing 'summary' field")
	}
	if _, ok := objMap["count"]; !ok {
		t.Error("BatchResult missing 'count' field")
	}

	t.Logf("BatchResult JSON: %s", string(jsonBytes))
}
