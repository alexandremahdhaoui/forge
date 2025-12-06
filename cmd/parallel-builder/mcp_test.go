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
	"testing"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -----------------------------------------------------------------------------
// Tests for combineArtifacts
// -----------------------------------------------------------------------------

func TestCombineArtifacts_EmptyList(t *testing.T) {
	result := combineArtifacts("test-artifact", []*forge.Artifact{})

	assert.Equal(t, "test-artifact", result.Name)
	assert.Equal(t, "parallel-build", result.Type)
	assert.Equal(t, ".", result.Location)
	assert.Equal(t, "no-artifacts", result.Version)
	// Verify timestamp is valid RFC3339
	_, err := time.Parse(time.RFC3339, result.Timestamp)
	assert.NoError(t, err, "timestamp should be valid RFC3339")
}

func TestCombineArtifacts_SingleArtifact(t *testing.T) {
	artifacts := []*forge.Artifact{
		{
			Name:      "artifact-1",
			Type:      "binary",
			Location:  "/path/to/artifact",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   "v1.0.0",
		},
	}

	result := combineArtifacts("combined", artifacts)

	assert.Equal(t, "combined", result.Name)
	assert.Equal(t, "parallel-build", result.Type)
	assert.Equal(t, "/path/to/artifact", result.Location)
	assert.Equal(t, "1-artifacts", result.Version)
	// Verify timestamp is valid RFC3339
	_, err := time.Parse(time.RFC3339, result.Timestamp)
	assert.NoError(t, err, "timestamp should be valid RFC3339")
}

func TestCombineArtifacts_MultipleArtifacts(t *testing.T) {
	artifacts := []*forge.Artifact{
		{
			Name:      "artifact-1",
			Type:      "binary",
			Location:  "/path/to/artifact1",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   "v1.0.0",
		},
		{
			Name:      "artifact-2",
			Type:      "container",
			Location:  "/path/to/artifact2",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   "v2.0.0",
		},
		{
			Name:      "artifact-3",
			Type:      "binary",
			Location:  "/path/to/artifact3",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   "v3.0.0",
		},
	}

	result := combineArtifacts("all-binaries", artifacts)

	assert.Equal(t, "all-binaries", result.Name)
	assert.Equal(t, "parallel-build", result.Type)
	assert.Equal(t, "multiple", result.Location)
	assert.Equal(t, "3-artifacts", result.Version)
	// Verify timestamp is valid RFC3339
	_, err := time.Parse(time.RFC3339, result.Timestamp)
	assert.NoError(t, err, "timestamp should be valid RFC3339")
}

// -----------------------------------------------------------------------------
// Tests for parseArtifact
// -----------------------------------------------------------------------------

func TestParseArtifact_NilResponse(t *testing.T) {
	result, err := parseArtifact(nil)

	require.NoError(t, err)
	assert.Equal(t, "unknown", result.Name)
	assert.Equal(t, "unknown", result.Type)
	assert.Equal(t, ".", result.Location)
	assert.Equal(t, "unknown", result.Version)
	// Verify timestamp is valid RFC3339
	_, parseErr := time.Parse(time.RFC3339, result.Timestamp)
	assert.NoError(t, parseErr, "timestamp should be valid RFC3339")
}

func TestParseArtifact_ValidArtifactMap(t *testing.T) {
	resp := map[string]any{
		"name":      "my-artifact",
		"type":      "binary",
		"location":  "/build/bin/my-app",
		"timestamp": "2024-01-15T10:30:00Z",
		"version":   "abc123",
	}

	result, err := parseArtifact(resp)

	require.NoError(t, err)
	assert.Equal(t, "my-artifact", result.Name)
	assert.Equal(t, "binary", result.Type)
	assert.Equal(t, "/build/bin/my-app", result.Location)
	assert.Equal(t, "2024-01-15T10:30:00Z", result.Timestamp)
	assert.Equal(t, "abc123", result.Version)
}

func TestParseArtifact_PartialFields(t *testing.T) {
	tests := []struct {
		name     string
		resp     map[string]any
		expected *forge.Artifact
	}{
		{
			name: "only name provided",
			resp: map[string]any{
				"name": "test-artifact",
			},
			expected: &forge.Artifact{
				Name:     "test-artifact",
				Type:     "unknown",
				Location: ".",
				Version:  "unknown",
			},
		},
		{
			name: "name and type provided",
			resp: map[string]any{
				"name": "test-artifact",
				"type": "container",
			},
			expected: &forge.Artifact{
				Name:     "test-artifact",
				Type:     "container",
				Location: ".",
				Version:  "unknown",
			},
		},
		{
			name: "location and version provided",
			resp: map[string]any{
				"location": "/some/path",
				"version":  "v1.2.3",
			},
			expected: &forge.Artifact{
				Name:     "unknown",
				Type:     "unknown",
				Location: "/some/path",
				Version:  "v1.2.3",
			},
		},
		{
			name: "empty map",
			resp: map[string]any{},
			expected: &forge.Artifact{
				Name:     "unknown",
				Type:     "unknown",
				Location: ".",
				Version:  "unknown",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseArtifact(tt.resp)

			require.NoError(t, err)
			assert.Equal(t, tt.expected.Name, result.Name)
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.Location, result.Location)
			assert.Equal(t, tt.expected.Version, result.Version)
			// Timestamp should be set to current time if not provided
			if tt.resp["timestamp"] == nil {
				_, parseErr := time.Parse(time.RFC3339, result.Timestamp)
				assert.NoError(t, parseErr, "timestamp should be valid RFC3339")
			}
		})
	}
}

func TestParseArtifact_UnexpectedType(t *testing.T) {
	tests := []struct {
		name string
		resp interface{}
	}{
		{
			name: "string response",
			resp: "not a map",
		},
		{
			name: "integer response",
			resp: 42,
		},
		{
			name: "slice response",
			resp: []string{"a", "b", "c"},
		},
		{
			name: "bool response",
			resp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseArtifact(tt.resp)

			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "unexpected response type")
		})
	}
}

func TestParseArtifact_NonStringFieldValues(t *testing.T) {
	// Test that non-string values in the map don't cause crashes
	// (they just get skipped, resulting in default values)
	resp := map[string]any{
		"name":      123,           // not a string
		"type":      true,          // not a string
		"location":  []string{"a"}, // not a string
		"timestamp": 12345,         // not a string
		"version":   nil,           // nil
	}

	result, err := parseArtifact(resp)

	require.NoError(t, err)
	assert.Equal(t, "unknown", result.Name)
	assert.Equal(t, "unknown", result.Type)
	assert.Equal(t, ".", result.Location)
	assert.Equal(t, "unknown", result.Version)
}

// -----------------------------------------------------------------------------
// Tests for mapToStruct
// -----------------------------------------------------------------------------

func TestMapToStruct_NilMap(t *testing.T) {
	var spec ParallelBuilderSpec
	err := mapToStruct(nil, &spec)

	assert.NoError(t, err)
	// spec should remain at zero value
	assert.Empty(t, spec.Builders)
}

func TestMapToStruct_ValidMap(t *testing.T) {
	input := map[string]any{
		"builders": []any{
			map[string]any{
				"name":   "builder-1",
				"engine": "go://go-build",
				"spec": map[string]any{
					"src":  "./cmd/app",
					"dest": "./build/bin",
				},
			},
			map[string]any{
				"name":   "builder-2",
				"engine": "go://go-build",
				"spec": map[string]any{
					"src":  "./cmd/tool",
					"dest": "./build/bin",
				},
			},
		},
	}

	var spec ParallelBuilderSpec
	err := mapToStruct(input, &spec)

	require.NoError(t, err)
	require.Len(t, spec.Builders, 2)

	assert.Equal(t, "builder-1", spec.Builders[0].Name)
	assert.Equal(t, "go://go-build", spec.Builders[0].Engine)
	assert.Equal(t, "./cmd/app", spec.Builders[0].Spec["src"])
	assert.Equal(t, "./build/bin", spec.Builders[0].Spec["dest"])

	assert.Equal(t, "builder-2", spec.Builders[1].Name)
	assert.Equal(t, "go://go-build", spec.Builders[1].Engine)
	assert.Equal(t, "./cmd/tool", spec.Builders[1].Spec["src"])
	assert.Equal(t, "./build/bin", spec.Builders[1].Spec["dest"])
}

func TestMapToStruct_EmptyBuildersArray(t *testing.T) {
	input := map[string]any{
		"builders": []any{},
	}

	var spec ParallelBuilderSpec
	err := mapToStruct(input, &spec)

	require.NoError(t, err)
	assert.Empty(t, spec.Builders)
}

func TestMapToStruct_BuilderWithoutName(t *testing.T) {
	input := map[string]any{
		"builders": []any{
			map[string]any{
				"engine": "go://go-build",
				"spec": map[string]any{
					"src": "./cmd/app",
				},
			},
		},
	}

	var spec ParallelBuilderSpec
	err := mapToStruct(input, &spec)

	require.NoError(t, err)
	require.Len(t, spec.Builders, 1)
	assert.Empty(t, spec.Builders[0].Name)
	assert.Equal(t, "go://go-build", spec.Builders[0].Engine)
}

func TestMapToStruct_InvalidTarget(t *testing.T) {
	input := map[string]any{
		"builders": []any{
			map[string]any{
				"engine": "go://go-build",
			},
		},
	}

	// Try to unmarshal into a non-pointer
	var str string
	err := mapToStruct(input, &str)

	// json.Unmarshal will return an error for mismatched types
	assert.Error(t, err)
}

func TestMapToStruct_BuilderConfig(t *testing.T) {
	// Test that individual BuilderConfig fields are correctly parsed
	input := map[string]any{
		"builders": []any{
			map[string]any{
				"name":   "test-builder",
				"engine": "go://parallel-builder",
				"spec": map[string]any{
					"nested": map[string]any{
						"key": "value",
					},
					"array": []any{1, 2, 3},
				},
			},
		},
	}

	var spec ParallelBuilderSpec
	err := mapToStruct(input, &spec)

	require.NoError(t, err)
	require.Len(t, spec.Builders, 1)

	builder := spec.Builders[0]
	assert.Equal(t, "test-builder", builder.Name)
	assert.Equal(t, "go://parallel-builder", builder.Engine)

	// Check nested spec values
	nested, ok := builder.Spec["nested"].(map[string]any)
	require.True(t, ok, "nested should be a map")
	assert.Equal(t, "value", nested["key"])

	arr, ok := builder.Spec["array"].([]any)
	require.True(t, ok, "array should be a slice")
	assert.Len(t, arr, 3)
}

// -----------------------------------------------------------------------------
// Tests for ParallelBuilderSpec and BuilderConfig types
// -----------------------------------------------------------------------------

func TestParallelBuilderSpec_ZeroValue(t *testing.T) {
	var spec ParallelBuilderSpec
	assert.Nil(t, spec.Builders)
}

func TestBuilderConfig_ZeroValue(t *testing.T) {
	var config BuilderConfig
	assert.Empty(t, config.Name)
	assert.Empty(t, config.Engine)
	assert.Nil(t, config.Spec)
}
