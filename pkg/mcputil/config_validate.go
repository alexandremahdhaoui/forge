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

package mcputil

import (
	"encoding/json"
	"fmt"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewValidateOutput creates a new ConfigValidateOutput with Valid set to true
// and empty error/warning slices. This is the starting point for building
// validation results.
//
// Example usage:
//
//	out := mcputil.NewValidateOutput()
//	if someField == "" {
//	    mcputil.AddValidationError(out, "spec.someField", "required field is missing")
//	}
//	return mcputil.ValidateOutputResult(out), nil, nil
func NewValidateOutput() *mcptypes.ConfigValidateOutput {
	return &mcptypes.ConfigValidateOutput{
		Valid:    true,
		Errors:   []mcptypes.ValidationError{},
		Warnings: []mcptypes.ValidationWarning{},
	}
}

// AddValidationError adds a validation error to the output and sets Valid to false.
// Use this when a configuration field fails validation and the configuration
// should not be used.
//
// Parameters:
//   - out: the ConfigValidateOutput to add the error to
//   - field: the JSON path to the invalid field (e.g., "spec.args[0]")
//   - message: a human-readable error description
//
// Example usage:
//
//	mcputil.AddValidationError(out, "spec.timeout", "must be a positive integer")
func AddValidationError(out *mcptypes.ConfigValidateOutput, field, message string) {
	out.Valid = false
	out.Errors = append(out.Errors, mcptypes.ValidationError{
		Field:   field,
		Message: message,
	})
}

// AddValidationWarning adds a validation warning to the output without changing Valid.
// Use this for non-fatal issues like deprecated configurations or suboptimal settings.
//
// Parameters:
//   - out: the ConfigValidateOutput to add the warning to
//   - message: a human-readable warning description
//
// Example usage:
//
//	mcputil.AddValidationWarning(out, "deprecated field 'oldName' will be removed in v2.0")
func AddValidationWarning(out *mcptypes.ConfigValidateOutput, message string) {
	out.Warnings = append(out.Warnings, mcptypes.ValidationWarning{
		Message: message,
	})
}

// AddValidationWarningWithField adds a validation warning with a field reference.
// Use this for warnings that relate to a specific configuration field.
//
// Parameters:
//   - out: the ConfigValidateOutput to add the warning to
//   - field: the JSON path to the relevant field (e.g., "spec.timeout")
//   - message: a human-readable warning description
//
// Example usage:
//
//	mcputil.AddValidationWarningWithField(out, "spec.oldField", "deprecated, use 'newField' instead")
func AddValidationWarningWithField(out *mcptypes.ConfigValidateOutput, field, message string) {
	out.Warnings = append(out.Warnings, mcptypes.ValidationWarning{
		Field:   field,
		Message: message,
	})
}

// ValidateOutputResult converts a ConfigValidateOutput to an MCP CallToolResult.
// This should be called at the end of validation to produce the final MCP response.
//
// The result is:
//   - A success result (IsError=false) if Valid is true
//   - An error result (IsError=true) if Valid is false
//
// The output is serialized as JSON in the result content.
//
// Example usage:
//
//	out := mcputil.NewValidateOutput()
//	// ... perform validation, add errors/warnings ...
//	return mcputil.ValidateOutputResult(out), out, nil
func ValidateOutputResult(out *mcptypes.ConfigValidateOutput) *mcp.CallToolResult {
	// Serialize the output to JSON
	jsonBytes, err := json.Marshal(out)
	if err != nil {
		// This should never happen with ConfigValidateOutput, but handle it gracefully
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("failed to serialize validation output: %v", err)},
			},
			IsError: true,
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonBytes)},
		},
		IsError: !out.Valid,
	}
}
