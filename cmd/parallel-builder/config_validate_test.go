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

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -----------------------------------------------------------------------------
// Tests for validateOwnSpec
// -----------------------------------------------------------------------------

func TestConfigValidate_ValidSpec(t *testing.T) {
	spec := map[string]interface{}{
		"builders": []interface{}{
			map[string]interface{}{
				"name":   "builder-1",
				"engine": "go://go-build",
				"spec": map[string]interface{}{
					"src":  "./cmd/app",
					"dest": "./build/bin",
				},
			},
			map[string]interface{}{
				"name":   "builder-2",
				"engine": "go://generic-builder",
				"spec": map[string]interface{}{
					"command": "echo",
					"args":    []interface{}{"hello"},
				},
			},
		},
	}

	builders, errors := validateOwnSpec(spec)

	assert.Empty(t, errors)
	require.Len(t, builders, 2)
	assert.Equal(t, "builder-1", builders[0].Name)
	assert.Equal(t, "go://go-build", builders[0].Engine)
	assert.Equal(t, "builder-2", builders[1].Name)
	assert.Equal(t, "go://generic-builder", builders[1].Engine)
}

func TestConfigValidate_NilSpec(t *testing.T) {
	builders, errors := validateOwnSpec(nil)

	assert.Nil(t, builders)
	require.Len(t, errors, 1)
	assert.Equal(t, "spec.builders", errors[0].Field)
	assert.Equal(t, "required field is missing", errors[0].Message)
}

func TestConfigValidate_EmptySpec(t *testing.T) {
	spec := map[string]interface{}{}

	builders, errors := validateOwnSpec(spec)

	assert.Nil(t, builders)
	require.Len(t, errors, 1)
	assert.Equal(t, "spec.builders", errors[0].Field)
	assert.Equal(t, "required field is missing", errors[0].Message)
}

func TestConfigValidate_MissingBuilders(t *testing.T) {
	spec := map[string]interface{}{
		"other": "value",
	}

	builders, errors := validateOwnSpec(spec)

	assert.Nil(t, builders)
	require.Len(t, errors, 1)
	assert.Equal(t, "spec.builders", errors[0].Field)
	assert.Equal(t, "required field is missing", errors[0].Message)
}

func TestConfigValidate_BuildersNotArray(t *testing.T) {
	spec := map[string]interface{}{
		"builders": "not-an-array",
	}

	builders, errors := validateOwnSpec(spec)

	assert.Nil(t, builders)
	require.Len(t, errors, 1)
	assert.Equal(t, "spec.builders", errors[0].Field)
	assert.Contains(t, errors[0].Message, "expected array")
}

func TestConfigValidate_EmptyBuildersArray(t *testing.T) {
	spec := map[string]interface{}{
		"builders": []interface{}{},
	}

	builders, errors := validateOwnSpec(spec)

	assert.Nil(t, builders)
	require.Len(t, errors, 1)
	assert.Equal(t, "spec.builders", errors[0].Field)
	assert.Equal(t, "builders array cannot be empty", errors[0].Message)
}

func TestConfigValidate_BuilderNotObject(t *testing.T) {
	spec := map[string]interface{}{
		"builders": []interface{}{
			"not-an-object",
		},
	}

	builders, errors := validateOwnSpec(spec)

	assert.Empty(t, builders)
	require.Len(t, errors, 1)
	assert.Equal(t, "spec.builders[0]", errors[0].Field)
	assert.Contains(t, errors[0].Message, "expected object")
}

func TestConfigValidate_BuilderWithoutName(t *testing.T) {
	spec := map[string]interface{}{
		"builders": []interface{}{
			map[string]interface{}{
				"engine": "go://go-build",
				"spec": map[string]interface{}{
					"src": "./cmd/app",
				},
			},
		},
	}

	builders, errors := validateOwnSpec(spec)

	assert.Empty(t, errors)
	require.Len(t, builders, 1)
	assert.Empty(t, builders[0].Name)
	assert.Equal(t, "go://go-build", builders[0].Engine)
}

func TestConfigValidate_BuilderWithoutEngine(t *testing.T) {
	spec := map[string]interface{}{
		"builders": []interface{}{
			map[string]interface{}{
				"name": "builder-1",
				"spec": map[string]interface{}{
					"src": "./cmd/app",
				},
			},
		},
	}

	// validateOwnSpec extracts builders even without engine,
	// but the recursive validation will catch missing engine
	builders, errors := validateOwnSpec(spec)

	assert.Empty(t, errors)
	require.Len(t, builders, 1)
	assert.Equal(t, "builder-1", builders[0].Name)
	assert.Empty(t, builders[0].Engine)
}

func TestConfigValidate_BuilderWithoutSpec(t *testing.T) {
	spec := map[string]interface{}{
		"builders": []interface{}{
			map[string]interface{}{
				"name":   "builder-1",
				"engine": "go://go-build",
			},
		},
	}

	builders, errors := validateOwnSpec(spec)

	assert.Empty(t, errors)
	require.Len(t, builders, 1)
	assert.Equal(t, "builder-1", builders[0].Name)
	assert.Equal(t, "go://go-build", builders[0].Engine)
	assert.Nil(t, builders[0].Spec)
}

func TestConfigValidate_MultipleBuilders(t *testing.T) {
	spec := map[string]interface{}{
		"builders": []interface{}{
			map[string]interface{}{
				"name":   "builder-1",
				"engine": "go://go-build",
			},
			map[string]interface{}{
				"name":   "builder-2",
				"engine": "go://container-build",
			},
			map[string]interface{}{
				"name":   "builder-3",
				"engine": "go://generic-builder",
			},
		},
	}

	builders, errors := validateOwnSpec(spec)

	assert.Empty(t, errors)
	require.Len(t, builders, 3)
	assert.Equal(t, "builder-1", builders[0].Name)
	assert.Equal(t, "builder-2", builders[1].Name)
	assert.Equal(t, "builder-3", builders[2].Name)
}

func TestConfigValidate_MixedValidAndInvalidBuilders(t *testing.T) {
	spec := map[string]interface{}{
		"builders": []interface{}{
			map[string]interface{}{
				"name":   "builder-1",
				"engine": "go://go-build",
			},
			"invalid-string", // Not an object
			map[string]interface{}{
				"name":   "builder-3",
				"engine": "go://generic-builder",
			},
		},
	}

	builders, errors := validateOwnSpec(spec)

	// Should have one error for the invalid builder
	require.Len(t, errors, 1)
	assert.Equal(t, "spec.builders[1]", errors[0].Field)
	assert.Contains(t, errors[0].Message, "expected object")

	// Should have two valid builders
	require.Len(t, builders, 2)
	assert.Equal(t, "builder-1", builders[0].Name)
	assert.Equal(t, "builder-3", builders[1].Name)
}

// -----------------------------------------------------------------------------
// Tests for aggregateResults
// -----------------------------------------------------------------------------

func TestAggregateResults_AllValid(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://go-build",
				SpecType: "builder",
				SpecName: "builder-1",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid:  true,
				Errors: []mcptypes.ValidationError{},
			},
		},
		{
			Ref: engineReference{
				URI:      "go://generic-builder",
				SpecType: "builder",
				SpecName: "builder-2",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid:  true,
				Errors: []mcptypes.ValidationError{},
			},
		},
	}

	combined := aggregateResults(results)

	assert.True(t, combined.Valid)
	assert.Empty(t, combined.Errors)
}

func TestAggregateResults_SomeInvalid(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://go-build",
				SpecType: "builder",
				SpecName: "builder-1",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid:  true,
				Errors: []mcptypes.ValidationError{},
			},
		},
		{
			Ref: engineReference{
				URI:      "go://generic-builder",
				SpecType: "builder",
				SpecName: "builder-2",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: false,
				Errors: []mcptypes.ValidationError{
					{
						Field:   "spec.command",
						Message: "required field is missing",
					},
				},
			},
		},
	}

	combined := aggregateResults(results)

	assert.False(t, combined.Valid)
	require.Len(t, combined.Errors, 1)
	assert.Equal(t, "spec.command", combined.Errors[0].Field)
	assert.Equal(t, "go://generic-builder", combined.Errors[0].Engine)
	assert.Equal(t, "builder", combined.Errors[0].SpecType)
	assert.Equal(t, "builder-2", combined.Errors[0].SpecName)
}

func TestAggregateResults_InfraError(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://unknown-engine",
				SpecType: "builder",
				SpecName: "builder-1",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid:      false,
				InfraError: "failed to resolve engine: unknown engine",
			},
		},
	}

	combined := aggregateResults(results)

	assert.False(t, combined.Valid)
	require.Len(t, combined.Errors, 1)
	assert.Equal(t, "", combined.Errors[0].Field)
	assert.Equal(t, "failed to resolve engine: unknown engine", combined.Errors[0].Message)
	assert.Equal(t, "go://unknown-engine", combined.Errors[0].Engine)
}

func TestAggregateResults_MergesWarnings(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://go-build",
				SpecType: "builder",
				SpecName: "builder-1",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: true,
				Warnings: []mcptypes.ValidationWarning{
					{Message: "Warning from builder-1"},
				},
			},
		},
		{
			Ref: engineReference{
				URI:      "go://generic-builder",
				SpecType: "builder",
				SpecName: "builder-2",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: true,
				Warnings: []mcptypes.ValidationWarning{
					{Message: "Warning from builder-2"},
				},
			},
		},
	}

	combined := aggregateResults(results)

	assert.True(t, combined.Valid)
	require.Len(t, combined.Warnings, 2)
	assert.Equal(t, "Warning from builder-1", combined.Warnings[0].Message)
	assert.Equal(t, "Warning from builder-2", combined.Warnings[1].Message)
}

func TestAggregateResults_PreservesEngineContext(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://generic-builder",
				SpecType: "builder",
				SpecName: "builder-1",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: false,
				Errors: []mcptypes.ValidationError{
					{
						Field:   "spec.command",
						Message: "required field is missing",
						// Engine not set - should be filled by aggregateResults
					},
				},
			},
		},
	}

	combined := aggregateResults(results)

	require.Len(t, combined.Errors, 1)
	assert.Equal(t, "go://generic-builder", combined.Errors[0].Engine)
	assert.Equal(t, "builder", combined.Errors[0].SpecType)
	assert.Equal(t, "builder-1", combined.Errors[0].SpecName)
}

func TestAggregateResults_PreservesExistingEngine(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://parallel-builder",
				SpecType: "builder",
				SpecName: "nested-builder",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: false,
				Errors: []mcptypes.ValidationError{
					{
						Field:   "spec.command",
						Message: "required field is missing",
						Engine:  "go://generic-builder", // Already set by nested validation
					},
				},
			},
		},
	}

	combined := aggregateResults(results)

	require.Len(t, combined.Errors, 1)
	// Should preserve the existing engine, not overwrite with ref.URI
	assert.Equal(t, "go://generic-builder", combined.Errors[0].Engine)
}

func TestAggregateResults_EmptyResults(t *testing.T) {
	combined := aggregateResults([]validationResult{})

	assert.True(t, combined.Valid)
	assert.Empty(t, combined.Errors)
	assert.Empty(t, combined.Warnings)
}

func TestAggregateResults_NilOutput(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://go-build",
				SpecType: "builder",
				SpecName: "builder-1",
			},
			Output: nil,
		},
	}

	combined := aggregateResults(results)

	assert.True(t, combined.Valid)
	assert.Empty(t, combined.Errors)
}

// -----------------------------------------------------------------------------
// Tests for parseConfigValidateOutput
// -----------------------------------------------------------------------------

func TestParseConfigValidateOutput_NilResult(t *testing.T) {
	output, err := parseConfigValidateOutput(nil)

	require.NoError(t, err)
	assert.True(t, output.Valid)
}

func TestParseConfigValidateOutput_ValidResult(t *testing.T) {
	result := map[string]interface{}{
		"valid":  true,
		"errors": []interface{}{},
	}

	output, err := parseConfigValidateOutput(result)

	require.NoError(t, err)
	assert.True(t, output.Valid)
	assert.Empty(t, output.Errors)
}

func TestParseConfigValidateOutput_InvalidResult(t *testing.T) {
	result := map[string]interface{}{
		"valid": false,
		"errors": []interface{}{
			map[string]interface{}{
				"field":   "spec.command",
				"message": "required field is missing",
			},
		},
	}

	output, err := parseConfigValidateOutput(result)

	require.NoError(t, err)
	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec.command", output.Errors[0].Field)
}

func TestParseConfigValidateOutput_WithWarnings(t *testing.T) {
	result := map[string]interface{}{
		"valid":  true,
		"errors": []interface{}{},
		"warnings": []interface{}{
			map[string]interface{}{
				"message": "Consider using a specific version",
			},
		},
	}

	output, err := parseConfigValidateOutput(result)

	require.NoError(t, err)
	assert.True(t, output.Valid)
	require.Len(t, output.Warnings, 1)
	assert.Equal(t, "Consider using a specific version", output.Warnings[0].Message)
}

func TestParseConfigValidateOutput_WithInfraError(t *testing.T) {
	result := map[string]interface{}{
		"valid":      false,
		"infraError": "engine process crashed",
	}

	output, err := parseConfigValidateOutput(result)

	require.NoError(t, err)
	assert.False(t, output.Valid)
	assert.Equal(t, "engine process crashed", output.InfraError)
}

// -----------------------------------------------------------------------------
// Tests for configValidateInputToParams
// -----------------------------------------------------------------------------

func TestConfigValidateInputToParams(t *testing.T) {
	input := mcptypes.ConfigValidateInput{
		Spec: map[string]interface{}{
			"command": "echo",
		},
		ConfigPath: "forge.yaml",
		SpecType:   "builder",
		SpecName:   "test-builder",
	}

	params := configValidateInputToParams(input)

	require.NotNil(t, params)
	spec, ok := params["spec"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "echo", spec["command"])
	assert.Equal(t, "forge.yaml", params["configPath"])
	assert.Equal(t, "builder", params["specType"])
	assert.Equal(t, "test-builder", params["specName"])
}
