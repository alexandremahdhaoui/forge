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

// handleConfigValidate validates go-test engine configuration.
// It checks the spec fields:
//   - packages ([]string, optional) - packages to test
//   - tags ([]string, optional) - build tags to use
//   - timeout (string, optional) - test timeout
//   - race (bool, optional) - enable race detector
//   - cover (bool, optional) - enable coverage
//   - coverprofile (string, optional) - coverage profile output path
//   - args ([]string, optional) - additional arguments to pass to go test
//   - env (map[string]string, optional) - environment variables to set for tests
//
// Returns ConfigValidateOutput with valid=true if spec is valid,
// or with errors for invalid field types.
func handleConfigValidate(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input mcptypes.ConfigValidateInput,
) (*mcp.CallToolResult, any, error) {
	output := validateSpec(input.Spec)

	// Return result with appropriate success/error status
	if output.Valid {
		result, artifact := mcputil.SuccessResultWithArtifact(
			"Configuration is valid",
			output,
		)
		return result, artifact, nil
	}

	// Validation failed
	result, artifact := mcputil.SuccessResultWithArtifact(
		fmt.Sprintf("Configuration validation failed with %d error(s)", len(output.Errors)),
		output,
	)
	return result, artifact, nil
}

// validateSpec validates the go-test spec fields.
// This is extracted for easier unit testing.
func validateSpec(spec map[string]interface{}) *mcptypes.ConfigValidateOutput {
	var errors []mcptypes.ValidationError

	// If spec is nil or empty, it's valid (no configuration to validate)
	if len(spec) == 0 {
		return &mcptypes.ConfigValidateOutput{
			Valid: true,
		}
	}

	// Validate packages field (optional, must be []string if present)
	if _, verr := mcptypes.ValidateStringSlice(spec, "packages"); verr != nil {
		errors = append(errors, *verr)
	}

	// Validate tags field (optional, must be []string if present)
	if _, verr := mcptypes.ValidateStringSlice(spec, "tags"); verr != nil {
		errors = append(errors, *verr)
	}

	// Validate timeout field (optional, must be string if present)
	if _, verr := mcptypes.ValidateString(spec, "timeout"); verr != nil {
		errors = append(errors, *verr)
	}

	// Validate race field (optional, must be bool if present)
	if _, verr := mcptypes.ValidateBool(spec, "race"); verr != nil {
		errors = append(errors, *verr)
	}

	// Validate cover field (optional, must be bool if present)
	if _, verr := mcptypes.ValidateBool(spec, "cover"); verr != nil {
		errors = append(errors, *verr)
	}

	// Validate coverprofile field (optional, must be string if present)
	if _, verr := mcptypes.ValidateString(spec, "coverprofile"); verr != nil {
		errors = append(errors, *verr)
	}

	// Validate args field (optional, must be []string if present)
	if _, verr := mcptypes.ValidateStringSlice(spec, "args"); verr != nil {
		errors = append(errors, *verr)
	}

	// Validate env field (optional, must be map[string]string if present)
	if _, verr := mcptypes.ValidateStringMap(spec, "env"); verr != nil {
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
