//go:build unit

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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		// Create temp directory with config file
		dir := t.TempDir()
		configContent := `name: go-build
type: builder
version: 0.15.0
description: Go binary builder with git versioning
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
`
		if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(configContent), 0o644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		config, err := ReadConfig(dir)
		if err != nil {
			t.Fatalf("ReadConfig failed: %v", err)
		}

		if config.Name != "go-build" {
			t.Errorf("Name = %q, want %q", config.Name, "go-build")
		}
		if config.Type != EngineTypeBuilder {
			t.Errorf("Type = %q, want %q", config.Type, EngineTypeBuilder)
		}
		if config.Version != "0.15.0" {
			t.Errorf("Version = %q, want %q", config.Version, "0.15.0")
		}
		if config.Description != "Go binary builder with git versioning" {
			t.Errorf("Description = %q, want %q", config.Description, "Go binary builder with git versioning")
		}
		if config.OpenAPI.SpecPath != "./spec.openapi.yaml" {
			t.Errorf("OpenAPI.SpecPath = %q, want %q", config.OpenAPI.SpecPath, "./spec.openapi.yaml")
		}
		if config.Generate.PackageName != "main" {
			t.Errorf("Generate.PackageName = %q, want %q", config.Generate.PackageName, "main")
		}
	})

	t.Run("test-runner type", func(t *testing.T) {
		dir := t.TempDir()
		configContent := `name: go-test
type: test-runner
version: 0.15.0
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
`
		if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(configContent), 0o644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		config, err := ReadConfig(dir)
		if err != nil {
			t.Fatalf("ReadConfig failed: %v", err)
		}

		if config.Type != EngineTypeTestRunner {
			t.Errorf("Type = %q, want %q", config.Type, EngineTypeTestRunner)
		}
	})

	t.Run("testenv-subengine type", func(t *testing.T) {
		dir := t.TempDir()
		configContent := `name: testenv-kind
type: testenv-subengine
version: 0.15.0
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
`
		if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(configContent), 0o644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		config, err := ReadConfig(dir)
		if err != nil {
			t.Fatalf("ReadConfig failed: %v", err)
		}

		if config.Type != EngineTypeTestEnvSubengine {
			t.Errorf("Type = %q, want %q", config.Type, EngineTypeTestEnvSubengine)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		dir := t.TempDir()

		_, err := ReadConfig(dir)
		if err == nil {
			t.Error("ReadConfig should fail for missing file")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte("invalid: yaml: content:"), 0o644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		_, err := ReadConfig(dir)
		if err == nil {
			t.Error("ReadConfig should fail for invalid YAML")
		}
	})
}

func TestValidateConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := &Config{
			Name:    "go-build",
			Type:    EngineTypeBuilder,
			Version: "0.15.0",
			OpenAPI: OpenAPIConfig{
				SpecPath: "./spec.openapi.yaml",
			},
			Generate: GenerateConfig{
				PackageName: "main",
			},
		}

		errors := ValidateConfig(config)
		if len(errors) > 0 {
			t.Errorf("ValidateConfig returned errors for valid config: %v", errors)
		}
	})

	t.Run("missing name", func(t *testing.T) {
		config := &Config{
			Type:    EngineTypeBuilder,
			Version: "0.15.0",
			OpenAPI: OpenAPIConfig{
				SpecPath: "./spec.openapi.yaml",
			},
			Generate: GenerateConfig{
				PackageName: "main",
			},
		}

		errors := ValidateConfig(config)
		if !hasErrorForField(errors, "name") {
			t.Error("Expected error for missing name")
		}
	})

	t.Run("invalid name format - uppercase", func(t *testing.T) {
		config := &Config{
			Name:    "Go-Build",
			Type:    EngineTypeBuilder,
			Version: "0.15.0",
			OpenAPI: OpenAPIConfig{
				SpecPath: "./spec.openapi.yaml",
			},
			Generate: GenerateConfig{
				PackageName: "main",
			},
		}

		errors := ValidateConfig(config)
		if !hasErrorForField(errors, "name") {
			t.Error("Expected error for invalid name format")
		}
	})

	t.Run("invalid name format - starts with number", func(t *testing.T) {
		config := &Config{
			Name:    "1go-build",
			Type:    EngineTypeBuilder,
			Version: "0.15.0",
			OpenAPI: OpenAPIConfig{
				SpecPath: "./spec.openapi.yaml",
			},
			Generate: GenerateConfig{
				PackageName: "main",
			},
		}

		errors := ValidateConfig(config)
		if !hasErrorForField(errors, "name") {
			t.Error("Expected error for name starting with number")
		}
	})

	t.Run("invalid name format - special characters", func(t *testing.T) {
		config := &Config{
			Name:    "go_build",
			Type:    EngineTypeBuilder,
			Version: "0.15.0",
			OpenAPI: OpenAPIConfig{
				SpecPath: "./spec.openapi.yaml",
			},
			Generate: GenerateConfig{
				PackageName: "main",
			},
		}

		errors := ValidateConfig(config)
		if !hasErrorForField(errors, "name") {
			t.Error("Expected error for name with underscores")
		}
	})

	t.Run("name too long", func(t *testing.T) {
		config := &Config{
			Name:    strings.Repeat("a", 65),
			Type:    EngineTypeBuilder,
			Version: "0.15.0",
			OpenAPI: OpenAPIConfig{
				SpecPath: "./spec.openapi.yaml",
			},
			Generate: GenerateConfig{
				PackageName: "main",
			},
		}

		errors := ValidateConfig(config)
		if !hasErrorForField(errors, "name") {
			t.Error("Expected error for name too long")
		}
	})

	t.Run("missing type", func(t *testing.T) {
		config := &Config{
			Name:    "go-build",
			Version: "0.15.0",
			OpenAPI: OpenAPIConfig{
				SpecPath: "./spec.openapi.yaml",
			},
			Generate: GenerateConfig{
				PackageName: "main",
			},
		}

		errors := ValidateConfig(config)
		if !hasErrorForField(errors, "type") {
			t.Error("Expected error for missing type")
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		config := &Config{
			Name:    "go-build",
			Type:    "invalid-type",
			Version: "0.15.0",
			OpenAPI: OpenAPIConfig{
				SpecPath: "./spec.openapi.yaml",
			},
			Generate: GenerateConfig{
				PackageName: "main",
			},
		}

		errors := ValidateConfig(config)
		if !hasErrorForField(errors, "type") {
			t.Error("Expected error for invalid type")
		}
	})

	t.Run("missing version", func(t *testing.T) {
		config := &Config{
			Name: "go-build",
			Type: EngineTypeBuilder,
			OpenAPI: OpenAPIConfig{
				SpecPath: "./spec.openapi.yaml",
			},
			Generate: GenerateConfig{
				PackageName: "main",
			},
		}

		errors := ValidateConfig(config)
		if !hasErrorForField(errors, "version") {
			t.Error("Expected error for missing version")
		}
	})

	t.Run("invalid version format", func(t *testing.T) {
		config := &Config{
			Name:    "go-build",
			Type:    EngineTypeBuilder,
			Version: "v0.15.0", // Invalid: has 'v' prefix
			OpenAPI: OpenAPIConfig{
				SpecPath: "./spec.openapi.yaml",
			},
			Generate: GenerateConfig{
				PackageName: "main",
			},
		}

		errors := ValidateConfig(config)
		if !hasErrorForField(errors, "version") {
			t.Error("Expected error for invalid version format")
		}
	})

	t.Run("invalid version format - not semver", func(t *testing.T) {
		config := &Config{
			Name:    "go-build",
			Type:    EngineTypeBuilder,
			Version: "1.0", // Invalid: missing patch version
			OpenAPI: OpenAPIConfig{
				SpecPath: "./spec.openapi.yaml",
			},
			Generate: GenerateConfig{
				PackageName: "main",
			},
		}

		errors := ValidateConfig(config)
		if !hasErrorForField(errors, "version") {
			t.Error("Expected error for non-semver version")
		}
	})

	t.Run("missing openapi.specPath", func(t *testing.T) {
		config := &Config{
			Name:    "go-build",
			Type:    EngineTypeBuilder,
			Version: "0.15.0",
			Generate: GenerateConfig{
				PackageName: "main",
			},
		}

		errors := ValidateConfig(config)
		if !hasErrorForField(errors, "openapi.specPath") {
			t.Error("Expected error for missing openapi.specPath")
		}
	})

	t.Run("missing generate.packageName", func(t *testing.T) {
		config := &Config{
			Name:    "go-build",
			Type:    EngineTypeBuilder,
			Version: "0.15.0",
			OpenAPI: OpenAPIConfig{
				SpecPath: "./spec.openapi.yaml",
			},
		}

		errors := ValidateConfig(config)
		if !hasErrorForField(errors, "generate.packageName") {
			t.Error("Expected error for missing generate.packageName")
		}
	})

	t.Run("invalid generate.packageName - uppercase", func(t *testing.T) {
		config := &Config{
			Name:    "go-build",
			Type:    EngineTypeBuilder,
			Version: "0.15.0",
			OpenAPI: OpenAPIConfig{
				SpecPath: "./spec.openapi.yaml",
			},
			Generate: GenerateConfig{
				PackageName: "Main",
			},
		}

		errors := ValidateConfig(config)
		if !hasErrorForField(errors, "generate.packageName") {
			t.Error("Expected error for invalid package name")
		}
	})

	t.Run("invalid generate.packageName - starts with number", func(t *testing.T) {
		config := &Config{
			Name:    "go-build",
			Type:    EngineTypeBuilder,
			Version: "0.15.0",
			OpenAPI: OpenAPIConfig{
				SpecPath: "./spec.openapi.yaml",
			},
			Generate: GenerateConfig{
				PackageName: "1main",
			},
		}

		errors := ValidateConfig(config)
		if !hasErrorForField(errors, "generate.packageName") {
			t.Error("Expected error for package name starting with number")
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		config := &Config{}

		errors := ValidateConfig(config)
		if len(errors) < 4 {
			t.Errorf("Expected at least 4 errors for empty config, got %d", len(errors))
		}
	})
}

func TestValidationError(t *testing.T) {
	err := ValidationError{
		Field:   "name",
		Message: "required field is missing",
	}

	expected := "name: required field is missing"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

// hasErrorForField checks if any error is for the specified field.
func hasErrorForField(errors []ValidationError, field string) bool {
	for _, e := range errors {
		if e.Field == field {
			return true
		}
	}
	return false
}
