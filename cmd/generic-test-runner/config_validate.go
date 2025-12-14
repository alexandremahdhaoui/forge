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
	"log"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/alexandremahdhaoui/forge/pkg/mcputil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// handleConfigValidate handles the config-validate MCP tool call.
// It validates the generic-test-runner spec fields:
// - command (string, REQUIRED) - command to execute
// - args ([]string, optional) - command arguments
// - env (map[string]string, optional) - environment variables
// - workDir (string, optional) - working directory
// - envFile (string, optional) - path to env file
func handleConfigValidate(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input mcptypes.ConfigValidateInput,
) (*mcp.CallToolResult, any, error) {
	log.Printf("Validating generic-test-runner config for spec: %s", input.SpecName)

	output := validateGenericTestRunnerSpec(input.Spec)

	if output.Valid {
		result, artifact := mcputil.SuccessResultWithArtifact(
			"Configuration is valid",
			output,
		)
		return result, artifact, nil
	}

	result, artifact := mcputil.ErrorResultWithArtifact(
		"Configuration validation failed",
		output,
	)
	return result, artifact, nil
}

// validateGenericTestRunnerSpec validates the generic-test-runner spec fields.
func validateGenericTestRunnerSpec(spec map[string]interface{}) *mcptypes.ConfigValidateOutput {
	output := &mcptypes.ConfigValidateOutput{
		Valid:  true,
		Errors: []mcptypes.ValidationError{},
	}

	// Handle nil spec
	if spec == nil {
		output.Valid = false
		output.Errors = append(output.Errors, mcptypes.ValidationError{
			Field:   "spec.command",
			Message: "required field is missing",
		})
		return output
	}

	// Validate command (REQUIRED)
	_, err := mcptypes.ValidateStringRequired(spec, "command")
	if err != nil {
		output.Valid = false
		output.Errors = append(output.Errors, *err)
	}

	// Validate args (optional, []string)
	_, err = mcptypes.ValidateStringSlice(spec, "args")
	if err != nil {
		output.Valid = false
		output.Errors = append(output.Errors, *err)
	}

	// Validate env (optional, map[string]string)
	_, err = mcptypes.ValidateStringMap(spec, "env")
	if err != nil {
		output.Valid = false
		output.Errors = append(output.Errors, *err)
	}

	// Validate workDir (optional, string)
	_, err = mcptypes.ValidateString(spec, "workDir")
	if err != nil {
		output.Valid = false
		output.Errors = append(output.Errors, *err)
	}

	// Validate envFile (optional, string)
	_, err = mcptypes.ValidateString(spec, "envFile")
	if err != nil {
		output.Valid = false
		output.Errors = append(output.Errors, *err)
	}

	return output
}
