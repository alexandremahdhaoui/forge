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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate_ValidSpec(t *testing.T) {
	// Valid spec with all fields
	spec := map[string]interface{}{
		"dockerfile": "Dockerfile.custom",
		"context":    "./build",
		"buildArgs": map[string]interface{}{
			"VERSION": "1.0.0",
			"COMMIT":  "abc123",
		},
		"tags":     []interface{}{"myimage:latest", "myimage:v1"},
		"target":   "production",
		"push":     true,
		"registry": "docker.io/myrepo",
	}

	output := ValidateMap(spec)

	assert.True(t, output.Valid)
	assert.Empty(t, output.Errors)
}

func TestConfigValidate_EmptySpec(t *testing.T) {
	// Empty spec should be valid (no required fields)
	tests := []struct {
		name string
		spec map[string]interface{}
	}{
		{
			name: "nil spec",
			spec: nil,
		},
		{
			name: "empty spec",
			spec: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := ValidateMap(tt.spec)

			assert.True(t, output.Valid)
			assert.Empty(t, output.Errors)
		})
	}
}

func TestConfigValidate_InvalidDockerfileType(t *testing.T) {
	// dockerfile is not a string (it's a number)
	spec := map[string]interface{}{
		"dockerfile": 123,
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected string")
}

func TestConfigValidate_InvalidContextType(t *testing.T) {
	// context is not a string (it's a boolean)
	spec := map[string]interface{}{
		"context": true,
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected string")
}

func TestConfigValidate_InvalidBuildArgsType(t *testing.T) {
	// buildArgs is not a map (it's a string)
	spec := map[string]interface{}{
		"buildArgs": "invalid-not-a-map",
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected map[string]string")
}

func TestConfigValidate_InvalidBuildArgsValue(t *testing.T) {
	// buildArgs map contains a non-string value
	spec := map[string]interface{}{
		"buildArgs": map[string]interface{}{
			"VERSION": "1.0.0",
			"COUNT":   123, // invalid: number instead of string
		},
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected string")
}

func TestConfigValidate_InvalidTagsType(t *testing.T) {
	// tags is not an array (it's a string)
	spec := map[string]interface{}{
		"tags": "invalid-not-an-array",
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected []string")
}

func TestConfigValidate_InvalidTagsElement(t *testing.T) {
	// tags array contains a non-string element
	spec := map[string]interface{}{
		"tags": []interface{}{"myimage:latest", 123, "myimage:v1"},
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected string")
}

func TestConfigValidate_InvalidTargetType(t *testing.T) {
	// target is not a string (it's a number)
	spec := map[string]interface{}{
		"target": 456,
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected string")
}

func TestConfigValidate_InvalidPushType(t *testing.T) {
	// push is not a bool (it's a string)
	spec := map[string]interface{}{
		"push": "yes",
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected bool")
}

func TestConfigValidate_InvalidRegistryType(t *testing.T) {
	// registry is not a string (it's an array)
	spec := map[string]interface{}{
		"registry": []interface{}{"docker.io", "ghcr.io"},
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected string")
}

func TestConfigValidate_PartialValidSpec(t *testing.T) {
	tests := []struct {
		name string
		spec map[string]interface{}
	}{
		{
			name: "only dockerfile",
			spec: map[string]interface{}{
				"dockerfile": "Dockerfile",
			},
		},
		{
			name: "only buildArgs",
			spec: map[string]interface{}{
				"buildArgs": map[string]interface{}{
					"VERSION": "1.0.0",
				},
			},
		},
		{
			name: "only tags",
			spec: map[string]interface{}{
				"tags": []interface{}{"myimage:latest"},
			},
		},
		{
			name: "push false",
			spec: map[string]interface{}{
				"push": false,
			},
		},
		{
			name: "context and target",
			spec: map[string]interface{}{
				"context": ".",
				"target":  "builder",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := ValidateMap(tt.spec)

			assert.True(t, output.Valid)
			assert.Empty(t, output.Errors)
		})
	}
}

// TestFromMap tests that FromMap correctly parses a valid spec
func TestFromMap_Valid(t *testing.T) {
	spec := map[string]interface{}{
		"dockerfile": "Dockerfile.custom",
		"context":    "./build",
		"buildArgs": map[string]interface{}{
			"VERSION": "1.0.0",
		},
		"tags":     []interface{}{"myimage:latest"},
		"target":   "production",
		"push":     true,
		"registry": "docker.io/myrepo",
	}

	s, err := FromMap(spec)
	require.NoError(t, err)

	assert.Equal(t, "Dockerfile.custom", s.Dockerfile)
	assert.Equal(t, "./build", s.Context)
	assert.Equal(t, map[string]string{"VERSION": "1.0.0"}, s.BuildArgs)
	assert.Equal(t, []string{"myimage:latest"}, s.Tags)
	assert.Equal(t, "production", s.Target)
	assert.True(t, s.Push)
	assert.Equal(t, "docker.io/myrepo", s.Registry)
}

// TestToMap tests that ToMap correctly serializes a Spec
func TestToMap(t *testing.T) {
	s := &Spec{
		Dockerfile: "Dockerfile",
		Context:    ".",
		BuildArgs:  map[string]string{"VERSION": "1.0.0"},
		Tags:       []string{"myimage:latest"},
		Target:     "builder",
		Push:       true,
		Registry:   "docker.io",
	}

	m := s.ToMap()

	assert.Equal(t, "Dockerfile", m["dockerfile"])
	assert.Equal(t, ".", m["context"])
	assert.Equal(t, map[string]string{"VERSION": "1.0.0"}, m["buildArgs"])
	assert.Equal(t, []string{"myimage:latest"}, m["tags"])
	assert.Equal(t, "builder", m["target"])
	assert.Equal(t, true, m["push"])
	assert.Equal(t, "docker.io", m["registry"])
}
