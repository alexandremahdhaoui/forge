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
func handleConfigValidate(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input mcptypes.ConfigValidateInput,
) (*mcp.CallToolResult, any, error) {
	output := validateKindSpec(input.Spec)

	// Return result with appropriate success/error status
	if output.Valid {
		result, artifact := mcputil.SuccessResultWithArtifact(
			"testenv-kind configuration is valid",
			output,
		)
		return result, artifact, nil
	}

	// Validation failed
	result, artifact := mcputil.ErrorResultWithArtifact(
		fmt.Sprintf("testenv-kind configuration validation failed with %d error(s)", len(output.Errors)),
		output,
	)
	return result, artifact, nil
}

// validateKindSpec validates the testenv-kind spec fields.
func validateKindSpec(spec map[string]interface{}) *mcptypes.ConfigValidateOutput {
	output := &mcptypes.ConfigValidateOutput{
		Valid:    true,
		Errors:   []mcptypes.ValidationError{},
		Warnings: []mcptypes.ValidationWarning{},
	}

	if spec == nil {
		return output
	}

	// Validate name (string, optional)
	if _, err := mcptypes.ValidateString(spec, "name"); err != nil {
		output.Errors = append(output.Errors, *err)
		output.Valid = false
	}

	// Validate image (string, optional)
	if _, err := mcptypes.ValidateString(spec, "image"); err != nil {
		output.Errors = append(output.Errors, *err)
		output.Valid = false
	}

	// Validate config (string, optional) - path to kind config file
	if _, err := mcptypes.ValidateString(spec, "config"); err != nil {
		output.Errors = append(output.Errors, *err)
		output.Valid = false
	}

	// Validate waitTimeout (string, optional)
	if _, err := mcptypes.ValidateString(spec, "waitTimeout"); err != nil {
		output.Errors = append(output.Errors, *err)
		output.Valid = false
	}

	// Validate retain (bool, optional)
	if _, err := mcptypes.ValidateBool(spec, "retain"); err != nil {
		output.Errors = append(output.Errors, *err)
		output.Valid = false
	}

	return output
}
