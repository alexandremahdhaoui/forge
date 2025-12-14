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

package mcptypes

import (
	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

// ConfigValidateInput is the input for the config-validate MCP tool.
// This is sent to engines to validate their specific configuration.
type ConfigValidateInput struct {
	// Spec contains engine-specific configuration to validate.
	// Each engine interprets this according to its own schema.
	Spec map[string]interface{} `json:"spec,omitempty"`

	// ForgeSpec is the complete forge.yaml spec.
	// Required for orchestrators (testenv) to access referenced sections
	// like kindenv, localContainerRegistry, etc.
	ForgeSpec *forge.Spec `json:"forgeSpec,omitempty"`

	// ConfigPath is the path to forge.yaml (for error messages).
	ConfigPath string `json:"configPath,omitempty"`

	// DirectoryParams contains standardized directory paths.
	// Passed from forge for engines that need path context.
	DirectoryParams *DirectoryParams `json:"directoryParams,omitempty"`

	// SpecType indicates which forge.yaml section this spec came from.
	// Values: "build", "test", "testenv"
	// Used for error context.
	SpecType string `json:"specType,omitempty"`

	// SpecName is the name field from the forge.yaml spec.
	// e.g., "go-app" for a build spec, "unit" for a test spec.
	// Used for error context.
	SpecName string `json:"specName,omitempty"`
}

// ConfigValidateOutput is the output from the config-validate MCP tool.
// This is returned by engines after validating their configuration.
type ConfigValidateOutput struct {
	// Valid is true if the configuration passed all validation checks.
	// Will be false if there are any errors OR if InfraError is non-empty.
	Valid bool `json:"valid"`

	// Errors contains validation errors (if any).
	// Each error should reference the specific field that failed validation.
	Errors []ValidationError `json:"errors,omitempty"`

	// Warnings contains non-fatal validation issues.
	// The configuration is still valid, but may have suboptimal settings.
	Warnings []ValidationWarning `json:"warnings,omitempty"`

	// InfraError is non-empty if the engine failed to respond properly.
	// This indicates infrastructure-level failures like:
	// - Engine process failed to spawn
	// - MCP communication timeout
	// - Engine crashed during validation
	// When InfraError is set, Valid should be false.
	InfraError string `json:"infraError,omitempty"`
}

// ValidationError represents a single validation error.
// It provides detailed context about which field failed validation and why.
type ValidationError struct {
	// Field is the JSON path to the invalid field (e.g., "spec.args[0]").
	Field string `json:"field"`

	// Message is a human-readable error description.
	Message string `json:"message"`

	// Engine is the URI of the engine that reported this error.
	// Set by forge during aggregation, not by the engine itself.
	Engine string `json:"engine,omitempty"`

	// SpecType indicates which forge.yaml section contained the error.
	// Values: "build", "test", "testenv"
	// Propagated from ConfigValidateInput.SpecType.
	SpecType string `json:"specType,omitempty"`

	// SpecName is the name of the spec from forge.yaml.
	// Propagated from ConfigValidateInput.SpecName.
	SpecName string `json:"specName,omitempty"`
}

// ValidationWarning represents a non-fatal validation issue.
// The configuration is still valid, but there may be suboptimal settings
// or deprecated configurations that should be addressed.
type ValidationWarning struct {
	// Field is the optional JSON path to the relevant field.
	Field string `json:"field,omitempty"`

	// Message is a human-readable warning description.
	Message string `json:"message"`
}
