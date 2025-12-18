//go:build unit

// Copyright 2025 Alexandre Mahdhaoui
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

	"github.com/getkin/kin-openapi/openapi3"
)

// TestLoadOpenAPISpec tests the LoadOpenAPISpec function.
func TestLoadOpenAPISpec(t *testing.T) {
	t.Run("valid spec loads successfully", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `openapi: '3.0.0'
info:
  title: Test
  version: '1.0'
components:
  schemas:
    Spec:
      type: object
      properties:
        name:
          type: string
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		spec, err := LoadOpenAPISpec(specPath)
		if err != nil {
			t.Fatalf("LoadOpenAPISpec failed: %v", err)
		}

		if spec == nil {
			t.Fatal("spec is nil")
		}
		if spec.OpenAPI != "3.0.0" {
			t.Errorf("OpenAPI version = %q, want %q", spec.OpenAPI, "3.0.0")
		}
		if spec.Components == nil {
			t.Fatal("spec.Components is nil")
		}
		if spec.Components.Schemas == nil {
			t.Fatal("spec.Components.Schemas is nil")
		}
		if _, ok := spec.Components.Schemas["Spec"]; !ok {
			t.Error("Spec schema not found")
		}
	})

	t.Run("invalid path returns error", func(t *testing.T) {
		_, err := LoadOpenAPISpec("/nonexistent/path/spec.yaml")
		if err == nil {
			t.Error("Expected error for missing file")
		}
	})

	t.Run("invalid YAML returns error", func(t *testing.T) {
		dir := t.TempDir()
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte("invalid: yaml: content: ["), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		_, err := LoadOpenAPISpec(specPath)
		if err == nil {
			t.Error("Expected error for invalid YAML")
		}
	})

	t.Run("missing openapi field returns error", func(t *testing.T) {
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

		_, err := LoadOpenAPISpec(specPath)
		if err == nil {
			t.Error("Expected error for missing openapi field")
		}
	})
}

// TestConvertSchemaProperty tests the ConvertSchemaProperty function.
func TestConvertSchemaProperty(t *testing.T) {
	t.Run("string type property", func(t *testing.T) {
		schema := &openapi3.Schema{
			Type:        &openapi3.Types{"string"},
			Description: "A string field",
		}
		schemaRef := &openapi3.SchemaRef{Value: schema}

		prop := ConvertSchemaProperty("string_field", schemaRef, true)

		if prop.Name != "StringField" {
			t.Errorf("Name = %q, want %q", prop.Name, "StringField")
		}
		if prop.JsonName != "string_field" {
			t.Errorf("JsonName = %q, want %q", prop.JsonName, "string_field")
		}
		if prop.GoType != "string" {
			t.Errorf("GoType = %q, want %q", prop.GoType, "string")
		}
		if prop.Description != "A string field" {
			t.Errorf("Description = %q, want %q", prop.Description, "A string field")
		}
		if !prop.Required {
			t.Error("Required should be true")
		}
	})

	t.Run("integer type property", func(t *testing.T) {
		schema := &openapi3.Schema{
			Type: &openapi3.Types{"integer"},
		}
		schemaRef := &openapi3.SchemaRef{Value: schema}

		prop := ConvertSchemaProperty("count", schemaRef, false)

		if prop.GoType != "int" {
			t.Errorf("GoType = %q, want %q", prop.GoType, "int")
		}
		if prop.Required {
			t.Error("Required should be false")
		}
	})

	t.Run("integer with int32 format", func(t *testing.T) {
		schema := &openapi3.Schema{
			Type:   &openapi3.Types{"integer"},
			Format: "int32",
		}
		schemaRef := &openapi3.SchemaRef{Value: schema}

		prop := ConvertSchemaProperty("count", schemaRef, false)

		if prop.GoType != "int32" {
			t.Errorf("GoType = %q, want %q", prop.GoType, "int32")
		}
	})

	t.Run("integer with int64 format", func(t *testing.T) {
		schema := &openapi3.Schema{
			Type:   &openapi3.Types{"integer"},
			Format: "int64",
		}
		schemaRef := &openapi3.SchemaRef{Value: schema}

		prop := ConvertSchemaProperty("count", schemaRef, false)

		if prop.GoType != "int64" {
			t.Errorf("GoType = %q, want %q", prop.GoType, "int64")
		}
	})

	t.Run("boolean type property", func(t *testing.T) {
		schema := &openapi3.Schema{
			Type: &openapi3.Types{"boolean"},
		}
		schemaRef := &openapi3.SchemaRef{Value: schema}

		prop := ConvertSchemaProperty("enabled", schemaRef, false)

		if prop.GoType != "bool" {
			t.Errorf("GoType = %q, want %q", prop.GoType, "bool")
		}
	})

	t.Run("number type property", func(t *testing.T) {
		schema := &openapi3.Schema{
			Type: &openapi3.Types{"number"},
		}
		schemaRef := &openapi3.SchemaRef{Value: schema}

		prop := ConvertSchemaProperty("ratio", schemaRef, false)

		if prop.GoType != "float64" {
			t.Errorf("GoType = %q, want %q", prop.GoType, "float64")
		}
	})

	t.Run("number with float format", func(t *testing.T) {
		schema := &openapi3.Schema{
			Type:   &openapi3.Types{"number"},
			Format: "float",
		}
		schemaRef := &openapi3.SchemaRef{Value: schema}

		prop := ConvertSchemaProperty("ratio", schemaRef, false)

		if prop.GoType != "float32" {
			t.Errorf("GoType = %q, want %q", prop.GoType, "float32")
		}
	})

	t.Run("array of strings property", func(t *testing.T) {
		schema := &openapi3.Schema{
			Type: &openapi3.Types{"array"},
			Items: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: &openapi3.Types{"string"},
				},
			},
		}
		schemaRef := &openapi3.SchemaRef{Value: schema}

		prop := ConvertSchemaProperty("tags", schemaRef, false)

		if !prop.IsArray {
			t.Error("IsArray should be true")
		}
		if prop.ArrayItemType != "string" {
			t.Errorf("ArrayItemType = %q, want %q", prop.ArrayItemType, "string")
		}
		if prop.GoType != "[]string" {
			t.Errorf("GoType = %q, want %q", prop.GoType, "[]string")
		}
	})

	t.Run("array of $ref property (IsArrayOfRef)", func(t *testing.T) {
		schema := &openapi3.Schema{
			Type: &openapi3.Types{"array"},
			Items: &openapi3.SchemaRef{
				Ref: "#/components/schemas/VMResource",
				Value: &openapi3.Schema{
					Type: &openapi3.Types{"object"},
				},
			},
		}
		schemaRef := &openapi3.SchemaRef{Value: schema}

		prop := ConvertSchemaProperty("vms", schemaRef, false)

		if !prop.IsArray {
			t.Error("IsArray should be true")
		}
		if !prop.IsArrayOfRef {
			t.Error("IsArrayOfRef should be true")
		}
		if prop.ArrayItemType != "VMResource" {
			t.Errorf("ArrayItemType = %q, want %q", prop.ArrayItemType, "VMResource")
		}
		if prop.GoType != "[]VMResource" {
			t.Errorf("GoType = %q, want %q", prop.GoType, "[]VMResource")
		}
	})

	t.Run("map type property (additionalProperties)", func(t *testing.T) {
		hasAdditionalProps := true
		schema := &openapi3.Schema{
			Type: &openapi3.Types{"object"},
			AdditionalProperties: openapi3.AdditionalProperties{
				Has: &hasAdditionalProps,
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"string"},
					},
				},
			},
		}
		schemaRef := &openapi3.SchemaRef{Value: schema}

		prop := ConvertSchemaProperty("labels", schemaRef, false)

		if !prop.IsMap {
			t.Error("IsMap should be true")
		}
		if prop.MapValueType != "string" {
			t.Errorf("MapValueType = %q, want %q", prop.MapValueType, "string")
		}
		if prop.GoType != "map[string]string" {
			t.Errorf("GoType = %q, want %q", prop.GoType, "map[string]string")
		}
	})

	t.Run("$ref type property (IsRef)", func(t *testing.T) {
		schemaRef := &openapi3.SchemaRef{
			Ref: "#/components/schemas/VMResource",
			Value: &openapi3.Schema{
				Type: &openapi3.Types{"object"},
			},
		}

		prop := ConvertSchemaProperty("vm", schemaRef, true)

		if !prop.IsRef {
			t.Error("IsRef should be true")
		}
		if prop.RefType != "VMResource" {
			t.Errorf("RefType = %q, want %q", prop.RefType, "VMResource")
		}
		if prop.GoType != "VMResource" {
			t.Errorf("GoType = %q, want %q", prop.GoType, "VMResource")
		}
	})

	t.Run("enum type property", func(t *testing.T) {
		schema := &openapi3.Schema{
			Type: &openapi3.Types{"string"},
			Enum: []interface{}{"pending", "running", "completed"},
		}
		schemaRef := &openapi3.SchemaRef{Value: schema}

		prop := ConvertSchemaProperty("status", schemaRef, false)

		if !prop.IsEnum {
			t.Error("IsEnum should be true")
		}
		if len(prop.EnumValues) != 3 {
			t.Errorf("EnumValues length = %d, want 3", len(prop.EnumValues))
		}
		// Enum values should be sorted
		expected := []string{"completed", "pending", "running"}
		for i, v := range expected {
			if prop.EnumValues[i] != v {
				t.Errorf("EnumValues[%d] = %q, want %q", i, prop.EnumValues[i], v)
			}
		}
	})

	t.Run("nullable non-required property uses pointer", func(t *testing.T) {
		schema := &openapi3.Schema{
			Type:     &openapi3.Types{"string"},
			Nullable: true,
		}
		schemaRef := &openapi3.SchemaRef{Value: schema}

		prop := ConvertSchemaProperty("optional_field", schemaRef, false)

		if !prop.IsPointer {
			t.Error("IsPointer should be true for nullable non-required field")
		}
		if prop.GoType != "*string" {
			t.Errorf("GoType = %q, want %q", prop.GoType, "*string")
		}
	})

	t.Run("nullable required property does not use pointer", func(t *testing.T) {
		schema := &openapi3.Schema{
			Type:     &openapi3.Types{"string"},
			Nullable: true,
		}
		schemaRef := &openapi3.SchemaRef{Value: schema}

		prop := ConvertSchemaProperty("required_field", schemaRef, true)

		if prop.IsPointer {
			t.Error("IsPointer should be false for nullable required field")
		}
		if prop.GoType != "string" {
			t.Errorf("GoType = %q, want %q", prop.GoType, "string")
		}
	})

	t.Run("nil schema returns interface{}", func(t *testing.T) {
		schemaRef := &openapi3.SchemaRef{Value: nil}

		prop := ConvertSchemaProperty("unknown", schemaRef, false)

		if prop.GoType != "interface{}" {
			t.Errorf("GoType = %q, want %q", prop.GoType, "interface{}")
		}
	})

	t.Run("array without items returns []interface{}", func(t *testing.T) {
		schema := &openapi3.Schema{
			Type:  &openapi3.Types{"array"},
			Items: nil,
		}
		schemaRef := &openapi3.SchemaRef{Value: schema}

		prop := ConvertSchemaProperty("items", schemaRef, false)

		if prop.GoType != "[]interface{}" {
			t.Errorf("GoType = %q, want %q", prop.GoType, "[]interface{}")
		}
	})
}

// TestConvertSchemaToTypeDefinition tests the ConvertSchemaToTypeDefinition function.
func TestConvertSchemaToTypeDefinition(t *testing.T) {
	t.Run("simple object with properties", func(t *testing.T) {
		schema := &openapi3.Schema{
			Type:        &openapi3.Types{"object"},
			Description: "A simple object",
			Required:    []string{"name"},
			Properties: map[string]*openapi3.SchemaRef{
				"name": {
					Value: &openapi3.Schema{
						Type:        &openapi3.Types{"string"},
						Description: "The name",
					},
				},
				"count": {
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"integer"},
					},
				},
			},
		}
		schemaRef := &openapi3.SchemaRef{Value: schema}

		td := ConvertSchemaToTypeDefinition("SimpleObject", schemaRef)

		if td.Name != "SimpleObject" {
			t.Errorf("Name = %q, want %q", td.Name, "SimpleObject")
		}
		if td.JsonName != "SimpleObject" {
			t.Errorf("JsonName = %q, want %q", td.JsonName, "SimpleObject")
		}
		if td.Description != "A simple object" {
			t.Errorf("Description = %q, want %q", td.Description, "A simple object")
		}
		if td.IsUnion {
			t.Error("IsUnion should be false")
		}
		if td.IsEnum {
			t.Error("IsEnum should be false")
		}
		if len(td.Properties) != 2 {
			t.Errorf("Properties length = %d, want 2", len(td.Properties))
		}

		// Properties should be sorted alphabetically
		if td.Properties[0].JsonName != "count" {
			t.Errorf("First property = %q, want %q", td.Properties[0].JsonName, "count")
		}
		if td.Properties[1].JsonName != "name" {
			t.Errorf("Second property = %q, want %q", td.Properties[1].JsonName, "name")
		}

		// Check required field
		nameProp := td.Properties[1]
		if !nameProp.Required {
			t.Error("name property should be required")
		}
		countProp := td.Properties[0]
		if countProp.Required {
			t.Error("count property should not be required")
		}
	})

	t.Run("union type (oneOf with discriminator)", func(t *testing.T) {
		schema := &openapi3.Schema{
			Description: "A union type",
			OneOf: openapi3.SchemaRefs{
				{Ref: "#/components/schemas/TypeA"},
				{Ref: "#/components/schemas/TypeB"},
			},
			Discriminator: &openapi3.Discriminator{
				PropertyName: "kind",
				Mapping: map[string]string{
					"a": "#/components/schemas/TypeA",
					"b": "#/components/schemas/TypeB",
				},
			},
		}
		schemaRef := &openapi3.SchemaRef{Value: schema}

		td := ConvertSchemaToTypeDefinition("UnionType", schemaRef)

		if !td.IsUnion {
			t.Error("IsUnion should be true")
		}
		if len(td.UnionVariants) != 2 {
			t.Errorf("UnionVariants length = %d, want 2", len(td.UnionVariants))
		}
		// Variants should be sorted
		if td.UnionVariants[0] != "TypeA" || td.UnionVariants[1] != "TypeB" {
			t.Errorf("UnionVariants = %v, want [TypeA, TypeB]", td.UnionVariants)
		}
		if td.DiscriminatorField != "kind" {
			t.Errorf("DiscriminatorField = %q, want %q", td.DiscriminatorField, "kind")
		}
		if len(td.DiscriminatorMapping) != 2 {
			t.Errorf("DiscriminatorMapping length = %d, want 2", len(td.DiscriminatorMapping))
		}
		if td.DiscriminatorMapping["a"] != "TypeA" {
			t.Errorf("DiscriminatorMapping[a] = %q, want %q", td.DiscriminatorMapping["a"], "TypeA")
		}
	})

	t.Run("union type (anyOf)", func(t *testing.T) {
		schema := &openapi3.Schema{
			AnyOf: openapi3.SchemaRefs{
				{Ref: "#/components/schemas/TypeX"},
				{Ref: "#/components/schemas/TypeY"},
			},
		}
		schemaRef := &openapi3.SchemaRef{Value: schema}

		td := ConvertSchemaToTypeDefinition("AnyOfUnion", schemaRef)

		if !td.IsUnion {
			t.Error("IsUnion should be true for anyOf")
		}
		if len(td.UnionVariants) != 2 {
			t.Errorf("UnionVariants length = %d, want 2", len(td.UnionVariants))
		}
	})

	t.Run("enum type schema", func(t *testing.T) {
		schema := &openapi3.Schema{
			Type: &openapi3.Types{"string"},
			Enum: []interface{}{"red", "green", "blue"},
		}
		schemaRef := &openapi3.SchemaRef{Value: schema}

		td := ConvertSchemaToTypeDefinition("Color", schemaRef)

		if !td.IsEnum {
			t.Error("IsEnum should be true")
		}
		if len(td.EnumValues) != 3 {
			t.Errorf("EnumValues length = %d, want 3", len(td.EnumValues))
		}
		// Enum values should be sorted
		expected := []string{"blue", "green", "red"}
		for i, v := range expected {
			if td.EnumValues[i] != v {
				t.Errorf("EnumValues[%d] = %q, want %q", i, td.EnumValues[i], v)
			}
		}
	})

	t.Run("nil schemaRef returns empty definition", func(t *testing.T) {
		td := ConvertSchemaToTypeDefinition("Empty", nil)

		if td.Name != "Empty" {
			t.Errorf("Name = %q, want %q", td.Name, "Empty")
		}
		if len(td.Properties) != 0 {
			t.Errorf("Properties length = %d, want 0", len(td.Properties))
		}
	})

	t.Run("nil schema value returns empty definition", func(t *testing.T) {
		schemaRef := &openapi3.SchemaRef{Value: nil}

		td := ConvertSchemaToTypeDefinition("NilValue", schemaRef)

		if td.Name != "NilValue" {
			t.Errorf("Name = %q, want %q", td.Name, "NilValue")
		}
		if len(td.Properties) != 0 {
			t.Errorf("Properties length = %d, want 0", len(td.Properties))
		}
	})
}

// TestGenerateForgeTypes tests the GenerateForgeTypes function.
func TestGenerateForgeTypes(t *testing.T) {
	t.Run("simple spec with Spec schema", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `openapi: '3.0.0'
info:
  title: Test
  version: '1.0'
components:
  schemas:
    Spec:
      type: object
      properties:
        name:
          type: string
        timeout:
          type: integer
      required:
        - name
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		spec, err := LoadOpenAPISpec(specPath)
		if err != nil {
			t.Fatalf("LoadOpenAPISpec failed: %v", err)
		}

		types, err := GenerateForgeTypes(spec, "main")
		if err != nil {
			t.Fatalf("GenerateForgeTypes failed: %v", err)
		}

		if len(types) != 1 {
			t.Fatalf("Expected 1 type, got %d", len(types))
		}
		if types[0].Name != "Spec" {
			t.Errorf("Type name = %q, want %q", types[0].Name, "Spec")
		}
		if len(types[0].Properties) != 2 {
			t.Errorf("Properties count = %d, want 2", len(types[0].Properties))
		}
	})

	t.Run("multi-schema spec with dependencies", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `openapi: '3.0.0'
info:
  title: Test
  version: '1.0'
components:
  schemas:
    VMResource:
      type: object
      properties:
        name:
          type: string
        memory:
          type: integer
    Network:
      type: object
      properties:
        id:
          type: string
        vlan:
          type: integer
    Spec:
      type: object
      properties:
        vm:
          $ref: '#/components/schemas/VMResource'
        network:
          $ref: '#/components/schemas/Network'
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		spec, err := LoadOpenAPISpec(specPath)
		if err != nil {
			t.Fatalf("LoadOpenAPISpec failed: %v", err)
		}

		types, err := GenerateForgeTypes(spec, "main")
		if err != nil {
			t.Fatalf("GenerateForgeTypes failed: %v", err)
		}

		if len(types) != 3 {
			t.Fatalf("Expected 3 types, got %d", len(types))
		}

		// Types should be in topological order (dependencies before dependents)
		// VMResource and Network should come before Spec
		typeNames := make([]string, len(types))
		for i, td := range types {
			typeNames[i] = td.Name
		}

		specIndex := -1
		vmIndex := -1
		networkIndex := -1
		for i, name := range typeNames {
			switch name {
			case "Spec":
				specIndex = i
			case "VMResource":
				vmIndex = i
			case "Network":
				networkIndex = i
			}
		}

		if vmIndex >= specIndex {
			t.Errorf("VMResource (index %d) should come before Spec (index %d)", vmIndex, specIndex)
		}
		if networkIndex >= specIndex {
			t.Errorf("Network (index %d) should come before Spec (index %d)", networkIndex, specIndex)
		}
	})

	t.Run("missing Spec schema returns error", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `openapi: '3.0.0'
info:
  title: Test
  version: '1.0'
components:
  schemas:
    OtherSchema:
      type: object
      properties:
        name:
          type: string
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		spec, err := LoadOpenAPISpec(specPath)
		if err != nil {
			t.Fatalf("LoadOpenAPISpec failed: %v", err)
		}

		_, err = GenerateForgeTypes(spec, "main")
		if err == nil {
			t.Error("Expected error for missing Spec schema")
		}
	})

	t.Run("nil components returns error", func(t *testing.T) {
		spec := &openapi3.T{
			OpenAPI: "3.0.0",
			Info: &openapi3.Info{
				Title:   "Test",
				Version: "1.0",
			},
			Components: nil,
		}

		_, err := GenerateForgeTypes(spec, "main")
		if err == nil {
			t.Error("Expected error for nil components")
		}
	})

	t.Run("nil schemas returns error", func(t *testing.T) {
		spec := &openapi3.T{
			OpenAPI: "3.0.0",
			Info: &openapi3.Info{
				Title:   "Test",
				Version: "1.0",
			},
			Components: &openapi3.Components{
				Schemas: nil,
			},
		}

		_, err := GenerateForgeTypes(spec, "main")
		if err == nil {
			t.Error("Expected error for nil schemas")
		}
	})

	t.Run("circular reference with self-reference", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `openapi: '3.0.0'
info:
  title: Test
  version: '1.0'
components:
  schemas:
    Node:
      type: object
      properties:
        value:
          type: string
        children:
          type: array
          items:
            $ref: '#/components/schemas/Node'
    Spec:
      type: object
      properties:
        root:
          $ref: '#/components/schemas/Node'
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		spec, err := LoadOpenAPISpec(specPath)
		if err != nil {
			t.Fatalf("LoadOpenAPISpec failed: %v", err)
		}

		types, err := GenerateForgeTypes(spec, "main")
		if err != nil {
			t.Fatalf("GenerateForgeTypes failed: %v", err)
		}

		if len(types) != 2 {
			t.Fatalf("Expected 2 types, got %d", len(types))
		}

		// Find the Node type
		var nodeType *ForgeTypeDefinition
		for i := range types {
			if types[i].Name == "Node" {
				nodeType = &types[i]
				break
			}
		}
		if nodeType == nil {
			t.Fatal("Node type not found")
		}

		// Find the children property
		var childrenProp *ForgeProperty
		for i := range nodeType.Properties {
			if nodeType.Properties[i].JsonName == "children" {
				childrenProp = &nodeType.Properties[i]
				break
			}
		}
		if childrenProp == nil {
			t.Fatal("children property not found")
		}

		// Self-reference in array should use pointer
		if childrenProp.GoType != "[]*Node" {
			t.Errorf("children.GoType = %q, want %q", childrenProp.GoType, "[]*Node")
		}
	})

	t.Run("circular reference with mutual recursion", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `openapi: '3.0.0'
info:
  title: Test
  version: '1.0'
components:
  schemas:
    Parent:
      type: object
      properties:
        name:
          type: string
        child:
          $ref: '#/components/schemas/Child'
    Child:
      type: object
      properties:
        name:
          type: string
        parent:
          $ref: '#/components/schemas/Parent'
    Spec:
      type: object
      properties:
        root:
          $ref: '#/components/schemas/Parent'
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		spec, err := LoadOpenAPISpec(specPath)
		if err != nil {
			t.Fatalf("LoadOpenAPISpec failed: %v", err)
		}

		types, err := GenerateForgeTypes(spec, "main")
		if err != nil {
			t.Fatalf("GenerateForgeTypes failed: %v", err)
		}

		if len(types) != 3 {
			t.Fatalf("Expected 3 types, got %d", len(types))
		}

		// Find Parent and Child types
		var parentType, childType *ForgeTypeDefinition
		for i := range types {
			switch types[i].Name {
			case "Parent":
				parentType = &types[i]
			case "Child":
				childType = &types[i]
			}
		}
		if parentType == nil || childType == nil {
			t.Fatal("Parent or Child type not found")
		}

		// Find the child property in Parent
		var childProp *ForgeProperty
		for i := range parentType.Properties {
			if parentType.Properties[i].JsonName == "child" {
				childProp = &parentType.Properties[i]
				break
			}
		}

		// Find the parent property in Child
		var parentProp *ForgeProperty
		for i := range childType.Properties {
			if childType.Properties[i].JsonName == "parent" {
				parentProp = &childType.Properties[i]
				break
			}
		}

		// Both should use pointers due to mutual recursion
		if childProp == nil || parentProp == nil {
			t.Fatal("child or parent property not found")
		}

		if !childProp.IsPointer {
			t.Error("child property should use pointer")
		}
		if !parentProp.IsPointer {
			t.Error("parent property should use pointer")
		}
	})

	t.Run("diamond dependency", func(t *testing.T) {
		dir := t.TempDir()
		specContent := `openapi: '3.0.0'
info:
  title: Test
  version: '1.0'
components:
  schemas:
    D:
      type: object
      properties:
        value:
          type: string
    B:
      type: object
      properties:
        d:
          $ref: '#/components/schemas/D'
    C:
      type: object
      properties:
        d:
          $ref: '#/components/schemas/D'
    Spec:
      type: object
      properties:
        b:
          $ref: '#/components/schemas/B'
        c:
          $ref: '#/components/schemas/C'
`
		specPath := filepath.Join(dir, "spec.openapi.yaml")
		if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		spec, err := LoadOpenAPISpec(specPath)
		if err != nil {
			t.Fatalf("LoadOpenAPISpec failed: %v", err)
		}

		types, err := GenerateForgeTypes(spec, "main")
		if err != nil {
			t.Fatalf("GenerateForgeTypes failed: %v", err)
		}

		if len(types) != 4 {
			t.Fatalf("Expected 4 types, got %d", len(types))
		}

		// Verify topological order
		typeIndices := make(map[string]int)
		for i, td := range types {
			typeIndices[td.Name] = i
		}

		// D should come before B and C
		if typeIndices["D"] >= typeIndices["B"] {
			t.Errorf("D (index %d) should come before B (index %d)", typeIndices["D"], typeIndices["B"])
		}
		if typeIndices["D"] >= typeIndices["C"] {
			t.Errorf("D (index %d) should come before C (index %d)", typeIndices["D"], typeIndices["C"])
		}

		// B and C should come before Spec
		if typeIndices["B"] >= typeIndices["Spec"] {
			t.Errorf("B (index %d) should come before Spec (index %d)", typeIndices["B"], typeIndices["Spec"])
		}
		if typeIndices["C"] >= typeIndices["Spec"] {
			t.Errorf("C (index %d) should come before Spec (index %d)", typeIndices["C"], typeIndices["Spec"])
		}
	})
}

// TestSchemaToGoType tests the schemaToGoType helper function.
func TestSchemaToGoType(t *testing.T) {
	tests := []struct {
		name     string
		schema   *openapi3.Schema
		expected string
	}{
		{
			name:     "nil schema",
			schema:   nil,
			expected: "interface{}",
		},
		{
			name: "string type",
			schema: &openapi3.Schema{
				Type: &openapi3.Types{"string"},
			},
			expected: "string",
		},
		{
			name: "integer type",
			schema: &openapi3.Schema{
				Type: &openapi3.Types{"integer"},
			},
			expected: "int",
		},
		{
			name: "integer int32 format",
			schema: &openapi3.Schema{
				Type:   &openapi3.Types{"integer"},
				Format: "int32",
			},
			expected: "int32",
		},
		{
			name: "integer int64 format",
			schema: &openapi3.Schema{
				Type:   &openapi3.Types{"integer"},
				Format: "int64",
			},
			expected: "int64",
		},
		{
			name: "number type",
			schema: &openapi3.Schema{
				Type: &openapi3.Types{"number"},
			},
			expected: "float64",
		},
		{
			name: "number float format",
			schema: &openapi3.Schema{
				Type:   &openapi3.Types{"number"},
				Format: "float",
			},
			expected: "float32",
		},
		{
			name: "boolean type",
			schema: &openapi3.Schema{
				Type: &openapi3.Types{"boolean"},
			},
			expected: "bool",
		},
		{
			name: "array of strings",
			schema: &openapi3.Schema{
				Type: &openapi3.Types{"array"},
				Items: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"string"},
					},
				},
			},
			expected: "[]string",
		},
		{
			name: "array without items",
			schema: &openapi3.Schema{
				Type: &openapi3.Types{"array"},
			},
			expected: "[]interface{}",
		},
		{
			name: "map of strings",
			schema: func() *openapi3.Schema {
				hasAdditionalProps := true
				return &openapi3.Schema{
					Type: &openapi3.Types{"object"},
					AdditionalProperties: openapi3.AdditionalProperties{
						Has: &hasAdditionalProps,
						Schema: &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type: &openapi3.Types{"string"},
							},
						},
					},
				}
			}(),
			expected: "map[string]string",
		},
		{
			name: "map without schema",
			schema: func() *openapi3.Schema {
				hasAdditionalProps := true
				return &openapi3.Schema{
					Type: &openapi3.Types{"object"},
					AdditionalProperties: openapi3.AdditionalProperties{
						Has: &hasAdditionalProps,
					},
				}
			}(),
			expected: "map[string]interface{}",
		},
		{
			name: "object with properties (returns interface{})",
			schema: &openapi3.Schema{
				Type: &openapi3.Types{"object"},
				Properties: map[string]*openapi3.SchemaRef{
					"name": {Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
				},
			},
			expected: "interface{}",
		},
		{
			name: "unknown type",
			schema: &openapi3.Schema{
				Type: &openapi3.Types{"unknown"},
			},
			expected: "interface{}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := schemaToGoType(tt.schema)
			if got != tt.expected {
				t.Errorf("schemaToGoType() = %q, want %q", got, tt.expected)
			}
		})
	}
}
