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
)

func TestParseOpenAPISpec(t *testing.T) {
	t.Run("valid spec with various types", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `openapi: 3.0.3
info:
  title: Test Engine
  version: 0.1.0
components:
  schemas:
    Spec:
      type: object
      properties:
        stringField:
          type: string
          description: A string field
        boolField:
          type: boolean
          description: A boolean field
        intField:
          type: integer
          description: An integer field
        numberField:
          type: number
          description: A number field
        arrayField:
          type: array
          items:
            type: string
          description: An array of strings
        intArrayField:
          type: array
          items:
            type: integer
          description: An array of integers
        mapField:
          type: object
          additionalProperties:
            type: string
          description: A map of strings
      required:
        - stringField
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		schema, err := ParseOpenAPISpec(specPath)
		if err != nil {
			t.Fatalf("ParseOpenAPISpec failed: %v", err)
		}

		// Check properties count
		if len(schema.Properties) != 7 {
			t.Errorf("Expected 7 properties, got %d", len(schema.Properties))
		}

		// Check required fields
		if len(schema.Required) != 1 || schema.Required[0] != "stringField" {
			t.Errorf("Expected required=[stringField], got %v", schema.Required)
		}

		// Find and verify string field
		stringField := findProperty(schema.Properties, "stringField")
		if stringField == nil {
			t.Fatal("stringField not found")
		}
		if stringField.Type != "string" {
			t.Errorf("stringField.Type = %q, want %q", stringField.Type, "string")
		}
		if stringField.Description != "A string field" {
			t.Errorf("stringField.Description = %q, want %q", stringField.Description, "A string field")
		}
		if !stringField.Required {
			t.Error("stringField should be required")
		}

		// Verify bool field
		boolField := findProperty(schema.Properties, "boolField")
		if boolField == nil {
			t.Fatal("boolField not found")
		}
		if boolField.Type != "boolean" {
			t.Errorf("boolField.Type = %q, want %q", boolField.Type, "boolean")
		}
		if boolField.Required {
			t.Error("boolField should not be required")
		}

		// Verify int field
		intField := findProperty(schema.Properties, "intField")
		if intField == nil {
			t.Fatal("intField not found")
		}
		if intField.Type != "integer" {
			t.Errorf("intField.Type = %q, want %q", intField.Type, "integer")
		}

		// Verify number field
		numberField := findProperty(schema.Properties, "numberField")
		if numberField == nil {
			t.Fatal("numberField not found")
		}
		if numberField.Type != "number" {
			t.Errorf("numberField.Type = %q, want %q", numberField.Type, "number")
		}

		// Verify array field
		arrayField := findProperty(schema.Properties, "arrayField")
		if arrayField == nil {
			t.Fatal("arrayField not found")
		}
		if arrayField.Type != "array" {
			t.Errorf("arrayField.Type = %q, want %q", arrayField.Type, "array")
		}
		if arrayField.Items == nil || arrayField.Items.Type != "string" {
			t.Error("arrayField.Items should be string type")
		}

		// Verify int array field
		intArrayField := findProperty(schema.Properties, "intArrayField")
		if intArrayField == nil {
			t.Fatal("intArrayField not found")
		}
		if intArrayField.Items == nil || intArrayField.Items.Type != "integer" {
			t.Error("intArrayField.Items should be integer type")
		}

		// Verify map field
		mapField := findProperty(schema.Properties, "mapField")
		if mapField == nil {
			t.Fatal("mapField not found")
		}
		if mapField.Type != "object" {
			t.Errorf("mapField.Type = %q, want %q", mapField.Type, "object")
		}
		if mapField.AdditionalProperties == nil || mapField.AdditionalProperties.Type != "string" {
			t.Error("mapField.AdditionalProperties should be string type")
		}
	})

	t.Run("nested object", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `openapi: 3.0.3
info:
  title: Test Engine
  version: 0.1.0
components:
  schemas:
    Spec:
      type: object
      properties:
        nested:
          type: object
          properties:
            subField:
              type: string
              description: A nested string field
            subInt:
              type: integer
          required:
            - subField
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		schema, err := ParseOpenAPISpec(specPath)
		if err != nil {
			t.Fatalf("ParseOpenAPISpec failed: %v", err)
		}

		// Find nested property
		nested := findProperty(schema.Properties, "nested")
		if nested == nil {
			t.Fatal("nested not found")
		}
		if nested.Type != "object" {
			t.Errorf("nested.Type = %q, want %q", nested.Type, "object")
		}
		if len(nested.Properties) != 2 {
			t.Errorf("Expected 2 nested properties, got %d", len(nested.Properties))
		}

		// Find subField
		subField := findProperty(nested.Properties, "subField")
		if subField == nil {
			t.Fatal("subField not found")
		}
		if subField.Type != "string" {
			t.Errorf("subField.Type = %q, want %q", subField.Type, "string")
		}
		if !subField.Required {
			t.Error("subField should be required")
		}

		// Find subInt
		subInt := findProperty(nested.Properties, "subInt")
		if subInt == nil {
			t.Fatal("subInt not found")
		}
		if subInt.Required {
			t.Error("subInt should not be required")
		}
	})

	t.Run("enum field", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `openapi: 3.0.3
info:
  title: Test Engine
  version: 0.1.0
components:
  schemas:
    Spec:
      type: object
      properties:
        status:
          type: string
          enum:
            - pending
            - running
            - completed
            - failed
          description: The status of the operation
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		schema, err := ParseOpenAPISpec(specPath)
		if err != nil {
			t.Fatalf("ParseOpenAPISpec failed: %v", err)
		}

		status := findProperty(schema.Properties, "status")
		if status == nil {
			t.Fatal("status not found")
		}
		if len(status.Enum) != 4 {
			t.Errorf("Expected 4 enum values, got %d", len(status.Enum))
		}
		expectedEnums := []string{"pending", "running", "completed", "failed"}
		for i, expected := range expectedEnums {
			if status.Enum[i] != expected {
				t.Errorf("Enum[%d] = %q, want %q", i, status.Enum[i], expected)
			}
		}
	})

	t.Run("default values", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `openapi: 3.0.3
info:
  title: Test Engine
  version: 0.1.0
components:
  schemas:
    Spec:
      type: object
      properties:
        timeout:
          type: integer
          default: 30
          description: Timeout in seconds
        verbose:
          type: boolean
          default: false
        name:
          type: string
          default: "default-name"
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		schema, err := ParseOpenAPISpec(specPath)
		if err != nil {
			t.Fatalf("ParseOpenAPISpec failed: %v", err)
		}

		timeout := findProperty(schema.Properties, "timeout")
		if timeout == nil {
			t.Fatal("timeout not found")
		}
		if timeout.Default != 30 {
			t.Errorf("timeout.Default = %v, want 30", timeout.Default)
		}

		verbose := findProperty(schema.Properties, "verbose")
		if verbose == nil {
			t.Fatal("verbose not found")
		}
		if verbose.Default != false {
			t.Errorf("verbose.Default = %v, want false", verbose.Default)
		}

		name := findProperty(schema.Properties, "name")
		if name == nil {
			t.Fatal("name not found")
		}
		if name.Default != "default-name" {
			t.Errorf("name.Default = %v, want %q", name.Default, "default-name")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := ParseOpenAPISpec("/nonexistent/path/spec.yaml")
		if err == nil {
			t.Error("Expected error for missing file")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		dir := t.TempDir()
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte("invalid: yaml: content:"), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		_, err := ParseOpenAPISpec(specPath)
		if err == nil {
			t.Error("Expected error for invalid YAML")
		}
	})

	t.Run("missing openapi field", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `info:
  title: Test
components:
  schemas:
    Spec:
      type: object
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		_, err := ParseOpenAPISpec(specPath)
		if err == nil {
			t.Error("Expected error for missing openapi field")
		}
	})

	t.Run("missing Spec schema", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `openapi: 3.0.3
info:
  title: Test
components:
  schemas:
    OtherSchema:
      type: object
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		_, err := ParseOpenAPISpec(specPath)
		if err == nil {
			t.Error("Expected error for missing Spec schema")
		}
	})

	t.Run("Spec not object type", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `openapi: 3.0.3
info:
  title: Test
components:
  schemas:
    Spec:
      type: string
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		_, err := ParseOpenAPISpec(specPath)
		if err == nil {
			t.Error("Expected error for Spec not being object type")
		}
	})

	t.Run("error on $ref", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `openapi: 3.0.3
info:
  title: Test
components:
  schemas:
    OtherType:
      type: object
      properties:
        foo:
          type: string
    Spec:
      type: object
      properties:
        ref:
          $ref: '#/components/schemas/OtherType'
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		_, err := ParseOpenAPISpec(specPath)
		if err == nil {
			t.Error("Expected error for $ref")
		}
	})

	t.Run("error on array of objects", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `openapi: 3.0.3
info:
  title: Test
components:
  schemas:
    Spec:
      type: object
      properties:
        items:
          type: array
          items:
            type: object
            properties:
              name:
                type: string
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		_, err := ParseOpenAPISpec(specPath)
		if err == nil {
			t.Error("Expected error for array of objects")
		}
	})

	t.Run("error on oneOf", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `openapi: 3.0.3
info:
  title: Test
components:
  schemas:
    Spec:
      type: object
      properties:
        field:
          oneOf:
            - type: string
            - type: integer
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		_, err := ParseOpenAPISpec(specPath)
		if err == nil {
			t.Error("Expected error for oneOf")
		}
	})

	t.Run("error on anyOf", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `openapi: 3.0.3
info:
  title: Test
components:
  schemas:
    Spec:
      type: object
      properties:
        field:
          anyOf:
            - type: string
            - type: integer
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		_, err := ParseOpenAPISpec(specPath)
		if err == nil {
			t.Error("Expected error for anyOf")
		}
	})

	t.Run("error on allOf", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `openapi: 3.0.3
info:
  title: Test
components:
  schemas:
    Spec:
      type: object
      properties:
        field:
          allOf:
            - type: object
              properties:
                name:
                  type: string
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		_, err := ParseOpenAPISpec(specPath)
		if err == nil {
			t.Error("Expected error for allOf")
		}
	})

	t.Run("error on array missing items", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `openapi: 3.0.3
info:
  title: Test
components:
  schemas:
    Spec:
      type: object
      properties:
        arrayField:
          type: array
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		_, err := ParseOpenAPISpec(specPath)
		if err == nil {
			t.Error("Expected error for array missing items")
		}
	})
}

func TestPropertySchemaGoType(t *testing.T) {
	tests := []struct {
		name     string
		prop     PropertySchema
		expected string
	}{
		{
			name:     "string",
			prop:     PropertySchema{Type: "string"},
			expected: "string",
		},
		{
			name:     "boolean",
			prop:     PropertySchema{Type: "boolean"},
			expected: "bool",
		},
		{
			name:     "integer",
			prop:     PropertySchema{Type: "integer"},
			expected: "int",
		},
		{
			name:     "number",
			prop:     PropertySchema{Type: "number"},
			expected: "float64",
		},
		{
			name:     "array of strings",
			prop:     PropertySchema{Type: "array", Items: &PropertySchema{Type: "string"}},
			expected: "[]string",
		},
		{
			name:     "array of integers",
			prop:     PropertySchema{Type: "array", Items: &PropertySchema{Type: "integer"}},
			expected: "[]int",
		},
		{
			name:     "map of strings",
			prop:     PropertySchema{Type: "object", AdditionalProperties: &PropertySchema{Type: "string"}},
			expected: "map[string]string",
		},
		{
			name:     "array without items",
			prop:     PropertySchema{Type: "array"},
			expected: "[]interface{}",
		},
		{
			name:     "object without additionalProperties",
			prop:     PropertySchema{Type: "object"},
			expected: "interface{}",
		},
		{
			name:     "unknown type",
			prop:     PropertySchema{Type: "unknown"},
			expected: "interface{}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.prop.GoType()
			if got != tt.expected {
				t.Errorf("GoType() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSpecSchemaIsRequired(t *testing.T) {
	schema := &SpecSchema{
		Required: []string{"field1", "field2"},
	}

	if !schema.IsRequired("field1") {
		t.Error("field1 should be required")
	}
	if !schema.IsRequired("field2") {
		t.Error("field2 should be required")
	}
	if schema.IsRequired("field3") {
		t.Error("field3 should not be required")
	}
}

// findProperty finds a property by name in a slice of properties.
func findProperty(props []PropertySchema, name string) *PropertySchema {
	for i := range props {
		if props[i].Name == name {
			return &props[i]
		}
	}
	return nil
}
