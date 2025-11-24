//go:build unit

package main

import (
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// TestEnvPropagationFiltering tests the EnvPropagation filtering logic
func TestEnvPropagationFiltering(t *testing.T) {
	tests := []struct {
		name              string
		testenvEnv        map[string]string
		envPropagation    *forge.EnvPropagation
		wantEnvPresent    map[string]string
		wantEnvNotPresent []string
	}{
		{
			name: "No filtering - all vars propagated",
			testenvEnv: map[string]string{
				"KUBECONFIG":   "/tmp/kubeconfig",
				"REGISTRY_URL": "localhost:5000",
				"NAMESPACE":    "test-ns",
			},
			envPropagation: nil, // No filtering config
			wantEnvPresent: map[string]string{
				"KUBECONFIG":   "/tmp/kubeconfig",
				"REGISTRY_URL": "localhost:5000",
				"NAMESPACE":    "test-ns",
			},
			wantEnvNotPresent: []string{},
		},
		{
			name: "Whitelist filtering - only whitelisted vars propagated",
			testenvEnv: map[string]string{
				"KUBECONFIG":   "/tmp/kubeconfig",
				"REGISTRY_URL": "localhost:5000",
				"NAMESPACE":    "test-ns",
				"OTHER_VAR":    "other-value",
			},
			envPropagation: &forge.EnvPropagation{
				Whitelist: []string{"KUBECONFIG", "REGISTRY_URL"},
			},
			wantEnvPresent: map[string]string{
				"KUBECONFIG":   "/tmp/kubeconfig",
				"REGISTRY_URL": "localhost:5000",
			},
			wantEnvNotPresent: []string{"NAMESPACE", "OTHER_VAR"},
		},
		{
			name: "Blacklist filtering - blacklisted vars excluded",
			testenvEnv: map[string]string{
				"KUBECONFIG":   "/tmp/kubeconfig",
				"REGISTRY_URL": "localhost:5000",
				"NAMESPACE":    "test-ns",
				"SECRET_TOKEN": "secret-value",
			},
			envPropagation: &forge.EnvPropagation{
				Blacklist: []string{"SECRET_TOKEN", "NAMESPACE"},
			},
			wantEnvPresent: map[string]string{
				"KUBECONFIG":   "/tmp/kubeconfig",
				"REGISTRY_URL": "localhost:5000",
			},
			wantEnvNotPresent: []string{"SECRET_TOKEN", "NAMESPACE"},
		},
		{
			name: "Disabled propagation - no vars propagated",
			testenvEnv: map[string]string{
				"KUBECONFIG":   "/tmp/kubeconfig",
				"REGISTRY_URL": "localhost:5000",
				"NAMESPACE":    "test-ns",
			},
			envPropagation: &forge.EnvPropagation{
				Disabled: true,
			},
			wantEnvPresent:    map[string]string{},
			wantEnvNotPresent: []string{"KUBECONFIG", "REGISTRY_URL", "NAMESPACE"},
		},
		{
			name: "Whitelist with non-existent vars - only existing whitelisted vars propagated",
			testenvEnv: map[string]string{
				"KUBECONFIG": "/tmp/kubeconfig",
				"NAMESPACE":  "test-ns",
			},
			envPropagation: &forge.EnvPropagation{
				Whitelist: []string{"KUBECONFIG", "REGISTRY_URL", "NON_EXISTENT"},
			},
			wantEnvPresent: map[string]string{
				"KUBECONFIG": "/tmp/kubeconfig",
			},
			wantEnvNotPresent: []string{"NAMESPACE", "REGISTRY_URL", "NON_EXISTENT"},
		},
		{
			name: "Empty whitelist - no filtering (all vars propagated)",
			testenvEnv: map[string]string{
				"KUBECONFIG": "/tmp/kubeconfig",
				"NAMESPACE":  "test-ns",
			},
			envPropagation: &forge.EnvPropagation{
				Whitelist: []string{},
			},
			wantEnvPresent: map[string]string{
				"KUBECONFIG": "/tmp/kubeconfig",
				"NAMESPACE":  "test-ns",
			},
			wantEnvNotPresent: []string{},
		},
		{
			name: "Empty blacklist - no filtering (all vars propagated)",
			testenvEnv: map[string]string{
				"KUBECONFIG": "/tmp/kubeconfig",
				"NAMESPACE":  "test-ns",
			},
			envPropagation: &forge.EnvPropagation{
				Blacklist: []string{},
			},
			wantEnvPresent: map[string]string{
				"KUBECONFIG": "/tmp/kubeconfig",
				"NAMESPACE":  "test-ns",
			},
			wantEnvNotPresent: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the filtering logic from runTestsWrapper
			testEnv := make(map[string]string)

			if len(tt.testenvEnv) > 0 {
				if tt.envPropagation != nil && tt.envPropagation.Disabled {
					// Propagation disabled - skip all testenv env vars
					// testEnv remains empty
				} else if tt.envPropagation != nil && len(tt.envPropagation.Whitelist) > 0 {
					// Whitelist mode - only propagate whitelisted vars
					for _, key := range tt.envPropagation.Whitelist {
						if value, ok := tt.testenvEnv[key]; ok {
							testEnv[key] = value
						}
					}
				} else if tt.envPropagation != nil && len(tt.envPropagation.Blacklist) > 0 {
					// Blacklist mode - propagate all except blacklisted vars
					for key, value := range tt.testenvEnv {
						if !contains(tt.envPropagation.Blacklist, key) {
							testEnv[key] = value
						}
					}
				} else {
					// No filtering - propagate all testenv vars
					for key, value := range tt.testenvEnv {
						testEnv[key] = value
					}
				}
			}

			// Verify expected vars are present with correct values
			for wantKey, wantValue := range tt.wantEnvPresent {
				gotValue, ok := testEnv[wantKey]
				if !ok {
					t.Errorf("Expected env var %q to be present, but it was not found", wantKey)
					continue
				}
				if gotValue != wantValue {
					t.Errorf("Env var %q = %q, want %q", wantKey, gotValue, wantValue)
				}
			}

			// Verify unwanted vars are not present
			for _, unwantedKey := range tt.wantEnvNotPresent {
				if _, ok := testEnv[unwantedKey]; ok {
					t.Errorf("Expected env var %q to NOT be present, but it was found with value %q", unwantedKey, testEnv[unwantedKey])
				}
			}

			// Verify no extra vars are present (only expected ones)
			if len(testEnv) != len(tt.wantEnvPresent) {
				t.Errorf("Expected %d env vars, but got %d. testEnv: %+v", len(tt.wantEnvPresent), len(testEnv), testEnv)
			}
		})
	}
}

// TestContainsHelper tests the contains helper function
func TestContainsHelper(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		item  string
		want  bool
	}{
		{
			name:  "Item exists in slice",
			slice: []string{"apple", "banana", "cherry"},
			item:  "banana",
			want:  true,
		},
		{
			name:  "Item does not exist in slice",
			slice: []string{"apple", "banana", "cherry"},
			item:  "orange",
			want:  false,
		},
		{
			name:  "Empty slice",
			slice: []string{},
			item:  "apple",
			want:  false,
		},
		{
			name:  "Case sensitive - exact match required",
			slice: []string{"KUBECONFIG", "NAMESPACE"},
			item:  "kubeconfig",
			want:  false,
		},
		{
			name:  "Case sensitive - exact match found",
			slice: []string{"KUBECONFIG", "NAMESPACE"},
			item:  "KUBECONFIG",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.slice, tt.item)
			if got != tt.want {
				t.Errorf("contains(%v, %q) = %v, want %v", tt.slice, tt.item, got, tt.want)
			}
		})
	}
}

// TestNormalizeEnvKey verifies environment variable key normalization.
func TestNormalizeEnvKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "testenv-kind.kubeconfig",
			want:  "TESTENV_KIND_KUBECONFIG",
		},
		{
			input: "cluster.name",
			want:  "CLUSTER_NAME",
		},
		{
			input: "registry-url",
			want:  "REGISTRY_URL",
		},
		{
			input: "ALREADY_UPPER",
			want:  "ALREADY_UPPER",
		},
		{
			input: "mixed.Case-Test_123",
			want:  "MIXED_CASE_TEST_123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeEnvKey(tt.input)
			if got != tt.want {
				t.Errorf("normalizeEnvKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestEnvPropagationIntegration tests the full integration with RunInput
func TestEnvPropagationIntegration(t *testing.T) {
	tests := []struct {
		name              string
		input             mcptypes.RunInput
		wantEnvPresent    map[string]string
		wantEnvNotPresent []string
	}{
		{
			name: "Integration: Whitelist with runner override",
			input: mcptypes.RunInput{
				Stage: "integration",
				Name:  "test-whitelist",
				TestenvEnv: map[string]string{
					"KUBECONFIG":   "/tmp/kubeconfig",
					"REGISTRY_URL": "localhost:5000",
					"NAMESPACE":    "test-ns",
				},
				EnvPropagation: &forge.EnvPropagation{
					Whitelist: []string{"KUBECONFIG"},
				},
				Env: map[string]string{
					"CUSTOM_VAR": "custom-value",
				},
			},
			wantEnvPresent: map[string]string{
				"KUBECONFIG": "/tmp/kubeconfig",
				"CUSTOM_VAR": "custom-value",
			},
			wantEnvNotPresent: []string{"REGISTRY_URL", "NAMESPACE"},
		},
		{
			name: "Integration: Disabled with metadata",
			input: mcptypes.RunInput{
				Stage: "integration",
				Name:  "test-disabled",
				TestenvEnv: map[string]string{
					"KUBECONFIG": "/tmp/kubeconfig",
				},
				EnvPropagation: &forge.EnvPropagation{
					Disabled: true,
				},
				TestenvTmpDir: "/tmp/test-123",
				TestenvMetadata: map[string]string{
					"cluster.name": "kind-test",
				},
			},
			wantEnvPresent: map[string]string{
				"FORGE_TESTENV_TMPDIR":        "/tmp/test-123",
				"FORGE_METADATA_CLUSTER_NAME": "kind-test",
			},
			wantEnvNotPresent: []string{"KUBECONFIG"},
		},
		{
			name: "Integration: Blacklist with artifact files",
			input: mcptypes.RunInput{
				Stage: "integration",
				Name:  "test-blacklist",
				TestenvEnv: map[string]string{
					"KUBECONFIG":   "/tmp/kubeconfig",
					"SECRET_TOKEN": "secret",
				},
				EnvPropagation: &forge.EnvPropagation{
					Blacklist: []string{"SECRET_TOKEN"},
				},
				TestenvTmpDir: "/tmp/test-123",
				ArtifactFiles: map[string]string{
					"testenv-kind.kubeconfig": "kubeconfig",
				},
			},
			wantEnvPresent: map[string]string{
				"KUBECONFIG":                             "/tmp/kubeconfig",
				"FORGE_TESTENV_TMPDIR":                   "/tmp/test-123",
				"FORGE_ARTIFACT_TESTENV_KIND_KUBECONFIG": "/tmp/test-123/kubeconfig",
			},
			wantEnvNotPresent: []string{"SECRET_TOKEN"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the full env building logic from runTestsWrapper
			testEnv := make(map[string]string)

			// Apply EnvPropagation filtering
			if len(tt.input.TestenvEnv) > 0 {
				if tt.input.EnvPropagation != nil && tt.input.EnvPropagation.Disabled {
					// Skip all testenv env vars
				} else if tt.input.EnvPropagation != nil && len(tt.input.EnvPropagation.Whitelist) > 0 {
					for _, key := range tt.input.EnvPropagation.Whitelist {
						if value, ok := tt.input.TestenvEnv[key]; ok {
							testEnv[key] = value
						}
					}
				} else if tt.input.EnvPropagation != nil && len(tt.input.EnvPropagation.Blacklist) > 0 {
					for key, value := range tt.input.TestenvEnv {
						if !contains(tt.input.EnvPropagation.Blacklist, key) {
							testEnv[key] = value
						}
					}
				} else {
					for key, value := range tt.input.TestenvEnv {
						testEnv[key] = value
					}
				}
			}

			// Add legacy FORGE_* env vars
			if tt.input.TestenvTmpDir != "" {
				testEnv["FORGE_TESTENV_TMPDIR"] = tt.input.TestenvTmpDir
			}
			if len(tt.input.ArtifactFiles) > 0 {
				for key, relPath := range tt.input.ArtifactFiles {
					var absPath string
					if tt.input.TestenvTmpDir != "" {
						absPath = tt.input.TestenvTmpDir + "/" + relPath
					} else {
						absPath = relPath
					}
					envKey := "FORGE_ARTIFACT_" + normalizeEnvKey(key)
					testEnv[envKey] = absPath
				}
			}
			if len(tt.input.TestenvMetadata) > 0 {
				for key, value := range tt.input.TestenvMetadata {
					envKey := "FORGE_METADATA_" + normalizeEnvKey(key)
					testEnv[envKey] = value
				}
			}

			// Override with runner-specific env
			if len(tt.input.Env) > 0 {
				for key, value := range tt.input.Env {
					testEnv[key] = value
				}
			}

			// Verify expected vars are present
			for wantKey, wantValue := range tt.wantEnvPresent {
				gotValue, ok := testEnv[wantKey]
				if !ok {
					t.Errorf("Expected env var %q to be present, but it was not found", wantKey)
					continue
				}
				if gotValue != wantValue {
					t.Errorf("Env var %q = %q, want %q", wantKey, gotValue, wantValue)
				}
			}

			// Verify unwanted vars are not present
			for _, unwantedKey := range tt.wantEnvNotPresent {
				if _, ok := testEnv[unwantedKey]; ok {
					t.Errorf("Expected env var %q to NOT be present, but it was found with value %q", unwantedKey, testEnv[unwantedKey])
				}
			}
		})
	}
}
