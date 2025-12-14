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
// It validates the go-gen-protobuf spec fields:
// - protoDir (string, optional) - directory containing .proto files
// - outputDir (string, optional) - directory for generated Go code
// - includes ([]string, optional) - additional include paths for protoc
func handleConfigValidate(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input mcptypes.ConfigValidateInput,
) (*mcp.CallToolResult, any, error) {
	log.Printf("Validating go-gen-protobuf config for spec: %s", input.SpecName)

	output := validateGoGenProtobufSpec(input.Spec)

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

// validateGoGenProtobufSpec validates the go-gen-protobuf spec fields.
func validateGoGenProtobufSpec(spec map[string]interface{}) *mcptypes.ConfigValidateOutput {
	output := &mcptypes.ConfigValidateOutput{
		Valid:  true,
		Errors: []mcptypes.ValidationError{},
	}

	// Handle nil spec - all fields are optional, so nil spec is valid
	if spec == nil {
		return output
	}

	// Validate protoDir (optional, string)
	_, err := mcptypes.ValidateString(spec, "protoDir")
	if err != nil {
		output.Valid = false
		output.Errors = append(output.Errors, *err)
	}

	// Validate outputDir (optional, string)
	_, err = mcptypes.ValidateString(spec, "outputDir")
	if err != nil {
		output.Valid = false
		output.Errors = append(output.Errors, *err)
	}

	// Validate includes (optional, []string)
	_, err = mcptypes.ValidateStringSlice(spec, "includes")
	if err != nil {
		output.Valid = false
		output.Errors = append(output.Errors, *err)
	}

	return output
}
