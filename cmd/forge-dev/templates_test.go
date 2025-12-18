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
	"io/fs"
	"strings"
	"testing"
)

func TestTemplatesFS_NotEmpty(t *testing.T) {
	// Verify that templates are properly embedded
	entries, err := fs.ReadDir(templatesFS, "templates")
	if err != nil {
		t.Fatalf("Failed to read embedded templates: %v", err)
	}

	if len(entries) == 0 {
		t.Error("No templates found in embedded filesystem")
	}

	// Check for expected template files
	expectedTemplates := []string{
		"spec.go.tmpl",
		"validate.go.tmpl",
		"mcp_builder.go.tmpl",
		"mcp_testrunner.go.tmpl",
		"mcp_testenv.go.tmpl",
		"mcp_dependency_detector.go.tmpl",
	}

	foundTemplates := make(map[string]bool)
	for _, entry := range entries {
		foundTemplates[entry.Name()] = true
	}

	for _, expected := range expectedTemplates {
		if !foundTemplates[expected] {
			t.Errorf("Expected template %q not found in embedded filesystem", expected)
		}
	}
}

func TestTemplatesFS_ReadContent(t *testing.T) {
	// Verify that template content can be read
	templates := []struct {
		name     string
		contains []string
	}{
		{
			name:     "spec.go.tmpl",
			contains: []string{"type Spec struct", "func FromMap(", "func (s *Spec) ToMap()"},
		},
		{
			name:     "validate.go.tmpl",
			contains: []string{"func Validate(", "func ValidateMap(", "ConfigValidateOutput"},
		},
		{
			name:     "mcp_builder.go.tmpl",
			contains: []string{"type BuildFunc func(", "SetupMCPServer", "wrapBuildFunc"},
		},
		{
			name:     "mcp_testrunner.go.tmpl",
			contains: []string{"type TestRunnerFunc func(", "SetupMCPServer", "wrapTestRunnerFunc"},
		},
		{
			name:     "mcp_testenv.go.tmpl",
			contains: []string{"type CreateFunc func(", "type DeleteFunc func(", "SetupMCPServer"},
		},
		{
			name:     "mcp_dependency_detector.go.tmpl",
			contains: []string{"SetupMCPServerBase", "handleConfigValidate", "config-validate"},
		},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			content, err := fs.ReadFile(templatesFS, "templates/"+tt.name)
			if err != nil {
				t.Fatalf("Failed to read template %s: %v", tt.name, err)
			}

			if len(content) == 0 {
				t.Errorf("Template %s is empty", tt.name)
			}

			contentStr := string(content)
			for _, want := range tt.contains {
				if !strings.Contains(contentStr, want) {
					t.Errorf("Template %s missing expected content %q", tt.name, want)
				}
			}
		})
	}
}

func TestParseTemplate(t *testing.T) {
	templates := []string{
		"spec.go.tmpl",
		"validate.go.tmpl",
		"mcp_builder.go.tmpl",
		"mcp_testrunner.go.tmpl",
		"mcp_testenv.go.tmpl",
		"mcp_dependency_detector.go.tmpl",
	}

	for _, name := range templates {
		t.Run(name, func(t *testing.T) {
			tmpl, err := parseTemplate(name)
			if err != nil {
				t.Fatalf("parseTemplate(%s) error = %v", name, err)
			}
			if tmpl == nil {
				t.Errorf("parseTemplate(%s) returned nil template", name)
			}
		})
	}
}

func TestTemplateFuncs(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() string
		expected string
	}{
		{"title empty", func() string { return toTitle("") }, ""},
		{"title lower", func() string { return toTitle("hello") }, "Hello"},
		{"title upper", func() string { return toTitle("Hello") }, "Hello"},
		{"camel empty", func() string { return toCamelCase("") }, ""},
		{"camel snake", func() string { return toCamelCase("hello_world") }, "HelloWorld"},
		{"camel kebab", func() string { return toCamelCase("hello-world") }, "HelloWorld"},
		{"camel mixed", func() string { return toCamelCase("hello-world_test") }, "HelloWorldTest"},
		{"zeroVal string", func() string { return zeroValue("string") }, `""`},
		{"zeroVal bool", func() string { return zeroValue("bool") }, "false"},
		{"zeroVal int", func() string { return zeroValue("int") }, "0"},
		{"zeroVal float64", func() string { return zeroValue("float64") }, "0.0"},
		{"zeroVal slice", func() string { return zeroValue("[]string") }, "nil"},
		{"zeroVal map", func() string { return zeroValue("map[string]string") }, "nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn()
			if got != tt.expected {
				t.Errorf("got %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestIsSimpleType(t *testing.T) {
	tests := []struct {
		goType string
		want   bool
	}{
		{"string", true},
		{"bool", true},
		{"int", true},
		{"float64", true},
		{"[]string", false},
		{"map[string]string", false},
		{"interface{}", false},
	}

	for _, tt := range tests {
		t.Run(tt.goType, func(t *testing.T) {
			got := isSimpleType(tt.goType)
			if got != tt.want {
				t.Errorf("isSimpleType(%s) = %v, want %v", tt.goType, got, tt.want)
			}
		})
	}
}

func TestJsonTag(t *testing.T) {
	tests := []struct {
		name     string
		required bool
		want     string
	}{
		{"field", true, "`json:\"field\"`"},
		{"field", false, "`json:\"field,omitempty\"`"},
		{"myField", true, "`json:\"myField\"`"},
		{"myField", false, "`json:\"myField,omitempty\"`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsonTag(tt.name, tt.required)
			if got != tt.want {
				t.Errorf("jsonTag(%s, %v) = %s, want %s", tt.name, tt.required, got, tt.want)
			}
		})
	}
}

func TestIsRef(t *testing.T) {
	tests := []struct {
		name string
		prop PropertySchema
		want bool
	}{
		{
			name: "property with ref",
			prop: PropertySchema{
				Name: "vm",
				Ref:  "#/components/schemas/VMResource",
			},
			want: true,
		},
		{
			name: "property without ref",
			prop: PropertySchema{
				Name: "name",
				Type: "string",
			},
			want: false,
		},
		{
			name: "empty ref string",
			prop: PropertySchema{
				Name: "data",
				Ref:  "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRef(tt.prop)
			if got != tt.want {
				t.Errorf("isRef() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRefType(t *testing.T) {
	vmSchema := &NamedSchema{Name: "VMResource"}

	tests := []struct {
		name string
		prop PropertySchema
		want string
	}{
		{
			name: "resolved ref returns schema name",
			prop: PropertySchema{
				Name:        "vm",
				Ref:         "#/components/schemas/VMResource",
				RefResolved: vmSchema,
			},
			want: "VMResource",
		},
		{
			name: "unresolved ref returns empty string",
			prop: PropertySchema{
				Name:        "vm",
				Ref:         "#/components/schemas/VMResource",
				RefResolved: nil,
			},
			want: "",
		},
		{
			name: "non-ref property returns empty string",
			prop: PropertySchema{
				Name: "name",
				Type: "string",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := refType(tt.prop)
			if got != tt.want {
				t.Errorf("refType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsArrayRef(t *testing.T) {
	tests := []struct {
		name string
		prop PropertySchema
		want bool
	}{
		{
			name: "array with ref items",
			prop: PropertySchema{
				Name: "vms",
				Type: "array",
				Items: &PropertySchema{
					Ref: "#/components/schemas/VMResource",
				},
			},
			want: true,
		},
		{
			name: "array with non-ref items",
			prop: PropertySchema{
				Name: "tags",
				Type: "array",
				Items: &PropertySchema{
					Type: "string",
				},
			},
			want: false,
		},
		{
			name: "non-array type",
			prop: PropertySchema{
				Name: "name",
				Type: "string",
			},
			want: false,
		},
		{
			name: "array with nil items",
			prop: PropertySchema{
				Name:  "data",
				Type:  "array",
				Items: nil,
			},
			want: false,
		},
		{
			name: "array with empty ref in items",
			prop: PropertySchema{
				Name: "items",
				Type: "array",
				Items: &PropertySchema{
					Type: "string",
					Ref:  "",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isArrayRef(tt.prop)
			if got != tt.want {
				t.Errorf("isArrayRef() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArrayRefType(t *testing.T) {
	vmSchema := &NamedSchema{Name: "VMResource"}

	tests := []struct {
		name string
		prop PropertySchema
		want string
	}{
		{
			name: "array with resolved ref items",
			prop: PropertySchema{
				Name: "vms",
				Type: "array",
				Items: &PropertySchema{
					Ref:         "#/components/schemas/VMResource",
					RefResolved: vmSchema,
				},
			},
			want: "VMResource",
		},
		{
			name: "array with unresolved ref items",
			prop: PropertySchema{
				Name: "vms",
				Type: "array",
				Items: &PropertySchema{
					Ref:         "#/components/schemas/VMResource",
					RefResolved: nil,
				},
			},
			want: "",
		},
		{
			name: "array with nil items",
			prop: PropertySchema{
				Name:  "data",
				Type:  "array",
				Items: nil,
			},
			want: "",
		},
		{
			name: "non-array property",
			prop: PropertySchema{
				Name: "name",
				Type: "string",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := arrayRefType(tt.prop)
			if got != tt.want {
				t.Errorf("arrayRefType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUsePointer(t *testing.T) {
	tests := []struct {
		name string
		prop PropertySchema
		want bool
	}{
		{
			name: "property with UsePointer true",
			prop: PropertySchema{
				Name:       "parent",
				Ref:        "#/components/schemas/Node",
				UsePointer: true,
			},
			want: true,
		},
		{
			name: "property with UsePointer false",
			prop: PropertySchema{
				Name:       "vm",
				Ref:        "#/components/schemas/VMResource",
				UsePointer: false,
			},
			want: false,
		},
		{
			name: "non-ref property",
			prop: PropertySchema{
				Name: "name",
				Type: "string",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := usePointer(tt.prop)
			if got != tt.want {
				t.Errorf("usePointer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestItemsUsePointer(t *testing.T) {
	tests := []struct {
		name string
		prop PropertySchema
		want bool
	}{
		{
			name: "array items with UsePointer true",
			prop: PropertySchema{
				Name: "children",
				Type: "array",
				Items: &PropertySchema{
					Ref:        "#/components/schemas/Node",
					UsePointer: true,
				},
			},
			want: true,
		},
		{
			name: "array items with UsePointer false",
			prop: PropertySchema{
				Name: "vms",
				Type: "array",
				Items: &PropertySchema{
					Ref:        "#/components/schemas/VMResource",
					UsePointer: false,
				},
			},
			want: false,
		},
		{
			name: "property with nil items",
			prop: PropertySchema{
				Name:  "data",
				Type:  "array",
				Items: nil,
			},
			want: false,
		},
		{
			name: "non-array property",
			prop: PropertySchema{
				Name: "name",
				Type: "string",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := itemsUsePointer(tt.prop)
			if got != tt.want {
				t.Errorf("itemsUsePointer() = %v, want %v", got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Tests for ForgeProperty template functions
// -----------------------------------------------------------------------------

func TestForgeGoType(t *testing.T) {
	tests := []struct {
		name string
		prop ForgeProperty
		want string
	}{
		{
			name: "string type",
			prop: ForgeProperty{GoType: "string"},
			want: "string",
		},
		{
			name: "array type",
			prop: ForgeProperty{GoType: "[]string"},
			want: "[]string",
		},
		{
			name: "map type",
			prop: ForgeProperty{GoType: "map[string]string"},
			want: "map[string]string",
		},
		{
			name: "reference type",
			prop: ForgeProperty{GoType: "VMResource"},
			want: "VMResource",
		},
		{
			name: "pointer type",
			prop: ForgeProperty{GoType: "*VMResource"},
			want: "*VMResource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := forgeGoType(tt.prop)
			if got != tt.want {
				t.Errorf("forgeGoType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestForgeIsRef(t *testing.T) {
	tests := []struct {
		name string
		prop ForgeProperty
		want bool
	}{
		{
			name: "is ref",
			prop: ForgeProperty{IsRef: true, RefType: "VMResource"},
			want: true,
		},
		{
			name: "not ref",
			prop: ForgeProperty{IsRef: false, GoType: "string"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := forgeIsRef(tt.prop)
			if got != tt.want {
				t.Errorf("forgeIsRef() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestForgeRefType(t *testing.T) {
	tests := []struct {
		name string
		prop ForgeProperty
		want string
	}{
		{
			name: "has ref type",
			prop: ForgeProperty{IsRef: true, RefType: "VMResource"},
			want: "VMResource",
		},
		{
			name: "no ref type",
			prop: ForgeProperty{IsRef: false, GoType: "string"},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := forgeRefType(tt.prop)
			if got != tt.want {
				t.Errorf("forgeRefType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestForgeIsArrayRef(t *testing.T) {
	tests := []struct {
		name string
		prop ForgeProperty
		want bool
	}{
		{
			name: "array of ref",
			prop: ForgeProperty{IsArray: true, IsArrayOfRef: true, ArrayItemType: "VMResource"},
			want: true,
		},
		{
			name: "array of strings",
			prop: ForgeProperty{IsArray: true, IsArrayOfRef: false, ArrayItemType: "string"},
			want: false,
		},
		{
			name: "not array",
			prop: ForgeProperty{IsArray: false, GoType: "string"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := forgeIsArrayRef(tt.prop)
			if got != tt.want {
				t.Errorf("forgeIsArrayRef() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestForgeArrayRefType(t *testing.T) {
	tests := []struct {
		name string
		prop ForgeProperty
		want string
	}{
		{
			name: "array of ref",
			prop: ForgeProperty{IsArray: true, IsArrayOfRef: true, ArrayItemType: "VMResource"},
			want: "VMResource",
		},
		{
			name: "array of strings",
			prop: ForgeProperty{IsArray: true, ArrayItemType: "string"},
			want: "string",
		},
		{
			name: "not array",
			prop: ForgeProperty{IsArray: false, GoType: "string"},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := forgeArrayRefType(tt.prop)
			if got != tt.want {
				t.Errorf("forgeArrayRefType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestForgeUsePointer(t *testing.T) {
	tests := []struct {
		name string
		prop ForgeProperty
		want bool
	}{
		{
			name: "uses pointer",
			prop: ForgeProperty{IsPointer: true, GoType: "*VMResource"},
			want: true,
		},
		{
			name: "does not use pointer",
			prop: ForgeProperty{IsPointer: false, GoType: "VMResource"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := forgeUsePointer(tt.prop)
			if got != tt.want {
				t.Errorf("forgeUsePointer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestForgeIsArray(t *testing.T) {
	tests := []struct {
		name string
		prop ForgeProperty
		want bool
	}{
		{
			name: "is array",
			prop: ForgeProperty{IsArray: true, GoType: "[]string"},
			want: true,
		},
		{
			name: "not array",
			prop: ForgeProperty{IsArray: false, GoType: "string"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := forgeIsArray(tt.prop)
			if got != tt.want {
				t.Errorf("forgeIsArray() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestForgeArrayItemType(t *testing.T) {
	tests := []struct {
		name string
		prop ForgeProperty
		want string
	}{
		{
			name: "array of strings",
			prop: ForgeProperty{IsArray: true, ArrayItemType: "string"},
			want: "string",
		},
		{
			name: "array of refs",
			prop: ForgeProperty{IsArray: true, IsArrayOfRef: true, ArrayItemType: "VMResource"},
			want: "VMResource",
		},
		{
			name: "not array",
			prop: ForgeProperty{IsArray: false},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := forgeArrayItemType(tt.prop)
			if got != tt.want {
				t.Errorf("forgeArrayItemType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestForgeIsMap(t *testing.T) {
	tests := []struct {
		name string
		prop ForgeProperty
		want bool
	}{
		{
			name: "is map",
			prop: ForgeProperty{IsMap: true, GoType: "map[string]string"},
			want: true,
		},
		{
			name: "not map",
			prop: ForgeProperty{IsMap: false, GoType: "string"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := forgeIsMap(tt.prop)
			if got != tt.want {
				t.Errorf("forgeIsMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestForgeMapValueType(t *testing.T) {
	tests := []struct {
		name string
		prop ForgeProperty
		want string
	}{
		{
			name: "map of strings",
			prop: ForgeProperty{IsMap: true, MapValueType: "string"},
			want: "string",
		},
		{
			name: "map of refs",
			prop: ForgeProperty{IsMap: true, MapValueType: "VMResource"},
			want: "VMResource",
		},
		{
			name: "not map",
			prop: ForgeProperty{IsMap: false},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := forgeMapValueType(tt.prop)
			if got != tt.want {
				t.Errorf("forgeMapValueType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestForgeIsEnum(t *testing.T) {
	tests := []struct {
		name string
		prop ForgeProperty
		want bool
	}{
		{
			name: "is enum",
			prop: ForgeProperty{IsEnum: true, EnumValues: []string{"a", "b", "c"}},
			want: true,
		},
		{
			name: "not enum",
			prop: ForgeProperty{IsEnum: false, GoType: "string"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := forgeIsEnum(tt.prop)
			if got != tt.want {
				t.Errorf("forgeIsEnum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestForgeEnumValues(t *testing.T) {
	tests := []struct {
		name string
		prop ForgeProperty
		want []string
	}{
		{
			name: "has enum values",
			prop: ForgeProperty{IsEnum: true, EnumValues: []string{"a", "b", "c"}},
			want: []string{"a", "b", "c"},
		},
		{
			name: "empty enum values",
			prop: ForgeProperty{IsEnum: true, EnumValues: []string{}},
			want: []string{},
		},
		{
			name: "nil enum values",
			prop: ForgeProperty{IsEnum: false},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := forgeEnumValues(tt.prop)
			if len(got) != len(tt.want) {
				t.Errorf("forgeEnumValues() len = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("forgeEnumValues()[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Tests for ForgeTypeDefinition template functions
// -----------------------------------------------------------------------------

func TestIsUnion(t *testing.T) {
	tests := []struct {
		name string
		td   ForgeTypeDefinition
		want bool
	}{
		{
			name: "is union",
			td:   ForgeTypeDefinition{IsUnion: true, UnionVariants: []string{"TypeA", "TypeB"}},
			want: true,
		},
		{
			name: "not union",
			td:   ForgeTypeDefinition{IsUnion: false},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUnion(tt.td)
			if got != tt.want {
				t.Errorf("isUnion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDiscField(t *testing.T) {
	tests := []struct {
		name string
		td   ForgeTypeDefinition
		want string
	}{
		{
			name: "has discriminator field",
			td:   ForgeTypeDefinition{IsUnion: true, DiscriminatorField: "kind"},
			want: "kind",
		},
		{
			name: "no discriminator field",
			td:   ForgeTypeDefinition{IsUnion: true},
			want: "",
		},
		{
			name: "not union",
			td:   ForgeTypeDefinition{IsUnion: false},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := discField(tt.td)
			if got != tt.want {
				t.Errorf("discField() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDiscMapping(t *testing.T) {
	tests := []struct {
		name string
		td   ForgeTypeDefinition
		want map[string]string
	}{
		{
			name: "has discriminator mapping",
			td: ForgeTypeDefinition{
				IsUnion:              true,
				DiscriminatorField:   "kind",
				DiscriminatorMapping: map[string]string{"a": "TypeA", "b": "TypeB"},
			},
			want: map[string]string{"a": "TypeA", "b": "TypeB"},
		},
		{
			name: "no discriminator mapping",
			td:   ForgeTypeDefinition{IsUnion: true},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := discMapping(tt.td)
			if len(got) != len(tt.want) {
				t.Errorf("discMapping() len = %d, want %d", len(got), len(tt.want))
				return
			}
			for k, v := range got {
				if tt.want[k] != v {
					t.Errorf("discMapping()[%q] = %q, want %q", k, v, tt.want[k])
				}
			}
		})
	}
}

func TestIsTypeEnum(t *testing.T) {
	tests := []struct {
		name string
		td   ForgeTypeDefinition
		want bool
	}{
		{
			name: "is enum",
			td:   ForgeTypeDefinition{IsEnum: true, EnumValues: []string{"a", "b", "c"}},
			want: true,
		},
		{
			name: "not enum",
			td:   ForgeTypeDefinition{IsEnum: false},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTypeEnum(tt.td)
			if got != tt.want {
				t.Errorf("isTypeEnum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnionVariants(t *testing.T) {
	tests := []struct {
		name string
		td   ForgeTypeDefinition
		want []string
	}{
		{
			name: "has variants",
			td:   ForgeTypeDefinition{IsUnion: true, UnionVariants: []string{"TypeA", "TypeB"}},
			want: []string{"TypeA", "TypeB"},
		},
		{
			name: "no variants",
			td:   ForgeTypeDefinition{IsUnion: true, UnionVariants: nil},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unionVariants(tt.td)
			if len(got) != len(tt.want) {
				t.Errorf("unionVariants() len = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("unionVariants()[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestTypeEnumValues(t *testing.T) {
	tests := []struct {
		name string
		td   ForgeTypeDefinition
		want []string
	}{
		{
			name: "has enum values",
			td:   ForgeTypeDefinition{IsEnum: true, EnumValues: []string{"red", "green", "blue"}},
			want: []string{"red", "green", "blue"},
		},
		{
			name: "no enum values",
			td:   ForgeTypeDefinition{IsEnum: true, EnumValues: nil},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := typeEnumValues(tt.td)
			if len(got) != len(tt.want) {
				t.Errorf("typeEnumValues() len = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("typeEnumValues()[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}
