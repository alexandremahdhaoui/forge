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
	"fmt"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/alexandremahdhaoui/forge/pkg/mcputil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// handleConfigValidate handles the config-validate MCP tool call.
// It validates the testenv-helm-install spec fields:
//   - charts ([]object, optional) - array of chart specifications
//
// Each chart is validated for:
//   - name (string, REQUIRED) - chart release name
//   - chart (string, optional)
//   - repo (string, optional)
//   - version (string, optional)
//   - namespace (string, optional)
//   - valuesFile (string, optional)
func handleConfigValidate(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input mcptypes.ConfigValidateInput,
) (*mcp.CallToolResult, any, error) {
	output := validateHelmInstallSpec(input.Spec)

	// Return result with appropriate success/error status
	if output.Valid {
		result, artifact := mcputil.SuccessResultWithArtifact(
			"testenv-helm-install configuration is valid",
			output,
		)
		return result, artifact, nil
	}

	// Validation failed
	result, artifact := mcputil.ErrorResultWithArtifact(
		fmt.Sprintf("testenv-helm-install configuration validation failed with %d error(s)", len(output.Errors)),
		output,
	)
	return result, artifact, nil
}

// validateHelmInstallSpec validates the testenv-helm-install spec fields.
// This is extracted for easier unit testing.
func validateHelmInstallSpec(spec map[string]interface{}) *mcptypes.ConfigValidateOutput {
	output := &mcptypes.ConfigValidateOutput{
		Valid:    true,
		Errors:   []mcptypes.ValidationError{},
		Warnings: []mcptypes.ValidationWarning{},
	}

	if spec == nil {
		return output
	}

	// Validate charts (array, optional)
	chartsVal, ok := spec["charts"]
	if !ok {
		// charts is optional, spec is valid without it
		return output
	}

	// If charts is present, it must be an array
	charts, ok := chartsVal.([]interface{})
	if !ok {
		output.Errors = append(output.Errors, mcptypes.ValidationError{
			Field:   "spec.charts",
			Message: fmt.Sprintf("expected array, got %T", chartsVal),
		})
		output.Valid = false
		return output
	}

	// Validate each chart in the array
	for i, chartVal := range charts {
		validateChart(output, chartVal, i)
	}

	return output
}

// validateChart validates a single chart specification within the charts array.
func validateChart(output *mcptypes.ConfigValidateOutput, chartVal interface{}, index int) {
	chart, ok := chartVal.(map[string]interface{})
	if !ok {
		output.Errors = append(output.Errors, mcptypes.ValidationError{
			Field:   fmt.Sprintf("spec.charts[%d]", index),
			Message: fmt.Sprintf("expected object, got %T", chartVal),
		})
		output.Valid = false
		return
	}

	// Validate name (string, REQUIRED)
	name, err := validateChartString(chart, "name", index)
	if err != nil {
		output.Errors = append(output.Errors, *err)
		output.Valid = false
	} else if name == "" {
		output.Errors = append(output.Errors, mcptypes.ValidationError{
			Field:   fmt.Sprintf("spec.charts[%d].name", index),
			Message: "required field is missing",
		})
		output.Valid = false
	}

	// Validate chart (string, optional)
	if _, err := validateChartString(chart, "chart", index); err != nil {
		output.Errors = append(output.Errors, *err)
		output.Valid = false
	}

	// Validate repo (string, optional)
	if _, err := validateChartString(chart, "repo", index); err != nil {
		output.Errors = append(output.Errors, *err)
		output.Valid = false
	}

	// Validate version (string, optional)
	if _, err := validateChartString(chart, "version", index); err != nil {
		output.Errors = append(output.Errors, *err)
		output.Valid = false
	}

	// Validate namespace (string, optional)
	if _, err := validateChartString(chart, "namespace", index); err != nil {
		output.Errors = append(output.Errors, *err)
		output.Valid = false
	}

	// Validate valuesFile (string, optional)
	if _, err := validateChartString(chart, "valuesFile", index); err != nil {
		output.Errors = append(output.Errors, *err)
		output.Valid = false
	}
}

// validateChartString validates that a field in a chart is a string or absent.
// Returns the string value and nil error if field is absent or valid.
// Returns empty string and ValidationError if field has wrong type.
func validateChartString(chart map[string]interface{}, field string, index int) (string, *mcptypes.ValidationError) {
	val, ok := chart[field]
	if !ok {
		return "", nil // Field is absent, which is OK for optional fields
	}

	str, ok := val.(string)
	if !ok {
		return "", &mcptypes.ValidationError{
			Field:   fmt.Sprintf("spec.charts[%d].%s", index, field),
			Message: fmt.Sprintf("expected string, got %T", val),
		}
	}

	return str, nil
}
