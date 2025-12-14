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

// handleConfigValidate handles the config-validate MCP tool for the parallel-builder.
// It performs recursive validation by:
// 1. Validating its own spec structure (checking that builders array exists and has engine fields)
// 2. For each builder in spec.builders:
//   - Extracts engine URI
//   - Spawns sub-builder MCP process
//   - Calls sub-builder's config-validate with builder.spec
//
// 3. Aggregating all results
func handleConfigValidate(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input mcptypes.ConfigValidateInput,
) (*mcp.CallToolResult, any, error) {
	log.Printf("parallel-builder: validating configuration")

	output := validateParallelBuilderSpec(ctx, input)

	// Return as structured MCP result
	if output.Valid {
		result, artifact := mcputil.SuccessResultWithArtifact(
			"Configuration is valid",
			output,
		)
		return result, artifact, nil
	}

	// Invalid - return errors
	result, artifact := mcputil.ErrorResultWithArtifact(
		"Configuration validation failed",
		output,
	)
	return result, artifact, nil
}

// validateParallelBuilderSpec performs the recursive validation of the parallel-builder spec.
func validateParallelBuilderSpec(ctx context.Context, input mcptypes.ConfigValidateInput) *mcptypes.ConfigValidateOutput {
	var errors []mcptypes.ValidationError
	var warnings []mcptypes.ValidationWarning

	// Step 1: Validate own spec structure
	builders, specErrors := validateOwnSpec(input.Spec)
	if len(specErrors) > 0 {
		errors = append(errors, specErrors...)
	}

	// If no builders to validate, return early
	if len(builders) == 0 {
		return &mcptypes.ConfigValidateOutput{
			Valid:    len(errors) == 0,
			Errors:   errors,
			Warnings: warnings,
		}
	}

	// Step 2 & 3: For each builder, call config-validate on the sub-builder
	caller := mcpcaller.NewCaller(Version)
	results := make([]validationResult, 0, len(builders))

	for i, builder := range builders {
		builderName := builder.Name
		if builderName == "" {
			builderName = fmt.Sprintf("builder[%d]", i)
		}

		// Validate that each builder has an engine field
		if builder.Engine == "" {
			errors = append(errors, mcptypes.ValidationError{
				Field:   fmt.Sprintf("spec.builders[%d].engine", i),
				Message: "required field is missing",
			})
			continue
		}

		// Resolve the engine URI to command and args
		command, args, err := caller.ResolveEngine(builder.Engine)
		if err != nil {
			results = append(results, validationResult{
				Ref: engineReference{
					URI:      builder.Engine,
					SpecType: "builder",
					SpecName: builderName,
				},
				Output: &mcptypes.ConfigValidateOutput{
					Valid:      false,
					InfraError: fmt.Sprintf("failed to resolve engine %s: %v", builder.Engine, err),
				},
			})
			continue
		}

		// Prepare ConfigValidateInput for the sub-builder
		subInput := mcptypes.ConfigValidateInput{
			Spec:       builder.Spec,
			ForgeSpec:  input.ForgeSpec,
			ConfigPath: input.ConfigPath,
			SpecType:   "builder",
			SpecName:   builderName,
		}

		// Convert to params map for MCP call
		params := configValidateInputToParams(subInput)

		// Call the sub-builder's config-validate tool
		result, err := caller.CallMCP(command, args, "config-validate", params)
		if err != nil {
			results = append(results, validationResult{
				Ref: engineReference{
					URI:      builder.Engine,
					SpecType: "builder",
					SpecName: builderName,
				},
				Output: &mcptypes.ConfigValidateOutput{
					Valid:      false,
					InfraError: fmt.Sprintf("failed to call config-validate on engine %s: %v", builder.Engine, err),
				},
			})
			continue
		}

		// Parse the result
		output, err := parseConfigValidateOutput(result)
		if err != nil {
			results = append(results, validationResult{
				Ref: engineReference{
					URI:      builder.Engine,
					SpecType: "builder",
					SpecName: builderName,
				},
				Output: &mcptypes.ConfigValidateOutput{
					Valid:      false,
					InfraError: fmt.Sprintf("failed to parse config-validate output from engine %s: %v", builder.Engine, err),
				},
			})
			continue
		}

		results = append(results, validationResult{
			Ref: engineReference{
				URI:      builder.Engine,
				SpecType: "builder",
				SpecName: builderName,
			},
			Output: output,
		})

		log.Printf("parallel-builder: validated sub-builder %s (%s): valid=%v", builderName, builder.Engine, output.Valid)
	}

	// Step 4: Aggregate all results
	aggregated := aggregateResults(results)

	// Merge any errors we collected during own validation
	if len(errors) > 0 {
		aggregated.Valid = false
		aggregated.Errors = append(errors, aggregated.Errors...)
	}

	// Merge warnings
	aggregated.Warnings = append(warnings, aggregated.Warnings...)

	return aggregated
}

// validateOwnSpec validates the parallel-builder's own spec structure.
// Returns the list of builders and any validation errors.
func validateOwnSpec(spec map[string]interface{}) ([]BuilderConfig, []mcptypes.ValidationError) {
	var errors []mcptypes.ValidationError

	// Handle nil spec
	if spec == nil {
		errors = append(errors, mcptypes.ValidationError{
			Field:   "spec.builders",
			Message: "required field is missing",
		})
		return nil, errors
	}

	// Check if builders field exists
	buildersRaw, ok := spec["builders"]
	if !ok {
		errors = append(errors, mcptypes.ValidationError{
			Field:   "spec.builders",
			Message: "required field is missing",
		})
		return nil, errors
	}

	// Try to convert to []interface{}
	buildersList, ok := buildersRaw.([]interface{})
	if !ok {
		errors = append(errors, mcptypes.ValidationError{
			Field:   "spec.builders",
			Message: fmt.Sprintf("expected array, got %T", buildersRaw),
		})
		return nil, errors
	}

	// Check if builders array is empty
	if len(buildersList) == 0 {
		errors = append(errors, mcptypes.ValidationError{
			Field:   "spec.builders",
			Message: "builders array cannot be empty",
		})
		return nil, errors
	}

	// Parse each builder
	builders := make([]BuilderConfig, 0, len(buildersList))
	for i, item := range buildersList {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			errors = append(errors, mcptypes.ValidationError{
				Field:   fmt.Sprintf("spec.builders[%d]", i),
				Message: fmt.Sprintf("expected object, got %T", item),
			})
			continue
		}

		// Extract name (optional)
		name, _ := itemMap["name"].(string)

		// Extract engine (required)
		engine, _ := itemMap["engine"].(string)

		// Extract spec (optional)
		var builderSpec map[string]interface{}
		if specRaw, ok := itemMap["spec"]; ok {
			if specMap, ok := specRaw.(map[string]interface{}); ok {
				builderSpec = specMap
			}
		}

		builders = append(builders, BuilderConfig{
			Name:   name,
			Engine: engine,
			Spec:   builderSpec,
		})
	}

	return builders, errors
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

// aggregateResults combines validation results from multiple sub-builders into a single output.
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
