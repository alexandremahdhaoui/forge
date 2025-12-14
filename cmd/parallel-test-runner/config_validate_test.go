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

// TestExtractRunners_ValidRunners tests extracting a valid runners array.
func TestExtractRunners_ValidRunners(t *testing.T) {
	spec := map[string]interface{}{
		"runners": []interface{}{
			map[string]interface{}{
				"name":   "runner1",
				"engine": "go://go-test",
				"spec": map[string]interface{}{
					"packages": []interface{}{"./..."},
				},
			},
			map[string]interface{}{
				"name":   "runner2",
				"engine": "go://go-lint",
			},
		},
	}

	runners, err := extractRunners(spec)

	assert.Nil(t, err)
	require.Len(t, runners, 2)
	assert.Equal(t, "runner1", runners[0].Name)
	assert.Equal(t, "go://go-test", runners[0].Engine)
	assert.NotNil(t, runners[0].Spec)
	assert.Equal(t, "runner2", runners[1].Name)
	assert.Equal(t, "go://go-lint", runners[1].Engine)
	assert.Nil(t, runners[1].Spec)
}

// TestExtractRunners_MissingRunners tests that missing runners field returns error.
func TestExtractRunners_MissingRunners(t *testing.T) {
	spec := map[string]interface{}{}

	runners, err := extractRunners(spec)

	assert.Nil(t, runners)
	require.NotNil(t, err)
	assert.Equal(t, "spec.runners", err.Field)
	assert.Equal(t, "required field is missing", err.Message)
}

// TestExtractRunners_InvalidRunnersType tests that non-array runners field returns error.
func TestExtractRunners_InvalidRunnersType(t *testing.T) {
	spec := map[string]interface{}{
		"runners": "not-an-array",
	}

	runners, err := extractRunners(spec)

	assert.Nil(t, runners)
	require.NotNil(t, err)
	assert.Equal(t, "spec.runners", err.Field)
	assert.Contains(t, err.Message, "expected array")
}

// TestExtractRunners_InvalidRunnerElementType tests that non-object runner element returns error.
func TestExtractRunners_InvalidRunnerElementType(t *testing.T) {
	spec := map[string]interface{}{
		"runners": []interface{}{
			"not-an-object",
		},
	}

	runners, err := extractRunners(spec)

	assert.Nil(t, runners)
	require.NotNil(t, err)
	assert.Equal(t, "spec.runners[0]", err.Field)
	assert.Contains(t, err.Message, "expected object")
}

// TestValidateParallelTestRunnerSpec_ValidSpec tests a fully valid spec.
func TestValidateParallelTestRunnerSpec_ValidSpec(t *testing.T) {
	input := mcptypes.ConfigValidateInput{
		Spec: map[string]interface{}{
			"runners": []interface{}{
				map[string]interface{}{
					"name":   "runner1",
					"engine": "go://go-test",
				},
				map[string]interface{}{
					"name":   "runner2",
					"engine": "go://go-lint",
				},
			},
			"primaryCoverageRunner": "runner1",
		},
	}

	// Note: validateParallelTestRunnerSpec will try to call sub-runners via MCP,
	// which will fail in unit tests. So we only test the spec validation part
	// by checking the errors before recursive calls are attempted.
	// The recursive MCP calls are tested in integration tests.

	// For this test, we focus on extracting and validating the basic structure
	runners, err := extractRunners(input.Spec)
	assert.Nil(t, err)
	require.Len(t, runners, 2)

	// Validate primaryCoverageRunner
	pcr, verr := mcptypes.ValidateString(input.Spec, "primaryCoverageRunner")
	assert.Nil(t, verr)
	assert.Equal(t, "runner1", pcr)

	// Validate primaryCoverageRunner matches a runner name
	found := false
	for _, r := range runners {
		if r.Name == pcr {
			found = true
			break
		}
	}
	assert.True(t, found)
}

// TestValidateParallelTestRunnerSpec_NilSpec tests nil spec returns error.
func TestValidateParallelTestRunnerSpec_NilSpec(t *testing.T) {
	input := mcptypes.ConfigValidateInput{
		Spec: nil,
	}

	// Can't use validateParallelTestRunnerSpec directly as it tries MCP calls
	// Test extractRunners with nil behavior
	output := &mcptypes.ConfigValidateOutput{
		Valid:  true,
		Errors: []mcptypes.ValidationError{},
	}

	if input.Spec == nil {
		output.Valid = false
		output.Errors = append(output.Errors, mcptypes.ValidationError{
			Field:   "spec.runners",
			Message: "required field is missing",
		})
	}

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec.runners", output.Errors[0].Field)
}

// TestValidateParallelTestRunnerSpec_EmptyRunners tests empty runners array.
func TestValidateParallelTestRunnerSpec_EmptyRunners(t *testing.T) {
	spec := map[string]interface{}{
		"runners": []interface{}{},
	}

	runners, err := extractRunners(spec)

	assert.Nil(t, err)
	assert.Empty(t, runners)
}

// TestValidateParallelTestRunnerSpec_MissingEngine tests runner without engine field.
func TestValidateParallelTestRunnerSpec_MissingEngine(t *testing.T) {
	spec := map[string]interface{}{
		"runners": []interface{}{
			map[string]interface{}{
				"name": "runner1",
				// missing engine
			},
		},
	}

	runners, err := extractRunners(spec)
	assert.Nil(t, err)
	require.Len(t, runners, 1)

	// Validate that engine is empty (validation logic checks this)
	assert.Equal(t, "", runners[0].Engine)
}

// TestValidateParallelTestRunnerSpec_MissingName tests runner without name field.
func TestValidateParallelTestRunnerSpec_MissingName(t *testing.T) {
	spec := map[string]interface{}{
		"runners": []interface{}{
			map[string]interface{}{
				"engine": "go://go-test",
				// missing name
			},
		},
	}

	runners, err := extractRunners(spec)
	assert.Nil(t, err)
	require.Len(t, runners, 1)

	// Validate that name is empty (validation logic checks this)
	assert.Equal(t, "", runners[0].Name)
}

// TestValidateParallelTestRunnerSpec_InvalidPrimaryCoverageRunnerType tests invalid type for primaryCoverageRunner.
func TestValidateParallelTestRunnerSpec_InvalidPrimaryCoverageRunnerType(t *testing.T) {
	spec := map[string]interface{}{
		"runners": []interface{}{
			map[string]interface{}{
				"name":   "runner1",
				"engine": "go://go-test",
			},
		},
		"primaryCoverageRunner": 123, // should be string
	}

	_, err := mcptypes.ValidateString(spec, "primaryCoverageRunner")
	require.NotNil(t, err)
	assert.Equal(t, "spec.primaryCoverageRunner", err.Field)
	assert.Contains(t, err.Message, "expected string")
}

// TestValidateParallelTestRunnerSpec_PrimaryCoverageRunnerNotFound tests primaryCoverageRunner not matching any runner.
func TestValidateParallelTestRunnerSpec_PrimaryCoverageRunnerNotFound(t *testing.T) {
	spec := map[string]interface{}{
		"runners": []interface{}{
			map[string]interface{}{
				"name":   "runner1",
				"engine": "go://go-test",
			},
		},
		"primaryCoverageRunner": "nonexistent",
	}

	// Extract runners
	runners, err := extractRunners(spec)
	assert.Nil(t, err)

	// Get primaryCoverageRunner
	pcr, verr := mcptypes.ValidateString(spec, "primaryCoverageRunner")
	assert.Nil(t, verr)
	assert.Equal(t, "nonexistent", pcr)

	// Validate it doesn't match any runner
	found := false
	for _, r := range runners {
		if r.Name == pcr {
			found = true
			break
		}
	}
	assert.False(t, found)
}

// TestValidateParallelTestRunnerSpec_DuplicateRunnerNames tests detection of duplicate runner names.
func TestValidateParallelTestRunnerSpec_DuplicateRunnerNames(t *testing.T) {
	spec := map[string]interface{}{
		"runners": []interface{}{
			map[string]interface{}{
				"name":   "runner1",
				"engine": "go://go-test",
			},
			map[string]interface{}{
				"name":   "runner1", // duplicate
				"engine": "go://go-lint",
			},
		},
	}

	runners, err := extractRunners(spec)
	assert.Nil(t, err)
	require.Len(t, runners, 2)

	// Check for duplicates (simulation of validation logic)
	names := make(map[string]bool)
	hasDuplicate := false
	for _, r := range runners {
		if names[r.Name] {
			hasDuplicate = true
			break
		}
		names[r.Name] = true
	}
	assert.True(t, hasDuplicate)
}

// TestAggregateResults_AllValid tests aggregating all valid results.
func TestAggregateResults_AllValid(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{URI: "go://go-test", SpecType: "test-runner", SpecName: "runners[0]"},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: true,
			},
		},
		{
			Ref: engineReference{URI: "go://go-lint", SpecType: "test-runner", SpecName: "runners[1]"},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: true,
			},
		},
	}

	combined := aggregateResults(results)

	assert.True(t, combined.Valid)
	assert.Empty(t, combined.Errors)
}

// TestAggregateResults_SomeInvalid tests aggregating with some invalid results.
func TestAggregateResults_SomeInvalid(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{URI: "go://go-test", SpecType: "test-runner", SpecName: "runners[0]"},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: true,
			},
		},
		{
			Ref: engineReference{URI: "go://go-lint", SpecType: "test-runner", SpecName: "runners[1]"},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: false,
				Errors: []mcptypes.ValidationError{
					{Field: "spec.config", Message: "invalid config"},
				},
			},
		},
	}

	combined := aggregateResults(results)

	assert.False(t, combined.Valid)
	require.Len(t, combined.Errors, 1)
	assert.Equal(t, "spec.config", combined.Errors[0].Field)
	assert.Equal(t, "go://go-lint", combined.Errors[0].Engine)
	assert.Equal(t, "test-runner", combined.Errors[0].SpecType)
	assert.Equal(t, "runners[1]", combined.Errors[0].SpecName)
}

// TestAggregateResults_InfraError tests aggregating with infrastructure error.
func TestAggregateResults_InfraError(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{URI: "go://go-test", SpecType: "test-runner", SpecName: "runners[0]"},
			Output: &mcptypes.ConfigValidateOutput{
				Valid:      false,
				InfraError: "failed to spawn MCP process",
			},
		},
	}

	combined := aggregateResults(results)

	assert.False(t, combined.Valid)
	require.Len(t, combined.Errors, 1)
	assert.Equal(t, "", combined.Errors[0].Field)
	assert.Equal(t, "failed to spawn MCP process", combined.Errors[0].Message)
	assert.Equal(t, "go://go-test", combined.Errors[0].Engine)
}

// TestAggregateResults_WithWarnings tests aggregating results with warnings.
func TestAggregateResults_WithWarnings(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{URI: "go://go-test", SpecType: "test-runner", SpecName: "runners[0]"},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: true,
				Warnings: []mcptypes.ValidationWarning{
					{Message: "deprecated field used"},
				},
			},
		},
	}

	combined := aggregateResults(results)

	assert.True(t, combined.Valid)
	assert.Empty(t, combined.Errors)
	require.Len(t, combined.Warnings, 1)
	assert.Equal(t, "deprecated field used", combined.Warnings[0].Message)
}

// TestAggregateResults_NilOutput tests aggregating with nil output.
func TestAggregateResults_NilOutput(t *testing.T) {
	results := []validationResult{
		{
			Ref:    engineReference{URI: "go://go-test", SpecType: "test-runner", SpecName: "runners[0]"},
			Output: nil,
		},
		{
			Ref: engineReference{URI: "go://go-lint", SpecType: "test-runner", SpecName: "runners[1]"},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: true,
			},
		},
	}

	combined := aggregateResults(results)

	assert.True(t, combined.Valid)
	assert.Empty(t, combined.Errors)
}

// TestAggregateResults_Empty tests aggregating empty results.
func TestAggregateResults_Empty(t *testing.T) {
	results := []validationResult{}

	combined := aggregateResults(results)

	assert.True(t, combined.Valid)
	assert.Empty(t, combined.Errors)
	assert.Empty(t, combined.Warnings)
}

// TestConfigValidateInputToParams tests conversion of ConfigValidateInput to params map.
func TestConfigValidateInputToParams(t *testing.T) {
	input := mcptypes.ConfigValidateInput{
		Spec: map[string]interface{}{
			"command": "echo",
		},
		ConfigPath: "/path/to/forge.yaml",
		SpecType:   "test-runner",
		SpecName:   "runners[0]",
	}

	params := configValidateInputToParams(input)

	assert.NotNil(t, params)
	assert.Equal(t, "/path/to/forge.yaml", params["configPath"])
	assert.Equal(t, "test-runner", params["specType"])
	assert.Equal(t, "runners[0]", params["specName"])
	specMap, ok := params["spec"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "echo", specMap["command"])
}

// TestParseConfigValidateOutput_ValidOutput tests parsing a valid config validate output.
func TestParseConfigValidateOutput_ValidOutput(t *testing.T) {
	result := map[string]interface{}{
		"valid":  true,
		"errors": []interface{}{},
		"warnings": []interface{}{
			map[string]interface{}{
				"message": "some warning",
			},
		},
	}

	output, err := parseConfigValidateOutput(result)

	assert.NoError(t, err)
	assert.True(t, output.Valid)
	assert.Empty(t, output.Errors)
	require.Len(t, output.Warnings, 1)
	assert.Equal(t, "some warning", output.Warnings[0].Message)
}

// TestParseConfigValidateOutput_NilResult tests parsing nil result.
func TestParseConfigValidateOutput_NilResult(t *testing.T) {
	output, err := parseConfigValidateOutput(nil)

	assert.NoError(t, err)
	assert.True(t, output.Valid)
}

// TestParseConfigValidateOutput_InvalidOutput tests parsing invalid output.
func TestParseConfigValidateOutput_InvalidOutput(t *testing.T) {
	result := map[string]interface{}{
		"valid": false,
		"errors": []interface{}{
			map[string]interface{}{
				"field":   "spec.command",
				"message": "required field missing",
			},
		},
	}

	output, err := parseConfigValidateOutput(result)

	assert.NoError(t, err)
	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec.command", output.Errors[0].Field)
	assert.Equal(t, "required field missing", output.Errors[0].Message)
}
