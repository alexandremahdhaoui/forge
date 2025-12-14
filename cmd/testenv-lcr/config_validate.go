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

// handleConfigValidate validates testenv-lcr engine configuration.
// It checks the spec fields:
//   - enabled (bool, optional) - whether to enable local container registry
//   - namespace (string, optional) - namespace for registry deployment
//   - imagePullSecretNamespaces ([]string, optional) - namespaces to create image pull secrets in
//   - imagePullSecretName (string, optional) - name of the image pull secret
//   - images ([]interface{}, optional) - images to pull and push to local registry
//
// Returns ConfigValidateOutput with valid=true if spec is valid,
// or with errors for invalid field types.
func handleConfigValidate(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input mcptypes.ConfigValidateInput,
) (*mcp.CallToolResult, any, error) {
	output := validateLcrSpec(input.Spec)

	// Return result with appropriate success/error status
	if output.Valid {
		result, artifact := mcputil.SuccessResultWithArtifact(
			"Configuration is valid",
			output,
		)
		return result, artifact, nil
	}

	// Validation failed
	result, artifact := mcputil.ErrorResultWithArtifact(
		fmt.Sprintf("Configuration validation failed with %d error(s)", len(output.Errors)),
		output,
	)
	return result, artifact, nil
}

// validateLcrSpec validates the testenv-lcr spec fields.
// This is extracted for easier unit testing.
func validateLcrSpec(spec map[string]interface{}) *mcptypes.ConfigValidateOutput {
	var errors []mcptypes.ValidationError

	// If spec is nil or empty, it's valid (no configuration to validate)
	if len(spec) == 0 {
		return &mcptypes.ConfigValidateOutput{
			Valid: true,
		}
	}

	// Validate enabled field (optional, must be bool if present)
	if _, verr := mcptypes.ValidateBool(spec, "enabled"); verr != nil {
		errors = append(errors, *verr)
	}

	// Validate namespace field (optional, must be string if present)
	if _, verr := mcptypes.ValidateString(spec, "namespace"); verr != nil {
		errors = append(errors, *verr)
	}

	// Validate imagePullSecretNamespaces field (optional, must be []string if present)
	if _, verr := mcptypes.ValidateStringSlice(spec, "imagePullSecretNamespaces"); verr != nil {
		errors = append(errors, *verr)
	}

	// Validate imagePullSecretName field (optional, must be string if present)
	if _, verr := mcptypes.ValidateString(spec, "imagePullSecretName"); verr != nil {
		errors = append(errors, *verr)
	}

	// Validate images field (optional, must be array if present)
	// Complex validation of image items is handled at runtime
	if val, ok := spec["images"]; ok && val != nil {
		if _, isArray := val.([]interface{}); !isArray {
			errors = append(errors, mcptypes.ValidationError{
				Field:   "spec.images",
				Message: "expected array",
			})
		}
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
