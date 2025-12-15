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
	spec := map[string]interface{}{
		"command": "echo",
		"args":    []interface{}{"hello", "world"},
		"env": map[string]interface{}{
			"FOO": "bar",
		},
		"workDir": "/tmp",
		"envFile": ".env",
	}

	output := ValidateMap(spec)

	assert.True(t, output.Valid)
	assert.Empty(t, output.Errors)
}

func TestConfigValidate_MissingCommand(t *testing.T) {
	spec := map[string]interface{}{
		"args": []interface{}{"hello"},
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec.command", output.Errors[0].Field)
	assert.Equal(t, "required field is missing", output.Errors[0].Message)
}

func TestConfigValidate_EmptyCommand(t *testing.T) {
	spec := map[string]interface{}{
		"command": "",
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec.command", output.Errors[0].Field)
	assert.Equal(t, "required field is missing", output.Errors[0].Message)
}

func TestConfigValidate_InvalidArgsType(t *testing.T) {
	spec := map[string]interface{}{
		"command": "echo",
		"args":    "not-an-array",
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected []string")
}

func TestConfigValidate_InvalidEnvType(t *testing.T) {
	spec := map[string]interface{}{
		"command": "echo",
		"env":     "not-a-map",
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected map[string]string")
}

func TestConfigValidate_InvalidWorkDirType(t *testing.T) {
	spec := map[string]interface{}{
		"command": "echo",
		"workDir": 123,
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected string")
}

func TestConfigValidate_InvalidEnvFileType(t *testing.T) {
	spec := map[string]interface{}{
		"command": "echo",
		"envFile": true,
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected string")
}

func TestConfigValidate_NilSpec(t *testing.T) {
	output := ValidateMap(nil)

	// With generated code, nil spec parses to empty Spec, which then fails
	// validation because 'command' is required
	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec.command", output.Errors[0].Field)
	assert.Equal(t, "required field is missing", output.Errors[0].Message)
}

func TestConfigValidate_EmptySpec(t *testing.T) {
	spec := map[string]interface{}{}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec.command", output.Errors[0].Field)
	assert.Equal(t, "required field is missing", output.Errors[0].Message)
}

func TestConfigValidate_MinimalValidSpec(t *testing.T) {
	spec := map[string]interface{}{
		"command": "ls",
	}

	output := ValidateMap(spec)

	assert.True(t, output.Valid)
	assert.Empty(t, output.Errors)
}

func TestConfigValidate_InvalidArgsElementType(t *testing.T) {
	spec := map[string]interface{}{
		"command": "echo",
		"args":    []interface{}{"hello", 123},
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected string")
}

func TestConfigValidate_InvalidEnvValueType(t *testing.T) {
	spec := map[string]interface{}{
		"command": "echo",
		"env": map[string]interface{}{
			"FOO": 123,
		},
	}

	output := ValidateMap(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected string")
}

// TestFromMap_Valid tests that FromMap correctly parses a valid spec
func TestFromMap_Valid(t *testing.T) {
	spec := map[string]interface{}{
		"command": "echo",
		"args":    []interface{}{"hello", "world"},
		"env": map[string]interface{}{
			"FOO": "bar",
		},
		"workDir": "/tmp",
		"envFile": ".env",
	}

	s, err := FromMap(spec)
	require.NoError(t, err)

	assert.Equal(t, "echo", s.Command)
	assert.Equal(t, []string{"hello", "world"}, s.Args)
	assert.Equal(t, map[string]string{"FOO": "bar"}, s.Env)
	assert.Equal(t, "/tmp", s.WorkDir)
	assert.Equal(t, ".env", s.EnvFile)
}

// TestToMap tests that ToMap correctly serializes a Spec
func TestToMap(t *testing.T) {
	s := &Spec{
		Command: "echo",
		Args:    []string{"hello"},
		Env:     map[string]string{"FOO": "bar"},
		WorkDir: "/tmp",
		EnvFile: ".env",
	}

	m := s.ToMap()

	assert.Equal(t, "echo", m["command"])
	assert.Equal(t, []string{"hello"}, m["args"])
	assert.Equal(t, map[string]string{"FOO": "bar"}, m["env"])
	assert.Equal(t, "/tmp", m["workDir"])
	assert.Equal(t, ".env", m["envFile"])
}
