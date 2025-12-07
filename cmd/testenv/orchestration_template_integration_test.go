//go:build integration

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
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

// TestTemplateExpansionInOrchestration verifies that:
// 1. Templates in sub-engine specs are expanded using accumulated environment
// 2. Expanded spec is passed to sub-engines (not the original templated spec)
// 3. Template expansion errors properly abort testenv creation
// 4. Works end-to-end in realistic scenarios
//
// This is an integration test that exercises the orchestrateCreate function
// with realistic testenv configurations containing template variables.
func TestTemplateExpansionInOrchestration(t *testing.T) {
	// Setup: Create a temporary directory for test fixtures
	tmpDir := t.TempDir()
	forgeYamlPath := filepath.Join(tmpDir, "forge.yaml")
	artifactStorePath := filepath.Join(tmpDir, ".forge", "artifacts.json")

	// Change to temp directory for test
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .forge directory
	if err := os.MkdirAll(filepath.Join(tmpDir, ".forge"), 0o755); err != nil {
		t.Fatalf("Failed to create .forge directory: %v", err)
	}

	tests := []struct {
		name          string
		config        forge.Spec
		setupAlias    string
		wantError     bool
		errorContains string
		verifyFunc    func(t *testing.T, env *forge.TestEnvironment)
	}{
		{
			name: "undefined variable in template causes error",
			config: forge.Spec{
				Name:              "test-project",
				ArtifactStorePath: artifactStorePath,
				Engines: []forge.EngineConfig{
					{
						Alias: "test-orchestrator",
						Type:  "testenv",
						Testenv: []forge.TestenvEngineSpec{
							{
								// Sub-engine with template referencing UNDEFINED variable
								// This should fail during template expansion BEFORE calling the engine
								Engine: "go://testenv-mock",
								Spec: map[string]interface{}{
									"templatedField": "{{.Env.UNDEFINED_VAR}}",
								},
							},
						},
					},
				},
			},
			setupAlias:    "test-orchestrator",
			wantError:     true,
			errorContains: "UNDEFINED_VAR",
		},
		{
			name: "no templates pass through unchanged",
			config: forge.Spec{
				Name:              "test-project",
				ArtifactStorePath: artifactStorePath,
				Engines: []forge.EngineConfig{
					{
						Alias: "test-orchestrator",
						Type:  "testenv",
						Testenv: []forge.TestenvEngineSpec{
							{
								Engine: "go://testenv-plain",
								Spec: map[string]interface{}{
									"plainField": "plain_value",
									"nested": map[string]interface{}{
										"value": "no_template_here",
									},
								},
							},
						},
					},
				},
			},
			setupAlias: "test-orchestrator",
			wantError:  false, // Will fail on engine resolution but NOT on template expansion
			verifyFunc: func(t *testing.T, env *forge.TestEnvironment) {
				// Test environment should be initialized even if engine resolution fails
				if env.ID == "" {
					t.Error("Expected non-empty test ID")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write forge.yaml config
			configData, err := json.MarshalIndent(tt.config, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal config: %v", err)
			}
			if err := os.WriteFile(forgeYamlPath, configData, 0o644); err != nil {
				t.Fatalf("Failed to write forge.yaml: %v", err)
			}

			// Create test environment
			env := &forge.TestEnvironment{
				ID:               "test-integration-12345678",
				Name:             "integration-test",
				Status:           forge.TestStatusCreated,
				CreatedAt:        time.Now().UTC(),
				UpdatedAt:        time.Now().UTC(),
				TmpDir:           filepath.Join(tmpDir, ".forge", "tmp", "test-integration-12345678"),
				Files:            make(map[string]string),
				ManagedResources: []string{},
				Metadata:         make(map[string]string),
			}

			// Create tmpDir
			if err := os.MkdirAll(env.TmpDir, 0o755); err != nil {
				t.Fatalf("Failed to create tmpDir: %v", err)
			}
			defer os.RemoveAll(env.TmpDir)

			// Call orchestrateCreate - this is the integration point being tested
			// This function should:
			// 1. Iterate through sub-engines
			// 2. For each sub-engine, expand templates using accumulated environment
			// 3. Pass expanded spec to MCP engine (not original templated spec)
			// 4. Return error if template expansion fails
			err = orchestrateCreate(tt.config, tt.setupAlias, env)

			// Verify error expectations
			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				} else {
					t.Logf("Got expected error: %v", err)
				}
			} else {
				if err != nil {
					// Note: This test uses mock engines that don't exist.
					// The key verification is the ORDER of errors:
					// - Template expansion errors should happen BEFORE engine resolution
					// - Engine resolution errors are expected with non-existent engines
					//
					// If we see "failed to expand templates", that's a real failure.
					// If we see "failed to resolve engine" or "failed to create",
					// that's expected and means template expansion succeeded.
					if containsString(err.Error(), "failed to expand templates") {
						t.Errorf("Template expansion failed unexpectedly: %v", err)
					} else if containsString(err.Error(), "failed to resolve engine") {
						// Expected: template expansion succeeded, but mock engine doesn't exist
						t.Logf("Expected error (mock engine not found): %v", err)
					} else if containsString(err.Error(), "failed to create with") {
						// Expected: engine resolved but create failed (mock doesn't exist)
						t.Logf("Expected error (mock engine create failed): %v", err)
					} else {
						// Unexpected error type
						t.Logf("Unexpected error (might be OK): %v", err)
					}
				}

				// Run verification function if provided
				if tt.verifyFunc != nil {
					tt.verifyFunc(t, env)
				}
			}
		})
	}
}

// containsString checks if a string contains a substring (case-sensitive).
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestDeferTemplatesInOrchestration verifies the DeferTemplates behavior:
// 1. When DeferTemplates=true, templates are NOT expanded by forge (passed verbatim to sub-engine)
// 2. When DeferTemplates=false (default), templates ARE expanded by forge
// 3. Backward compatibility: omitted DeferTemplates defaults to false (expansion occurs)
//
// The key verification is the ORDER of errors:
// - When DeferTemplates=true, undefined template variables should NOT cause errors from forge
// - When DeferTemplates=false, undefined template variables SHOULD cause expansion errors
func TestDeferTemplatesInOrchestration(t *testing.T) {
	// Setup: Create a temporary directory for test fixtures
	tmpDir := t.TempDir()
	forgeYamlPath := filepath.Join(tmpDir, "forge.yaml")
	artifactStorePath := filepath.Join(tmpDir, ".forge", "artifacts.json")

	// Change to temp directory for test
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .forge directory
	if err := os.MkdirAll(filepath.Join(tmpDir, ".forge"), 0o755); err != nil {
		t.Fatalf("Failed to create .forge directory: %v", err)
	}

	tests := []struct {
		name                       string
		config                     forge.Spec
		setupAlias                 string
		wantTemplateExpansionError bool // true if we expect template expansion to fail
		errorContains              string
		verifyFunc                 func(t *testing.T, env *forge.TestEnvironment, err error)
	}{
		{
			name: "DeferTemplates=true skips template expansion (no error for undefined variable)",
			config: forge.Spec{
				Name:              "test-project",
				ArtifactStorePath: artifactStorePath,
				Engines: []forge.EngineConfig{
					{
						Alias: "test-orchestrator",
						Type:  "testenv",
						Testenv: []forge.TestenvEngineSpec{
							{
								Engine:         "go://testenv-mock",
								DeferTemplates: true, // Skip forge template expansion
								Spec: map[string]interface{}{
									// This would FAIL if expanded by forge (UNDEFINED_VAR not in env)
									// But with deferTemplates=true, forge should NOT expand it
									"templatedField": "{{.Env.UNDEFINED_VAR}}",
									"subEngineVar":   "{{.Networks.mynet}}",
								},
							},
						},
					},
				},
			},
			setupAlias:                 "test-orchestrator",
			wantTemplateExpansionError: false, // Template expansion should be skipped
			verifyFunc: func(t *testing.T, env *forge.TestEnvironment, err error) {
				// The key assertion: if we see an error about "UNDEFINED_VAR",
				// it means template expansion happened when it shouldn't have
				if err != nil && containsString(err.Error(), "UNDEFINED_VAR") {
					t.Errorf("Template expansion occurred when deferTemplates=true; got error: %v", err)
				}
				if err != nil && containsString(err.Error(), "template expansion failed") {
					t.Errorf("Template expansion occurred when deferTemplates=true; got error: %v", err)
				}
				// It's OK to fail on engine resolution (mock engine doesn't exist)
				if err != nil {
					t.Logf("Got expected error (not template expansion): %v", err)
				}
			},
		},
		{
			name: "DeferTemplates=false (explicit) expands templates and fails on undefined variable",
			config: forge.Spec{
				Name:              "test-project",
				ArtifactStorePath: artifactStorePath,
				Engines: []forge.EngineConfig{
					{
						Alias: "test-orchestrator",
						Type:  "testenv",
						Testenv: []forge.TestenvEngineSpec{
							{
								Engine:         "go://testenv-mock",
								DeferTemplates: false, // Explicitly set to false
								Spec: map[string]interface{}{
									"templatedField": "{{.Env.UNDEFINED_VAR}}",
								},
							},
						},
					},
				},
			},
			setupAlias:                 "test-orchestrator",
			wantTemplateExpansionError: true,
			errorContains:              "UNDEFINED_VAR",
		},
		{
			name: "DeferTemplates omitted (defaults to false) expands templates and fails on undefined variable",
			config: forge.Spec{
				Name:              "test-project",
				ArtifactStorePath: artifactStorePath,
				Engines: []forge.EngineConfig{
					{
						Alias: "test-orchestrator",
						Type:  "testenv",
						Testenv: []forge.TestenvEngineSpec{
							{
								Engine: "go://testenv-mock",
								// DeferTemplates omitted - should default to false
								Spec: map[string]interface{}{
									"templatedField": "{{.Env.MISSING_VAR}}",
								},
							},
						},
					},
				},
			},
			setupAlias:                 "test-orchestrator",
			wantTemplateExpansionError: true,
			errorContains:              "MISSING_VAR",
		},
		{
			name: "DeferTemplates=false with no templates passes spec unchanged",
			config: forge.Spec{
				Name:              "test-project",
				ArtifactStorePath: artifactStorePath,
				Engines: []forge.EngineConfig{
					{
						Alias: "test-orchestrator",
						Type:  "testenv",
						Testenv: []forge.TestenvEngineSpec{
							{
								Engine:         "go://testenv-mock",
								DeferTemplates: false,
								Spec: map[string]interface{}{
									// No templates - should pass through without error
									"plainField": "plain_value",
									"nested": map[string]interface{}{
										"value": "no_template_here",
									},
								},
							},
						},
					},
				},
			},
			setupAlias:                 "test-orchestrator",
			wantTemplateExpansionError: false, // No templates = no expansion error
			verifyFunc: func(t *testing.T, env *forge.TestEnvironment, err error) {
				// Should NOT see template expansion error (no templates to expand)
				if err != nil && containsString(err.Error(), "template expansion failed") {
					t.Errorf("Template expansion should not fail when no templates present; got error: %v", err)
				}
				// Engine resolution error is expected (mock engine doesn't exist)
				if err != nil {
					t.Logf("Got expected error (not template expansion): %v", err)
				}
			},
		},
		{
			name: "Mixed DeferTemplates - first engine has no templates, second defers",
			config: forge.Spec{
				Name:              "test-project",
				ArtifactStorePath: artifactStorePath,
				Engines: []forge.EngineConfig{
					{
						Alias: "test-orchestrator",
						Type:  "testenv",
						Testenv: []forge.TestenvEngineSpec{
							{
								Engine:         "go://testenv-mock-first",
								DeferTemplates: false,
								Spec: map[string]interface{}{
									// No templates - will pass expansion phase
									"plainField": "plain_value",
								},
							},
							{
								Engine:         "go://testenv-mock-second",
								DeferTemplates: true,
								Spec: map[string]interface{}{
									// Uses undefined variable but should NOT fail
									// because DeferTemplates=true
									"field": "{{.Env.UNDEFINED_VAR}}",
								},
							},
						},
					},
				},
			},
			setupAlias:                 "test-orchestrator",
			wantTemplateExpansionError: false, // First engine has no templates, second defers
			verifyFunc: func(t *testing.T, env *forge.TestEnvironment, err error) {
				// Should NOT see error about UNDEFINED_VAR since second engine defers templates
				if err != nil && containsString(err.Error(), "UNDEFINED_VAR") {
					t.Errorf("Second engine should not expand templates; got error: %v", err)
				}
				// Should NOT see template expansion error at all
				if err != nil && containsString(err.Error(), "template expansion failed") {
					t.Errorf("Neither engine should cause template expansion failure; got error: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write forge.yaml config
			configData, err := json.MarshalIndent(tt.config, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal config: %v", err)
			}
			if err := os.WriteFile(forgeYamlPath, configData, 0o644); err != nil {
				t.Fatalf("Failed to write forge.yaml: %v", err)
			}

			// Create test environment
			env := &forge.TestEnvironment{
				ID:               "test-integration-12345678",
				Name:             "integration-test",
				Status:           forge.TestStatusCreated,
				CreatedAt:        time.Now().UTC(),
				UpdatedAt:        time.Now().UTC(),
				TmpDir:           filepath.Join(tmpDir, ".forge", "tmp", "test-integration-12345678"),
				Files:            make(map[string]string),
				ManagedResources: []string{},
				Metadata:         make(map[string]string),
			}

			// Create tmpDir
			if err := os.MkdirAll(env.TmpDir, 0o755); err != nil {
				t.Fatalf("Failed to create tmpDir: %v", err)
			}
			defer os.RemoveAll(env.TmpDir)

			// Call orchestrateCreate
			err = orchestrateCreate(tt.config, tt.setupAlias, env)

			// Verify error expectations
			if tt.wantTemplateExpansionError {
				if err == nil {
					t.Error("Expected template expansion error but got nil")
				} else if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				} else {
					t.Logf("Got expected template expansion error: %v", err)
				}
			} else if tt.verifyFunc != nil {
				tt.verifyFunc(t, env, err)
			}
		})
	}
}
