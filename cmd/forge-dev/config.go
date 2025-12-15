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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigFileName is the expected name of the forge-dev configuration file.
const ConfigFileName = "forge-dev.yaml"

// EngineType represents the type of engine being configured.
type EngineType string

const (
	// EngineTypeBuilder is for build engines that produce artifacts.
	EngineTypeBuilder EngineType = "builder"
	// EngineTypeTestRunner is for test runner engines.
	EngineTypeTestRunner EngineType = "test-runner"
	// EngineTypeTestEnvSubengine is for test environment subengines.
	EngineTypeTestEnvSubengine EngineType = "testenv-subengine"
	// EngineTypeDependencyDetector is for dependency detector engines.
	EngineTypeDependencyDetector EngineType = "dependency-detector"
)

// ValidEngineTypes contains all valid engine types.
var ValidEngineTypes = []EngineType{
	EngineTypeBuilder,
	EngineTypeTestRunner,
	EngineTypeTestEnvSubengine,
	EngineTypeDependencyDetector,
}

// Config represents the forge-dev.yaml configuration file.
type Config struct {
	// Name is the engine name (required).
	// Must be lowercase alphanumeric with hyphens, starting with a letter.
	Name string `yaml:"name"`

	// Type is the engine type (required).
	// Must be one of: builder, test-runner, testenv-subengine.
	Type EngineType `yaml:"type"`

	// Version is the engine version in semver format (required).
	Version string `yaml:"version"`

	// Description is a human-readable description of the engine (optional).
	Description string `yaml:"description,omitempty"`

	// OpenAPI contains OpenAPI spec configuration.
	OpenAPI OpenAPIConfig `yaml:"openapi"`

	// Generate contains code generation settings.
	Generate GenerateConfig `yaml:"generate"`
}

// OpenAPIConfig contains OpenAPI specification configuration.
type OpenAPIConfig struct {
	// SpecPath is the path to the OpenAPI spec file, relative to forge-dev.yaml.
	SpecPath string `yaml:"specPath"`
}

// GenerateConfig contains code generation settings.
type GenerateConfig struct {
	// PackageName is the Go package name for generated code.
	PackageName string `yaml:"packageName"`
}

// ValidationError represents a single validation error.
type ValidationError struct {
	// Field is the path to the field that failed validation.
	Field string
	// Message describes the validation failure.
	Message string
}

// Error implements the error interface.
func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ReadConfig reads and parses the forge-dev.yaml configuration file from the specified directory.
func ReadConfig(dir string) (*Config, error) {
	configPath := filepath.Join(dir, ConfigFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", configPath, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", configPath, err)
	}

	return &config, nil
}

// nameRegexp validates that name is lowercase alphanumeric with hyphens, starting with a letter.
var nameRegexp = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// semverRegexp validates semantic versioning format (x.y.z).
var semverRegexp = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

// packageNameRegexp validates Go package names.
var packageNameRegexp = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// ValidateConfig validates the configuration and returns any validation errors.
func ValidateConfig(c *Config) []ValidationError {
	var errors []ValidationError

	// Validate name (required)
	if c.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "required field is missing",
		})
	} else if !nameRegexp.MatchString(c.Name) {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "must be lowercase alphanumeric with hyphens, starting with a letter",
		})
	} else if len(c.Name) > 64 {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "must be 64 characters or less",
		})
	}

	// Validate type (required)
	if c.Type == "" {
		errors = append(errors, ValidationError{
			Field:   "type",
			Message: "required field is missing",
		})
	} else if !isValidEngineType(c.Type) {
		errors = append(errors, ValidationError{
			Field:   "type",
			Message: fmt.Sprintf("must be one of: %s", strings.Join(engineTypeStrings(), ", ")),
		})
	}

	// Validate version (required)
	if c.Version == "" {
		errors = append(errors, ValidationError{
			Field:   "version",
			Message: "required field is missing",
		})
	} else if !semverRegexp.MatchString(c.Version) {
		errors = append(errors, ValidationError{
			Field:   "version",
			Message: "must be in semver format (x.y.z)",
		})
	}

	// Validate openapi.specPath (required)
	if c.OpenAPI.SpecPath == "" {
		errors = append(errors, ValidationError{
			Field:   "openapi.specPath",
			Message: "required field is missing",
		})
	}

	// Validate generate.packageName (required)
	if c.Generate.PackageName == "" {
		errors = append(errors, ValidationError{
			Field:   "generate.packageName",
			Message: "required field is missing",
		})
	} else if !packageNameRegexp.MatchString(c.Generate.PackageName) {
		errors = append(errors, ValidationError{
			Field:   "generate.packageName",
			Message: "must be a valid Go package name (lowercase alphanumeric with underscores, starting with a letter)",
		})
	}

	return errors
}

// isValidEngineType checks if the given type is a valid engine type.
func isValidEngineType(t EngineType) bool {
	for _, valid := range ValidEngineTypes {
		if t == valid {
			return true
		}
	}
	return false
}

// engineTypeStrings returns the string representations of all valid engine types.
func engineTypeStrings() []string {
	strs := make([]string, len(ValidEngineTypes))
	for i, t := range ValidEngineTypes {
		strs[i] = string(t)
	}
	return strs
}
