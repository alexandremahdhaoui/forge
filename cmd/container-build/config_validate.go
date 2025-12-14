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

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// handleConfigValidate validates container-build engine configuration.
// It checks the spec fields:
//   - dockerfile (string, optional) - path to Dockerfile
//   - context (string, optional) - build context path
//   - buildArgs (map[string]string, optional) - build arguments
//   - tags ([]string, optional) - image tags
//   - target (string, optional) - build target stage
//   - push (bool, optional) - whether to push image
//   - registry (string, optional) - registry URL
//
// Returns ConfigValidateOutput with valid=true if spec is valid,
// or with errors for invalid field types.
func handleConfigValidate(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input mcptypes.ConfigValidateInput,
) (*mcp.CallToolResult, any, error) {
	output := validateContainerBuildSpec(input.Spec)

	// Return as structured MCP result
	if output.Valid {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Configuration is valid"},
			},
		}, output, nil
	}

	// Invalid - return errors
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Configuration validation failed"},
		},
		IsError: true,
	}, output, nil
}

// validateContainerBuildSpec performs the actual validation of the spec.
func validateContainerBuildSpec(spec map[string]interface{}) *mcptypes.ConfigValidateOutput {
	var errors []mcptypes.ValidationError

	// If spec is nil or empty, it's valid (no configuration to validate)
	if len(spec) == 0 {
		return &mcptypes.ConfigValidateOutput{
			Valid: true,
		}
	}

	// Validate dockerfile field (optional, must be string if present)
	if _, verr := mcptypes.ValidateString(spec, "dockerfile"); verr != nil {
		errors = append(errors, *verr)
	}

	// Validate context field (optional, must be string if present)
	if _, verr := mcptypes.ValidateString(spec, "context"); verr != nil {
		errors = append(errors, *verr)
	}

	// Validate buildArgs field (optional, must be map[string]string if present)
	if _, verr := mcptypes.ValidateStringMap(spec, "buildArgs"); verr != nil {
		errors = append(errors, *verr)
	}

	// Validate tags field (optional, must be []string if present)
	if _, verr := mcptypes.ValidateStringSlice(spec, "tags"); verr != nil {
		errors = append(errors, *verr)
	}

	// Validate target field (optional, must be string if present)
	if _, verr := mcptypes.ValidateString(spec, "target"); verr != nil {
		errors = append(errors, *verr)
	}

	// Validate push field (optional, must be bool if present)
	if _, verr := mcptypes.ValidateBool(spec, "push"); verr != nil {
		errors = append(errors, *verr)
	}

	// Validate registry field (optional, must be string if present)
	if _, verr := mcptypes.ValidateString(spec, "registry"); verr != nil {
		errors = append(errors, *verr)
	}

	// Return result
	if len(errors) > 0 {
		return &mcptypes.ConfigValidateOutput{
			Valid:  false,
			Errors: errors,
		}
	}

	return &mcptypes.ConfigValidateOutput{
		Valid: true,
	}
}
