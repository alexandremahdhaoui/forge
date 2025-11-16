//go:build unit

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestHandleBuild tests the handleBuild function focusing on validation and error handling.
// Note: We test only the validation logic here, not the actual code generation since that
// requires real OpenAPI spec files and oapi-codegen execution.
func TestHandleBuild(t *testing.T) {
	tests := []struct {
		name      string
		input     mcptypes.BuildInput
		wantError bool
	}{
		{
			name: "missing name - should error",
			input: mcptypes.BuildInput{
				Name:   "", // Missing!
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"sourceFile": "./api/test.yaml",
					"client": map[string]interface{}{
						"enabled":     true,
						"packageName": "testclient",
					},
				},
			},
			wantError: true,
		},
		{
			name: "missing engine - should error",
			input: mcptypes.BuildInput{
				Name:   "test-api",
				Engine: "", // Missing!
				Spec: map[string]interface{}{
					"sourceFile": "./api/test.yaml",
					"client": map[string]interface{}{
						"enabled":     true,
						"packageName": "testclient",
					},
				},
			},
			wantError: true,
		},
		{
			name: "missing sourceFile and sourceDir - should error",
			input: mcptypes.BuildInput{
				Name:   "test-api",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					// Missing sourceFile AND sourceDir+name+version
					"client": map[string]interface{}{
						"enabled":     true,
						"packageName": "testclient",
					},
				},
			},
			wantError: true,
		},
		{
			name: "client enabled but missing packageName - should error",
			input: mcptypes.BuildInput{
				Name:   "test-api",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"sourceFile": "./api/test.yaml",
					"client": map[string]interface{}{
						"enabled": true,
						// Missing packageName!
					},
				},
			},
			wantError: true,
		},
		{
			name: "both client and server disabled - should error",
			input: mcptypes.BuildInput{
				Name:   "test-api",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"sourceFile": "./api/test.yaml",
					"client": map[string]interface{}{
						"enabled": false,
					},
					"server": map[string]interface{}{
						"enabled": false,
					},
				},
			},
			wantError: true,
		},
		{
			name: "nil spec - should error",
			input: mcptypes.BuildInput{
				Name:   "test-api",
				Engine: "go://go-gen-openapi",
				Spec:   nil,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := &mcp.CallToolRequest{}

			result, artifact, err := handleBuild(ctx, req, tt.input)
			// Check for unexpected errors from handler
			if err != nil {
				t.Fatalf("handleBuild returned error: %v", err)
			}

			// Check result error status
			if result.IsError != tt.wantError {
				t.Errorf("handleBuild IsError = %v, want %v", result.IsError, tt.wantError)
				if len(result.Content) > 0 {
					if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
						t.Logf("Error message: %s", textContent.Text)
					}
				}
			}

			// For error cases, artifact should be nil
			if tt.wantError && artifact != nil {
				t.Errorf("expected no artifact for error case, got %v", artifact)
			}
		})
	}
}

// TestHandleBuild_SuccessPath tests the success scenarios and validates artifact generation.
// These tests create temporary OpenAPI spec files and verify that handleBuild returns
// correct artifacts with all expected fields.
func TestHandleBuild_SuccessPath(t *testing.T) {
	// Setup: Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "go-gen-openapi-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a minimal valid OpenAPI spec file for testing
	specContent := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: OK
`
	specPath := filepath.Join(tempDir, "test.yaml")
	if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
		t.Fatalf("Failed to write spec file: %v", err)
	}

	// Create second spec file for templated pattern test (test.yaml.yaml)
	specPath2 := filepath.Join(tempDir, "test.yaml.yaml")
	if err := os.WriteFile(specPath2, []byte(specContent), 0o644); err != nil {
		t.Fatalf("Failed to write spec file: %v", err)
	}

	tests := []struct {
		name                string
		input               mcptypes.BuildInput
		expectedName        string
		expectedLocation    string
		expectedType        string
		shouldHaveVersion   bool
		shouldHaveTimestamp bool
	}{
		{
			name: "valid sourceFile pattern - should succeed",
			input: mcptypes.BuildInput{
				Name:   "test-api-v1",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"sourceFile":     specPath,
					"destinationDir": filepath.Join(tempDir, "generated"),
					"client": map[string]interface{}{
						"enabled":     true,
						"packageName": "testclient",
					},
				},
			},
			expectedName:        "test-api-v1",
			expectedLocation:    filepath.Join(tempDir, "generated"),
			expectedType:        "generated",
			shouldHaveVersion:   false, // NO Version field for generated code
			shouldHaveTimestamp: true,
		},
		{
			name: "valid sourceDir+name+version pattern - should succeed",
			input: mcptypes.BuildInput{
				Name:   "my-api",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"sourceDir":      tempDir,
					"name":           "test",
					"version":        "yaml", // Will create test.yaml.yaml pattern
					"destinationDir": filepath.Join(tempDir, "out"),
					"server": map[string]interface{}{
						"enabled":     true,
						"packageName": "testserver",
					},
				},
			},
			expectedName:        "my-api",
			expectedLocation:    filepath.Join(tempDir, "out"),
			expectedType:        "generated",
			shouldHaveVersion:   false,
			shouldHaveTimestamp: true,
		},
		{
			name: "default destinationDir - should succeed",
			input: mcptypes.BuildInput{
				Name:   "api-with-defaults",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"sourceFile": specPath,
					// No destinationDir - should default to ./pkg/generated
					"client": map[string]interface{}{
						"enabled":     true,
						"packageName": "defaultclient",
					},
				},
			},
			expectedName:        "api-with-defaults",
			expectedLocation:    "./pkg/generated", // Default value
			expectedType:        "generated",
			shouldHaveVersion:   false,
			shouldHaveTimestamp: true,
		},
		{
			name: "both client and server enabled - should succeed",
			input: mcptypes.BuildInput{
				Name:   "full-api",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"sourceFile":     specPath,
					"destinationDir": filepath.Join(tempDir, "fullgen"),
					"client": map[string]interface{}{
						"enabled":     true,
						"packageName": "client",
					},
					"server": map[string]interface{}{
						"enabled":     true,
						"packageName": "server",
					},
				},
			},
			expectedName:        "full-api",
			expectedLocation:    filepath.Join(tempDir, "fullgen"),
			expectedType:        "generated",
			shouldHaveVersion:   false,
			shouldHaveTimestamp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := &mcp.CallToolRequest{}

			beforeTime := time.Now().UTC()
			result, artifact, err := handleBuild(ctx, req, tt.input)
			afterTime := time.Now().UTC()

			// Should not return an error
			if err != nil {
				t.Fatalf("handleBuild returned error: %v", err)
			}

			// Result should NOT be an error
			if result.IsError {
				if len(result.Content) > 0 {
					if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
						t.Fatalf("handleBuild returned error result: %s", textContent.Text)
					}
				}
				t.Fatalf("handleBuild returned error result")
			}

			// Artifact should NOT be nil for success cases
			if artifact == nil {
				t.Fatal("expected artifact for success case, got nil")
			}

			// Verify artifact is of correct type
			artifactObj, ok := artifact.(forge.Artifact)
			if !ok {
				t.Fatalf("artifact is not forge.Artifact, got %T", artifact)
			}

			// CRITICAL: Verify artifact.Name == input.Name
			if artifactObj.Name != tt.expectedName {
				t.Errorf("artifact.Name = %q, want %q", artifactObj.Name, tt.expectedName)
			}

			// CRITICAL: Verify artifact.Location is correct (resolved destination directory)
			if artifactObj.Location != tt.expectedLocation {
				t.Errorf("artifact.Location = %q, want %q", artifactObj.Location, tt.expectedLocation)
			}

			// CRITICAL: Verify artifact.Type == "generated"
			if artifactObj.Type != tt.expectedType {
				t.Errorf("artifact.Type = %q, want %q", artifactObj.Type, tt.expectedType)
			}

			// CRITICAL: Verify NO Version field (should be empty)
			if tt.shouldHaveVersion {
				if artifactObj.Version == "" {
					t.Errorf("artifact.Version is empty, expected a value")
				}
			} else {
				if artifactObj.Version != "" {
					t.Errorf("artifact.Version = %q, want empty (generated code has no version)", artifactObj.Version)
				}
			}

			// CRITICAL: Verify Timestamp is set and valid
			if tt.shouldHaveTimestamp {
				if artifactObj.Timestamp == "" {
					t.Errorf("artifact.Timestamp is empty, expected a value")
				} else {
					// Parse timestamp to verify it's valid RFC3339
					parsedTime, err := time.Parse(time.RFC3339, artifactObj.Timestamp)
					if err != nil {
						t.Errorf("artifact.Timestamp %q is not valid RFC3339: %v", artifactObj.Timestamp, err)
					}
					// Verify timestamp is reasonable (within 5 seconds of the call)
					// Note: RFC3339 may truncate subseconds, so we allow some tolerance
					if parsedTime.Before(beforeTime.Add(-5*time.Second)) || parsedTime.After(afterTime.Add(5*time.Second)) {
						t.Errorf("artifact.Timestamp %v is not reasonably close to call time (before: %v, after: %v)", parsedTime, beforeTime, afterTime)
					}
				}
			}

			// Verify no unexpected fields are set
			expectedArtifact := forge.Artifact{
				Name:      tt.expectedName,
				Type:      tt.expectedType,
				Location:  tt.expectedLocation,
				Timestamp: artifactObj.Timestamp, // Use actual timestamp
				Version:   "",                    // Should be empty
			}

			// Compare all fields except Timestamp (already verified above)
			if artifactObj.Name != expectedArtifact.Name ||
				artifactObj.Type != expectedArtifact.Type ||
				artifactObj.Location != expectedArtifact.Location ||
				artifactObj.Version != expectedArtifact.Version {
				t.Errorf("artifact fields mismatch:\ngot:  %+v\nwant: %+v",
					artifactObj, expectedArtifact)
			}
		})
	}
}

// TestExtractOpenAPIConfigFromInput_Comprehensive tests the extractOpenAPIConfigFromInput function
// to verify it correctly resolves destination directories.
func TestExtractOpenAPIConfigFromInput_Comprehensive(t *testing.T) {
	tests := []struct {
		name                   string
		input                  mcptypes.BuildInput
		expectedDestinationDir string
		wantError              bool
	}{
		{
			name: "explicit destinationDir - should use it",
			input: mcptypes.BuildInput{
				Name:   "test",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"sourceFile":     "./api/test.yaml",
					"destinationDir": "/custom/path",
					"client": map[string]interface{}{
						"enabled":     true,
						"packageName": "client",
					},
				},
			},
			expectedDestinationDir: "/custom/path",
			wantError:              false,
		},
		{
			name: "no destinationDir - should default to ./pkg/generated",
			input: mcptypes.BuildInput{
				Name:   "test",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"sourceFile": "./api/test.yaml",
					// No destinationDir
					"client": map[string]interface{}{
						"enabled":     true,
						"packageName": "client",
					},
				},
			},
			expectedDestinationDir: "./pkg/generated",
			wantError:              false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := extractOpenAPIConfigFromInput(tt.input)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify Specs[0].DestinationDir is correctly set
			if len(config.Specs) == 0 {
				t.Fatal("config.Specs is empty")
			}

			if config.Specs[0].DestinationDir != tt.expectedDestinationDir {
				t.Errorf("config.Specs[0].DestinationDir = %q, want %q",
					config.Specs[0].DestinationDir, tt.expectedDestinationDir)
			}

			// Verify Defaults.DestinationDir matches (for backward compatibility)
			if config.Defaults.DestinationDir != tt.expectedDestinationDir {
				t.Errorf("config.Defaults.DestinationDir = %q, want %q",
					config.Defaults.DestinationDir, tt.expectedDestinationDir)
			}
		})
	}
}
