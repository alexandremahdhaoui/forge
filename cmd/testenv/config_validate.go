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
	"strings"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (c *testenvCommands) handleConfigValidate(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input mcptypes.ConfigValidateInput,
) (*mcp.CallToolResult, any, error) {
	log.Printf("testenv: validating configuration")

	output := c.validateTestenvSpec(ctx, input)

	msg := "Configuration is valid"
	if !output.Valid {
		msg = fmt.Sprintf("Configuration validation failed with %d error(s)", len(output.Errors))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}, output, nil
}

func (c *testenvCommands) validateTestenvSpec(ctx context.Context, input mcptypes.ConfigValidateInput) *mcptypes.ConfigValidateOutput {
	var errors []mcptypes.ValidationError
	var warnings []mcptypes.ValidationWarning

	if input.ForgeSpec == nil {
		log.Printf("testenv: forgeSpec is nil, performing basic validation only")
	}

	var subengines []forge.TestenvEngineSpec

	if input.ForgeSpec != nil {
		subengines = extractSubenginesFromForgeSpec(input.ForgeSpec, input.SpecName)
	}

	if len(subengines) == 0 && len(input.Spec) > 0 {
		extracted, extractErr := extractSubenginesFromSpec(input.Spec)
		if extractErr != nil {
			errors = append(errors, *extractErr)
		} else {
			subengines = extracted
		}
	}

	if len(subengines) == 0 {
		log.Printf("testenv: no subengines to validate")
		return &mcptypes.ConfigValidateOutput{
			Valid:    len(errors) == 0,
			Errors:   errors,
			Warnings: warnings,
		}
	}

	results := make([]validationResult, 0, len(subengines))

	for i, subengine := range subengines {
		if subengine.Engine == "" {
			errors = append(errors, mcptypes.ValidationError{
				Field:   fmt.Sprintf("testenv[%d].engine", i),
				Message: "subengine must have an engine field",
			})
			continue
		}

		subengineSpec := getSubengineConfig(subengine.Engine, subengine.Spec, input.ForgeSpec)

		subInput := mcptypes.ConfigValidateInput{
			Spec:       subengineSpec,
			ForgeSpec:  input.ForgeSpec,
			ConfigPath: input.ConfigPath,
			SpecType:   "testenv-subengine",
			SpecName:   fmt.Sprintf("%s[%d]", input.SpecName, i),
		}

		params := configValidateInputToParams(subInput)

		result, err := c.callEngine(subengine.Engine, "config-validate", params)
		if err != nil {
			results = append(results, validationResult{
				Ref: engineReference{
					URI:      subengine.Engine,
					SpecType: "testenv-subengine",
					SpecName: fmt.Sprintf("%s[%d]", input.SpecName, i),
				},
				Output: &mcptypes.ConfigValidateOutput{
					Valid:      false,
					InfraError: fmt.Sprintf("failed to call config-validate on engine %s: %v", subengine.Engine, err),
				},
			})
			continue
		}

		output, err := parseConfigValidateOutput(result)
		if err != nil {
			results = append(results, validationResult{
				Ref: engineReference{
					URI:      subengine.Engine,
					SpecType: "testenv-subengine",
					SpecName: fmt.Sprintf("%s[%d]", input.SpecName, i),
				},
				Output: &mcptypes.ConfigValidateOutput{
					Valid:      false,
					InfraError: fmt.Sprintf("failed to parse config-validate output from engine %s: %v", subengine.Engine, err),
				},
			})
			continue
		}

		results = append(results, validationResult{
			Ref: engineReference{
				URI:      subengine.Engine,
				SpecType: "testenv-subengine",
				SpecName: fmt.Sprintf("%s[%d]", input.SpecName, i),
			},
			Output: output,
		})

		log.Printf("testenv: validated subengine %s: valid=%v", subengine.Engine, output.Valid)
	}

	aggregated := aggregateResults(results)

	if len(errors) > 0 {
		aggregated.Valid = false
		aggregated.Errors = append(errors, aggregated.Errors...)
	}

	aggregated.Warnings = append(warnings, aggregated.Warnings...)

	return aggregated
}

func extractSubenginesFromForgeSpec(forgeSpec *forge.Spec, stageName string) []forge.TestenvEngineSpec {
	if forgeSpec == nil {
		return nil
	}

	var testSpec *forge.TestSpec
	for i := range forgeSpec.Test {
		if forgeSpec.Test[i].Name == stageName {
			testSpec = &forgeSpec.Test[i]
			break
		}
	}

	if testSpec == nil {
		return nil
	}

	testenvURI := testSpec.Testenv
	if testenvURI == "" {
		return nil
	}

	if strings.HasPrefix(testenvURI, "alias://") {
		alias := strings.TrimPrefix(testenvURI, "alias://")
		for i := range forgeSpec.Engines {
			if forgeSpec.Engines[i].Alias == alias && forgeSpec.Engines[i].Type == forge.TestenvEngineConfigType {
				return forgeSpec.Engines[i].Testenv
			}
		}
	}

	return nil
}

func extractSubenginesFromSpec(spec map[string]interface{}) ([]forge.TestenvEngineSpec, *mcptypes.ValidationError) {
	subenginesRaw, ok := spec["subengines"]
	if !ok {
		return nil, nil
	}

	subenginesList, ok := subenginesRaw.([]interface{})
	if !ok {
		return nil, &mcptypes.ValidationError{
			Field:   "spec.subengines",
			Message: fmt.Sprintf("expected array, got %T", subenginesRaw),
		}
	}

	subengines := make([]forge.TestenvEngineSpec, 0, len(subenginesList))
	for i, item := range subenginesList {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			return nil, &mcptypes.ValidationError{
				Field:   fmt.Sprintf("spec.subengines[%d]", i),
				Message: fmt.Sprintf("expected object, got %T", item),
			}
		}

		engine, _ := itemMap["engine"].(string)

		var subSpec map[string]interface{}
		if specRaw, ok := itemMap["spec"]; ok {
			if specMap, ok := specRaw.(map[string]interface{}); ok {
				subSpec = specMap
			}
		}

		deferTemplates, _ := itemMap["deferTemplates"].(bool)

		subengines = append(subengines, forge.TestenvEngineSpec{
			Engine:         engine,
			Spec:           subSpec,
			DeferTemplates: deferTemplates,
		})
	}

	return subengines, nil
}

func getSubengineConfig(engineURI string, subengineSpec map[string]interface{}, forgeSpec *forge.Spec) map[string]interface{} {
	if forgeSpec == nil {
		return subengineSpec
	}

	switch {
	case engineURI == "forge://testenv-helm-install":
		return subengineSpec

	case strings.HasPrefix(engineURI, "alias://"):
		alias := strings.TrimPrefix(engineURI, "alias://")
		for _, ec := range forgeSpec.Engines {
			if ec.Alias == alias {
				return subengineSpec
			}
		}
		return subengineSpec

	default:
		return subengineSpec
	}
}

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

func parseConfigValidateOutput(result interface{}) (*mcptypes.ConfigValidateOutput, error) {
	if result == nil {
		return &mcptypes.ConfigValidateOutput{
			Valid: true,
		}, nil
	}

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

type engineReference struct {
	URI      string
	SpecType string
	SpecName string
}

type validationResult struct {
	Ref    engineReference
	Output *mcptypes.ConfigValidateOutput
}

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

		basePath := []string{"testenv", r.Ref.SpecName, "spec"}

		if r.Output.InfraError != "" {
			combined.Valid = false
			combined.Errors = append(combined.Errors, mcptypes.ValidationError{
				Field:    "",
				Message:  r.Output.InfraError,
				Engine:   r.Ref.URI,
				SpecType: r.Ref.SpecType,
				SpecName: r.Ref.SpecName,
				Path:     basePath,
			})
		}

		if !r.Output.Valid {
			combined.Valid = false
			for _, err := range r.Output.Errors {
				if err.Engine == "" {
					err.Engine = r.Ref.URI
				}
				err.SpecType = r.Ref.SpecType
				err.SpecName = r.Ref.SpecName
				if len(err.Path) == 0 {
					err.Path = basePath
				}
				combined.Errors = append(combined.Errors, err)
			}
		}

		combined.Warnings = append(combined.Warnings, r.Output.Warnings...)
	}

	return combined
}
