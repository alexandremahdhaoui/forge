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
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

func TestValidateForgeDevConfig(t *testing.T) {
	t.Run("valid config returns valid=true", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create valid forge-dev.yaml
		configContent := `name: valid-engine
type: builder
version: 0.1.0
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
`
		if err := os.WriteFile(filepath.Join(tmpDir, "forge-dev.yaml"), []byte(configContent), 0o644); err != nil {
			t.Fatalf("writing forge-dev.yaml: %v", err)
		}

		// Create valid spec.openapi.yaml
		specContent := `openapi: 3.0.3
info:
  title: valid-engine Spec Schema
  version: 0.1.0
components:
  schemas:
    Spec:
      type: object
      properties:
        field:
          type: string
`
		if err := os.WriteFile(filepath.Join(tmpDir, "spec.openapi.yaml"), []byte(specContent), 0o644); err != nil {
			t.Fatalf("writing spec.openapi.yaml: %v", err)
		}

		input := mcptypes.ConfigValidateInput{
			Spec: map[string]interface{}{
				"configPath": tmpDir,
			},
		}

		output := validateForgeDevConfig(input)

		if !output.Valid {
			t.Errorf("expected valid=true, got errors: %v", output.Errors)
		}
	})

	t.Run("missing forge-dev.yaml returns error", func(t *testing.T) {
		tmpDir := t.TempDir()

		input := mcptypes.ConfigValidateInput{
			Spec: map[string]interface{}{
				"configPath": tmpDir,
			},
		}

		output := validateForgeDevConfig(input)

		if output.Valid {
			t.Error("expected valid=false for missing forge-dev.yaml")
		}
		if len(output.Errors) == 0 {
			t.Error("expected at least one error")
		}
		found := false
		for _, err := range output.Errors {
			if err.Field == "forge-dev.yaml" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected error for forge-dev.yaml field")
		}
	})

	t.Run("invalid type enum returns error", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create forge-dev.yaml with invalid type
		configContent := `name: invalid-engine
type: invalid-type
version: 0.1.0
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
`
		if err := os.WriteFile(filepath.Join(tmpDir, "forge-dev.yaml"), []byte(configContent), 0o644); err != nil {
			t.Fatalf("writing forge-dev.yaml: %v", err)
		}

		input := mcptypes.ConfigValidateInput{
			Spec: map[string]interface{}{
				"configPath": tmpDir,
			},
		}

		output := validateForgeDevConfig(input)

		if output.Valid {
			t.Error("expected valid=false for invalid type enum")
		}
		found := false
		for _, err := range output.Errors {
			if err.Field == "type" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected error for type field")
		}
	})

	t.Run("missing Spec schema in OpenAPI returns error", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create valid forge-dev.yaml
		configContent := `name: no-spec-engine
type: builder
version: 0.1.0
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
`
		if err := os.WriteFile(filepath.Join(tmpDir, "forge-dev.yaml"), []byte(configContent), 0o644); err != nil {
			t.Fatalf("writing forge-dev.yaml: %v", err)
		}

		// Create OpenAPI spec without Spec schema
		specContent := `openapi: 3.0.3
info:
  title: no-spec-engine Spec Schema
  version: 0.1.0
components:
  schemas:
    OtherSchema:
      type: object
      properties:
        field:
          type: string
`
		if err := os.WriteFile(filepath.Join(tmpDir, "spec.openapi.yaml"), []byte(specContent), 0o644); err != nil {
			t.Fatalf("writing spec.openapi.yaml: %v", err)
		}

		input := mcptypes.ConfigValidateInput{
			Spec: map[string]interface{}{
				"configPath": tmpDir,
			},
		}

		output := validateForgeDevConfig(input)

		if output.Valid {
			t.Error("expected valid=false for missing Spec schema")
		}
		found := false
		for _, err := range output.Errors {
			if err.Field == "spec.openapi.yaml" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected error for spec.openapi.yaml field")
		}
	})

	t.Run("missing configPath returns error", func(t *testing.T) {
		input := mcptypes.ConfigValidateInput{
			Spec: map[string]interface{}{},
		}

		output := validateForgeDevConfig(input)

		if output.Valid {
			t.Error("expected valid=false for missing configPath")
		}
		found := false
		for _, err := range output.Errors {
			if err.Field == "configPath" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error for configPath field, got: %v", output.Errors)
		}
	})

	t.Run("empty Spec schema returns warning", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create valid forge-dev.yaml
		configContent := `name: empty-spec-engine
type: builder
version: 0.1.0
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
`
		if err := os.WriteFile(filepath.Join(tmpDir, "forge-dev.yaml"), []byte(configContent), 0o644); err != nil {
			t.Fatalf("writing forge-dev.yaml: %v", err)
		}

		// Create OpenAPI spec with empty Spec schema
		specContent := `openapi: 3.0.3
info:
  title: empty-spec-engine Spec Schema
  version: 0.1.0
components:
  schemas:
    Spec:
      type: object
`
		if err := os.WriteFile(filepath.Join(tmpDir, "spec.openapi.yaml"), []byte(specContent), 0o644); err != nil {
			t.Fatalf("writing spec.openapi.yaml: %v", err)
		}

		input := mcptypes.ConfigValidateInput{
			Spec: map[string]interface{}{
				"configPath": tmpDir,
			},
		}

		output := validateForgeDevConfig(input)

		if !output.Valid {
			t.Errorf("expected valid=true, got errors: %v", output.Errors)
		}
		if len(output.Warnings) == 0 {
			t.Error("expected warning for empty Spec schema")
		}
	})

	t.Run("uses directoryParams.RootDir as fallback", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create valid forge-dev.yaml
		configContent := `name: dir-params-engine
type: builder
version: 0.1.0
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
`
		if err := os.WriteFile(filepath.Join(tmpDir, "forge-dev.yaml"), []byte(configContent), 0o644); err != nil {
			t.Fatalf("writing forge-dev.yaml: %v", err)
		}

		// Create valid spec.openapi.yaml
		specContent := `openapi: 3.0.3
info:
  title: dir-params-engine Spec Schema
  version: 0.1.0
components:
  schemas:
    Spec:
      type: object
      properties:
        field:
          type: string
`
		if err := os.WriteFile(filepath.Join(tmpDir, "spec.openapi.yaml"), []byte(specContent), 0o644); err != nil {
			t.Fatalf("writing spec.openapi.yaml: %v", err)
		}

		input := mcptypes.ConfigValidateInput{
			DirectoryParams: &mcptypes.DirectoryParams{
				RootDir: tmpDir,
			},
			Spec: map[string]interface{}{},
		}

		output := validateForgeDevConfig(input)

		if !output.Valid {
			t.Errorf("expected valid=true using directoryParams, got errors: %v", output.Errors)
		}
	})
}
