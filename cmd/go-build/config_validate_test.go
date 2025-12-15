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
	// Valid spec with both args and env
	spec := map[string]interface{}{
		"args": []interface{}{"-tags=netgo", "-ldflags=-w -s"},
		"env": map[string]interface{}{
			"GOOS":   "linux",
			"GOARCH": "amd64",
		},
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

func TestConfigValidate_InvalidArgsType(t *testing.T) {
	// args is not an array (it's a string)
	spec := map[string]interface{}{
		"args": "invalid-not-an-array",
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected []string")
}

func TestConfigValidate_InvalidEnvType(t *testing.T) {
	// env is not a map (it's a string)
	spec := map[string]interface{}{
		"env": "invalid-not-a-map",
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected map[string]string")
}

func TestConfigValidate_InvalidArgsElement(t *testing.T) {
	// args array contains a non-string element
	spec := map[string]interface{}{
		"args": []interface{}{"-tags=netgo", 123, "-ldflags=-w"},
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected string")
}

func TestConfigValidate_InvalidEnvValue(t *testing.T) {
	// env map contains a non-string value
	spec := map[string]interface{}{
		"env": map[string]interface{}{
			"GOOS":   "linux",
			"GOARCH": 123, // invalid: number instead of string
		},
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected string")
}

func TestConfigValidate_ValidArgsOnly(t *testing.T) {
	// Valid args without env
	spec := map[string]interface{}{
		"args": []interface{}{"-tags=netgo"},
	}

	output := ValidateMap(spec)

	assert.True(t, output.Valid)
	assert.Empty(t, output.Errors)
}

func TestConfigValidate_ValidEnvOnly(t *testing.T) {
	// Valid env without args
	spec := map[string]interface{}{
		"env": map[string]interface{}{
			"CGO_ENABLED": "0",
		},
	}

	output := ValidateMap(spec)

	assert.True(t, output.Valid)
	assert.Empty(t, output.Errors)
}

// TestFromMap tests that FromMap correctly parses a valid spec
func TestFromMap_Valid(t *testing.T) {
	spec := map[string]interface{}{
		"args": []interface{}{"-tags=netgo"},
		"env": map[string]interface{}{
			"GOOS": "linux",
		},
	}

	s, err := FromMap(spec)
	require.NoError(t, err)

	assert.Equal(t, []string{"-tags=netgo"}, s.Args)
	assert.Equal(t, map[string]string{"GOOS": "linux"}, s.Env)
}

// TestToMap tests that ToMap correctly serializes a Spec
func TestToMap(t *testing.T) {
	s := &Spec{
		Args: []string{"-tags=netgo"},
		Env:  map[string]string{"GOOS": "linux"},
	}

	m := s.ToMap()

	assert.Equal(t, []string{"-tags=netgo"}, m["args"])
	assert.Equal(t, map[string]string{"GOOS": "linux"}, m["env"])
}
