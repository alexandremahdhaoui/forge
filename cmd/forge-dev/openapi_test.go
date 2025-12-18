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
	"testing"
)

// Tests for PropertySchema.GoType() method which is still used by templates.
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
		{
			name:     "ref resolved",
			prop:     PropertySchema{Type: "ref", Ref: "#/components/schemas/VMResource", RefResolved: &NamedSchema{Name: "VMResource"}},
			expected: "VMResource",
		},
		{
			name:     "ref resolved with pointer",
			prop:     PropertySchema{Type: "ref", Ref: "#/components/schemas/VMResource", RefResolved: &NamedSchema{Name: "VMResource"}, UsePointer: true},
			expected: "*VMResource",
		},
		{
			name:     "unresolved ref",
			prop:     PropertySchema{Type: "ref", Ref: "#/components/schemas/VMResource"},
			expected: "interface{}",
		},
		{
			name: "array ref resolved",
			prop: PropertySchema{
				Type: "array",
				Items: &PropertySchema{
					Type:        "ref",
					Ref:         "#/components/schemas/VMResource",
					RefResolved: &NamedSchema{Name: "VMResource"},
				},
			},
			expected: "[]VMResource",
		},
		{
			name: "array ref resolved with pointer",
			prop: PropertySchema{
				Type: "array",
				Items: &PropertySchema{
					Type:        "ref",
					Ref:         "#/components/schemas/VMResource",
					RefResolved: &NamedSchema{Name: "VMResource"},
					UsePointer:  true,
				},
			},
			expected: "[]*VMResource",
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

// Tests for SpecSchema.IsRequired() method which is still used.
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

// Tests for extractSchemaName which is still used by the adapter.
func TestExtractSchemaName(t *testing.T) {
	tests := []struct {
		ref      string
		expected string
	}{
		{"#/components/schemas/VMResource", "VMResource"},
		{"#/components/schemas/Network", "Network"},
		{"#/components/schemas/Spec", "Spec"},
		{"./external.yaml#/components/schemas/X", ""},
		{"", ""},
		{"#/definitions/X", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			got := extractSchemaName(tt.ref)
			if got != tt.expected {
				t.Errorf("extractSchemaName(%q) = %q, want %q", tt.ref, got, tt.expected)
			}
		})
	}
}

// Tests for SchemaRegistry methods which are reused by the new code.
func TestSchemaRegistry_Basic(t *testing.T) {
	t.Run("NewSchemaRegistry creates empty registry", func(t *testing.T) {
		registry := NewSchemaRegistry()
		if registry == nil {
			t.Fatal("NewSchemaRegistry returned nil")
		}
		if registry.Get("nonexistent") != nil {
			t.Error("Get on empty registry should return nil")
		}
		if registry.GetSpec() != nil {
			t.Error("GetSpec on empty registry should return nil")
		}
	})

	t.Run("Register and Get", func(t *testing.T) {
		registry := NewSchemaRegistry()
		schema := &NamedSchema{Name: "TestSchema"}
		registry.Register("TestSchema", schema)

		got := registry.Get("TestSchema")
		if got != schema {
			t.Errorf("Get returned wrong schema: got %v, want %v", got, schema)
		}
	})

	t.Run("GetSpec returns Spec schema", func(t *testing.T) {
		registry := NewSchemaRegistry()
		specSchema := &NamedSchema{Name: "Spec"}
		otherSchema := &NamedSchema{Name: "Other"}
		registry.Register("Spec", specSchema)
		registry.Register("Other", otherSchema)

		got := registry.GetSpec()
		if got != specSchema {
			t.Errorf("GetSpec returned wrong schema: got %v, want %v", got, specSchema)
		}
	})
}

// Tests for the topological sort using Tarjan's SCC algorithm.
func TestSchemaRegistry_ComputeOrder(t *testing.T) {
	t.Run("single schema with no dependencies", func(t *testing.T) {
		registry := NewSchemaRegistry()
		registry.Register("Spec", &NamedSchema{
			Name:       "Spec",
			Properties: []PropertySchema{{Name: "field", Type: "string"}},
		})

		if err := registry.ComputeOrder(); err != nil {
			t.Fatalf("ComputeOrder failed: %v", err)
		}

		order := registry.GetGenerationOrder()
		if len(order) != 1 || order[0] != "Spec" {
			t.Errorf("Expected order [Spec], got %v", order)
		}
	})

	t.Run("dependencies ordered before dependents", func(t *testing.T) {
		registry := NewSchemaRegistry()

		// VMResource has no dependencies
		registry.Register("VMResource", &NamedSchema{
			Name:       "VMResource",
			Properties: []PropertySchema{{Name: "name", Type: "string"}},
		})

		// Spec depends on VMResource
		registry.Register("Spec", &NamedSchema{
			Name: "Spec",
			Properties: []PropertySchema{{
				Name: "vm",
				Ref:  "#/components/schemas/VMResource",
			}},
		})

		if err := registry.ComputeOrder(); err != nil {
			t.Fatalf("ComputeOrder failed: %v", err)
		}

		order := registry.GetGenerationOrder()
		if len(order) != 2 {
			t.Fatalf("Expected 2 schemas, got %d", len(order))
		}

		// VMResource must come before Spec
		vmIndex, specIndex := -1, -1
		for i, name := range order {
			if name == "VMResource" {
				vmIndex = i
			}
			if name == "Spec" {
				specIndex = i
			}
		}
		if vmIndex >= specIndex {
			t.Errorf("VMResource (index %d) should come before Spec (index %d)", vmIndex, specIndex)
		}
	})

	t.Run("alphabetical order for independent schemas", func(t *testing.T) {
		registry := NewSchemaRegistry()

		// No dependencies between schemas
		registry.Register("Zebra", &NamedSchema{Name: "Zebra", Properties: []PropertySchema{{Name: "name", Type: "string"}}})
		registry.Register("Apple", &NamedSchema{Name: "Apple", Properties: []PropertySchema{{Name: "name", Type: "string"}}})
		registry.Register("Middle", &NamedSchema{Name: "Middle", Properties: []PropertySchema{{Name: "name", Type: "string"}}})

		if err := registry.ComputeOrder(); err != nil {
			t.Fatalf("ComputeOrder failed: %v", err)
		}

		order := registry.GetGenerationOrder()
		expected := []string{"Apple", "Middle", "Zebra"}
		for i, name := range expected {
			if order[i] != name {
				t.Errorf("Expected order[%d] = %q, got %q", i, name, order[i])
			}
		}
	})
}

// Tests for circular reference detection using Tarjan's SCC.
func TestSchemaRegistry_CircularRefs(t *testing.T) {
	t.Run("self-referencing schema marks UsePointer", func(t *testing.T) {
		registry := NewSchemaRegistry()

		// LinkedList -> LinkedList (self-reference)
		registry.Register("LinkedList", &NamedSchema{
			Name: "LinkedList",
			Properties: []PropertySchema{
				{Name: "value", Type: "string"},
				{Name: "next", Ref: "#/components/schemas/LinkedList"},
			},
		})

		if err := registry.ComputeOrder(); err != nil {
			t.Fatalf("ComputeOrder failed: %v", err)
		}

		schema := registry.Get("LinkedList")
		nextProp := findProperty(schema.Properties, "next")
		if nextProp == nil {
			t.Fatal("next property not found")
		}
		if !nextProp.UsePointer {
			t.Error("next.UsePointer should be true for self-reference")
		}
	})

	t.Run("mutual recursion marks UsePointer on both", func(t *testing.T) {
		registry := NewSchemaRegistry()

		// Parent -> Child, Child -> Parent
		registry.Register("Parent", &NamedSchema{
			Name: "Parent",
			Properties: []PropertySchema{
				{Name: "name", Type: "string"},
				{Name: "child", Ref: "#/components/schemas/Child"},
			},
		})
		registry.Register("Child", &NamedSchema{
			Name: "Child",
			Properties: []PropertySchema{
				{Name: "name", Type: "string"},
				{Name: "parent", Ref: "#/components/schemas/Parent"},
			},
		})

		if err := registry.ComputeOrder(); err != nil {
			t.Fatalf("ComputeOrder failed: %v", err)
		}

		parentSchema := registry.Get("Parent")
		childSchema := registry.Get("Child")

		childProp := findProperty(parentSchema.Properties, "child")
		parentProp := findProperty(childSchema.Properties, "parent")

		if !childProp.UsePointer {
			t.Error("child.UsePointer should be true in cycle")
		}
		if !parentProp.UsePointer {
			t.Error("parent.UsePointer should be true in cycle")
		}
	})

	t.Run("array items self-reference marks UsePointer", func(t *testing.T) {
		registry := NewSchemaRegistry()

		// Node has array of Node children
		registry.Register("Node", &NamedSchema{
			Name: "Node",
			Properties: []PropertySchema{
				{Name: "value", Type: "string"},
				{
					Name: "children",
					Type: "array",
					Items: &PropertySchema{
						Ref: "#/components/schemas/Node",
					},
				},
			},
		})

		if err := registry.ComputeOrder(); err != nil {
			t.Fatalf("ComputeOrder failed: %v", err)
		}

		schema := registry.Get("Node")
		childrenProp := findProperty(schema.Properties, "children")
		if childrenProp == nil || childrenProp.Items == nil {
			t.Fatal("children property or Items not found")
		}
		if !childrenProp.Items.UsePointer {
			t.Error("children.Items.UsePointer should be true for self-reference")
		}
	})
}
