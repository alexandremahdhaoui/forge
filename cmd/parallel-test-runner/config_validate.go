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
	"encoding/json"
	"fmt"
	"log"

	"github.com/alexandremahdhaoui/forge/internal/mcpcaller"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/alexandremahdhaoui/forge/pkg/mcputil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// handleRecursiveConfigValidate handles the config-validate MCP tool call.
// It performs recursive validation by:
// 1. Validating own spec structure (runners array with engine and name fields)
// 2. Semantic validation (primaryCoverageRunner must match a runner name if set)
// 3. Spawning each sub-runner MCP process and calling config-validate
// 4. Aggregating all results
func handleRecursiveConfigValidate(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input mcptypes.ConfigValidateInput,
) (*mcp.CallToolResult, any, error) {
	log.Printf("parallel-test-runner: validating configuration")

	output := validateParallelTestRunnerSpec(ctx, input)

	if output.Valid {
		result, artifact := mcputil.SuccessResultWithArtifact(
			"Configuration is valid",
			output,
		)
		return result, artifact, nil
	}

	result, artifact := mcputil.SuccessResultWithArtifact(
		"Configuration validation failed",
		output,
	)
	return result, artifact, nil
}

// validateParallelTestRunnerSpec performs the recursive validation of the parallel-test-runner spec.
func validateParallelTestRunnerSpec(ctx context.Context, input mcptypes.ConfigValidateInput) *mcptypes.ConfigValidateOutput {
	output := &mcptypes.ConfigValidateOutput{
		Valid:    true,
		Errors:   []mcptypes.ValidationError{},
		Warnings: []mcptypes.ValidationWarning{},
	}

	// Handle nil spec
	if input.Spec == nil {
		output.Valid = false
		output.Errors = append(output.Errors, mcptypes.ValidationError{
			Field:   "spec.runners",
			Message: "required field is missing",
		})
		return output
	}

	// Step 1: Validate own spec structure

	// Validate primaryCoverageRunner (optional, string)
	primaryCoverageRunner, err := mcptypes.ValidateString(input.Spec, "primaryCoverageRunner")
	if err != nil {
		output.Valid = false
		output.Errors = append(output.Errors, *err)
	}

	// Validate runners (REQUIRED, array of runner configurations)
	runners, runnersErr := extractRunners(input.Spec)
	if runnersErr != nil {
		output.Valid = false
		output.Errors = append(output.Errors, *runnersErr)
		return output
	}

	if len(runners) == 0 {
		output.Valid = false
		output.Errors = append(output.Errors, mcptypes.ValidationError{
			Field:   "spec.runners",
			Message: "at least one runner is required",
		})
		return output
	}

	// Step 2: Semantic validation
	// If primaryCoverageRunner is set, it must match one of the runner names
	if primaryCoverageRunner != "" {
		found := false
		for _, runner := range runners {
			if runner.Name == primaryCoverageRunner {
				found = true
				break
			}
		}
		if !found {
			output.Valid = false
			output.Errors = append(output.Errors, mcptypes.ValidationError{
				Field:   "spec.primaryCoverageRunner",
				Message: fmt.Sprintf("primaryCoverageRunner '%s' does not match any runner name", primaryCoverageRunner),
			})
		}
	}

	// Validate each runner has required fields
	runnerNames := make(map[string]bool)
	for i, runner := range runners {
		// Validate engine (REQUIRED)
		if runner.Engine == "" {
			output.Valid = false
			output.Errors = append(output.Errors, mcptypes.ValidationError{
				Field:   fmt.Sprintf("spec.runners[%d].engine", i),
				Message: "required field is missing",
			})
		}

		// Validate name (REQUIRED)
		if runner.Name == "" {
			output.Valid = false
			output.Errors = append(output.Errors, mcptypes.ValidationError{
				Field:   fmt.Sprintf("spec.runners[%d].name", i),
				Message: "required field is missing",
			})
		} else {
			// Check for duplicate names
			if runnerNames[runner.Name] {
				output.Valid = false
				output.Errors = append(output.Errors, mcptypes.ValidationError{
					Field:   fmt.Sprintf("spec.runners[%d].name", i),
					Message: fmt.Sprintf("duplicate runner name '%s'", runner.Name),
				})
			}
			runnerNames[runner.Name] = true
		}
	}

	// If we have validation errors at this point, return early
	if !output.Valid {
		return output
	}

	// Step 3 & 4: For each runner, spawn sub-runner MCP process and call config-validate
	caller := mcpcaller.NewCaller(Version)
	results := make([]validationResult, 0, len(runners))

	for i, runner := range runners {
		// Resolve engine URI to command and args
		command, args, resolveErr := caller.ResolveEngine(runner.Engine)
		if resolveErr != nil {
			results = append(results, validationResult{
				Ref: engineReference{
					URI:      runner.Engine,
					SpecType: "test-runner",
					SpecName: fmt.Sprintf("%s.runners[%d]", input.SpecName, i),
				},
				Output: &mcptypes.ConfigValidateOutput{
					Valid:      false,
					InfraError: fmt.Sprintf("failed to resolve engine %s: %v", runner.Engine, resolveErr),
				},
			})
			continue
		}

		// Prepare ConfigValidateInput for the sub-runner
		subInput := mcptypes.ConfigValidateInput{
			Spec:       runner.Spec,
			ForgeSpec:  input.ForgeSpec,
			ConfigPath: input.ConfigPath,
			SpecType:   "test-runner",
			SpecName:   fmt.Sprintf("%s.runners[%d]", input.SpecName, i),
		}

		// Convert to params map for MCP call
		params := configValidateInputToParams(subInput)

		// Call the sub-runner's config-validate tool
		resp, callErr := caller.CallMCP(command, args, "config-validate", params)
		if callErr != nil {
			results = append(results, validationResult{
				Ref: engineReference{
					URI:      runner.Engine,
					SpecType: "test-runner",
					SpecName: fmt.Sprintf("%s.runners[%d]", input.SpecName, i),
				},
				Output: &mcptypes.ConfigValidateOutput{
					Valid:      false,
					InfraError: fmt.Sprintf("failed to call config-validate on engine %s: %v", runner.Engine, callErr),
				},
			})
			continue
		}

		// Parse the result
		subOutput, parseErr := parseConfigValidateOutput(resp)
		if parseErr != nil {
			results = append(results, validationResult{
				Ref: engineReference{
					URI:      runner.Engine,
					SpecType: "test-runner",
					SpecName: fmt.Sprintf("%s.runners[%d]", input.SpecName, i),
				},
				Output: &mcptypes.ConfigValidateOutput{
					Valid:      false,
					InfraError: fmt.Sprintf("failed to parse config-validate output from engine %s: %v", runner.Engine, parseErr),
				},
			})
			continue
		}

		results = append(results, validationResult{
			Ref: engineReference{
				URI:      runner.Engine,
				SpecType: "test-runner",
				SpecName: fmt.Sprintf("%s.runners[%d]", input.SpecName, i),
			},
			Output: subOutput,
		})

		log.Printf("parallel-test-runner: validated sub-runner %s (%s): valid=%v", runner.Name, runner.Engine, subOutput.Valid)
	}

	// Step 4: Aggregate all results
	aggregated := aggregateResults(results)

	// Merge any errors we collected during own validation
	if len(output.Errors) > 0 {
		aggregated.Valid = false
		aggregated.Errors = append(output.Errors, aggregated.Errors...)
	}

	// Merge warnings
	aggregated.Warnings = append(output.Warnings, aggregated.Warnings...)

	return aggregated
}

// runnerSpec represents a single runner configuration extracted from the spec.
type runnerSpec struct {
	Name   string
	Engine string
	Spec   map[string]interface{}
}

// extractRunners extracts runners from the input spec map.
func extractRunners(spec map[string]interface{}) ([]runnerSpec, *mcptypes.ValidationError) {
	runnersRaw, ok := spec["runners"]
	if !ok {
		return nil, &mcptypes.ValidationError{
			Field:   "spec.runners",
			Message: "required field is missing",
		}
	}

	// Try to convert to []interface{}
	runnersList, ok := runnersRaw.([]interface{})
	if !ok {
		return nil, &mcptypes.ValidationError{
			Field:   "spec.runners",
			Message: fmt.Sprintf("expected array, got %T", runnersRaw),
		}
	}

	runners := make([]runnerSpec, 0, len(runnersList))
	for i, item := range runnersList {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			return nil, &mcptypes.ValidationError{
				Field:   fmt.Sprintf("spec.runners[%d]", i),
				Message: fmt.Sprintf("expected object, got %T", item),
			}
		}

		// Extract engine field
		engine, _ := itemMap["engine"].(string)

		// Extract name field
		name, _ := itemMap["name"].(string)

		// Extract spec field
		var runnerSpecMap map[string]interface{}
		if specRaw, ok := itemMap["spec"]; ok {
			if specMap, ok := specRaw.(map[string]interface{}); ok {
				runnerSpecMap = specMap
			}
		}

		runners = append(runners, runnerSpec{
			Name:   name,
			Engine: engine,
			Spec:   runnerSpecMap,
		})
	}

	return runners, nil
}

// engineReference represents a reference to an engine for validation result tracking.
type engineReference struct {
	URI      string
	SpecType string
	SpecName string
}

// validationResult pairs an engine reference with its validation output.
type validationResult struct {
	Ref    engineReference
	Output *mcptypes.ConfigValidateOutput
}

// aggregateResults combines validation results from multiple sub-runners into a single output.
func aggregateResults(results []validationResult) *mcptypes.ConfigValidateOutput {
	combined := &mcptypes.ConfigValidateOutput{
		Valid:    true,
		Errors:   []mcptypes.ValidationError{},
		Warnings: []mcptypes.ValidationWarning{},
	}

	for _, r := range results {
		if r.Output == nil {
			continue
		}

		// Handle infrastructure errors
		if r.Output.InfraError != "" {
			combined.Valid = false
			combined.Errors = append(combined.Errors, mcptypes.ValidationError{
				Field:    "",
				Message:  r.Output.InfraError,
				Engine:   r.Ref.URI,
				SpecType: r.Ref.SpecType,
				SpecName: r.Ref.SpecName,
			})
		}

		// Handle validation errors
		if !r.Output.Valid {
			combined.Valid = false
			for _, err := range r.Output.Errors {
				// Set engine context if not already set
				if err.Engine == "" {
					err.Engine = r.Ref.URI
				}
				// Set spec context
				err.SpecType = r.Ref.SpecType
				err.SpecName = r.Ref.SpecName
				combined.Errors = append(combined.Errors, err)
			}
		}

		// Collect all warnings
		combined.Warnings = append(combined.Warnings, r.Output.Warnings...)
	}

	return combined
}

// configValidateInputToParams converts ConfigValidateInput to map[string]any for MCP calls.
func configValidateInputToParams(input mcptypes.ConfigValidateInput) map[string]any {
	data, err := json.Marshal(input)
	if err != nil {
		return nil
	}

	var params map[string]any
	if err := json.Unmarshal(data, &params); err != nil {
		return nil
	}

	return params
}

// parseConfigValidateOutput parses the MCP tool result into a ConfigValidateOutput.
func parseConfigValidateOutput(result interface{}) (*mcptypes.ConfigValidateOutput, error) {
	if result == nil {
		// No result - assume valid (engine may not have implemented config-validate)
		return &mcptypes.ConfigValidateOutput{
			Valid: true,
		}, nil
	}

	// Convert result to JSON and back to ConfigValidateOutput
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var output mcptypes.ConfigValidateOutput
	if err := json.Unmarshal(resultBytes, &output); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &output, nil
}
