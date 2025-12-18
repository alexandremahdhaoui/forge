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
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestGenerateSpecFile(t *testing.T) {
	tests := []struct {
		name       string
		schema     *SpecSchema
		config     *Config
		checksum   string
		wantFields []string
		wantFuncs  []string
		wantErr    bool
	}{
		{
			name: "basic string fields",
			schema: &SpecSchema{
				Properties: []PropertySchema{
					{Name: "name", Type: "string", Required: true, Description: "The name field"},
					{Name: "version", Type: "string", Required: false, Description: "Optional version"},
				},
				Required: []string{"name"},
			},
			config: &Config{
				Name: "test-engine",
				Type: EngineTypeBuilder,
				Generate: GenerateConfig{
					PackageName: "main",
				},
			},
			checksum:   "sha256:abc123",
			wantFields: []string{"Name string", "Version string"},
			wantFuncs:  []string{"func FromMap(", "func (s *Spec) ToMap()"},
			wantErr:    false,
		},
		{
			name: "multiple types",
			schema: &SpecSchema{
				Properties: []PropertySchema{
					{Name: "count", Type: "integer", Required: false, Description: "Item count"},
					{Name: "enabled", Type: "boolean", Required: false, Description: "Is enabled"},
					{Name: "ratio", Type: "number", Required: false, Description: "A ratio"},
				},
			},
			config: &Config{
				Name: "test-engine",
				Type: EngineTypeBuilder,
				Generate: GenerateConfig{
					PackageName: "main",
				},
			},
			checksum:   "sha256:def456",
			wantFields: []string{"Count int", "Enabled bool", "Ratio float64"},
			wantFuncs:  []string{"func FromMap(", "func (s *Spec) ToMap()"},
			wantErr:    false,
		},
		{
			name: "array and map types",
			schema: &SpecSchema{
				Properties: []PropertySchema{
					{
						Name:        "args",
						Type:        "array",
						Description: "Arguments",
						Items:       &PropertySchema{Type: "string"},
					},
					{
						Name:                 "env",
						Type:                 "object",
						Description:          "Environment variables",
						AdditionalProperties: &PropertySchema{Type: "string"},
					},
				},
			},
			config: &Config{
				Name: "test-engine",
				Type: EngineTypeBuilder,
				Generate: GenerateConfig{
					PackageName: "main",
				},
			},
			checksum:   "sha256:ghi789",
			wantFields: []string{"Args []string", "Env map[string]string"},
			wantFuncs:  []string{"func FromMap(", "func (s *Spec) ToMap()"},
			wantErr:    false,
		},
		{
			name: "empty schema",
			schema: &SpecSchema{
				Properties: []PropertySchema{},
			},
			config: &Config{
				Name: "test-engine",
				Type: EngineTypeBuilder,
				Generate: GenerateConfig{
					PackageName: "main",
				},
			},
			checksum:   "sha256:empty",
			wantFields: []string{},
			wantFuncs:  []string{"func FromMap(", "func (s *Spec) ToMap()"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pass nil registry for backwards compatibility tests
			got, err := GenerateSpecFile(tt.schema, tt.config, tt.checksum, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateSpecFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			code := string(got)

			// Check that the code contains expected fields
			for _, field := range tt.wantFields {
				if !strings.Contains(code, field) {
					t.Errorf("Generated code missing field %q", field)
				}
			}

			// Check that the code contains expected functions
			for _, fn := range tt.wantFuncs {
				if !strings.Contains(code, fn) {
					t.Errorf("Generated code missing function %q", fn)
				}
			}

			// Check that the code contains the checksum
			if !strings.Contains(code, tt.checksum) {
				t.Errorf("Generated code missing checksum %q", tt.checksum)
			}

			// Verify the generated code compiles
			fset := token.NewFileSet()
			_, parseErr := parser.ParseFile(fset, "spec.go", got, parser.AllErrors)
			if parseErr != nil {
				t.Errorf("Generated code does not compile: %v\nCode:\n%s", parseErr, code)
			}
		})
	}
}

func TestGenerateSpecFile_Compiles(t *testing.T) {
	// Test with a more complex schema to ensure generated code compiles
	schema := &SpecSchema{
		Properties: []PropertySchema{
			{Name: "name", Type: "string", Required: true, Description: "Engine name"},
			{Name: "version", Type: "string", Required: false, Description: "Version"},
			{Name: "count", Type: "integer", Required: false, Description: "Count"},
			{Name: "enabled", Type: "boolean", Required: false, Description: "Enabled flag"},
			{Name: "ratio", Type: "number", Required: false, Description: "A ratio"},
			{
				Name:        "args",
				Type:        "array",
				Description: "Arguments",
				Items:       &PropertySchema{Type: "string"},
			},
			{
				Name:                 "env",
				Type:                 "object",
				Description:          "Environment variables",
				AdditionalProperties: &PropertySchema{Type: "string"},
			},
		},
		Required: []string{"name"},
	}

	config := &Config{
		Name: "complex-engine",
		Type: EngineTypeBuilder,
		Generate: GenerateConfig{
			PackageName: "main",
		},
	}

	// Pass nil registry for backwards compatibility tests
	got, err := GenerateSpecFile(schema, config, "sha256:test", nil)
	if err != nil {
		t.Fatalf("GenerateSpecFile() error = %v", err)
	}

	// Verify the generated code compiles
	fset := token.NewFileSet()
	_, parseErr := parser.ParseFile(fset, "spec.go", got, parser.AllErrors)
	if parseErr != nil {
		t.Errorf("Generated code does not compile: %v\nCode:\n%s", parseErr, string(got))
	}
}

func TestHasArrayOrMapType(t *testing.T) {
	tests := []struct {
		name       string
		properties []PropertySchema
		want       bool
	}{
		{
			name:       "empty",
			properties: []PropertySchema{},
			want:       false,
		},
		{
			name: "only simple types",
			properties: []PropertySchema{
				{Name: "s", Type: "string"},
				{Name: "i", Type: "integer"},
				{Name: "b", Type: "boolean"},
			},
			want: false,
		},
		{
			name: "has array",
			properties: []PropertySchema{
				{Name: "s", Type: "string"},
				{Name: "arr", Type: "array", Items: &PropertySchema{Type: "string"}},
			},
			want: true,
		},
		{
			name: "has map",
			properties: []PropertySchema{
				{Name: "s", Type: "string"},
				{Name: "m", Type: "object", AdditionalProperties: &PropertySchema{Type: "string"}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasArrayOrMapType(tt.properties)
			if got != tt.want {
				t.Errorf("hasArrayOrMapType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGenerateSpecFileFromTypes tests the new ForgeTypeDefinition-based generation path.
func TestGenerateSpecFileFromTypes(t *testing.T) {
	tests := []struct {
		name       string
		types      []ForgeTypeDefinition
		config     *Config
		checksum   string
		wantFields []string
		wantFuncs  []string
		wantErr    bool
	}{
		{
			name: "basic string fields",
			types: []ForgeTypeDefinition{
				{
					Name:     "Spec",
					JsonName: "Spec",
					Properties: []ForgeProperty{
						{Name: "Name", JsonName: "name", GoType: "string", Required: true, Description: "The name field"},
						{Name: "Version", JsonName: "version", GoType: "string", Required: false, Description: "Optional version"},
					},
				},
			},
			config: &Config{
				Name: "test-engine",
				Type: EngineTypeBuilder,
				Generate: GenerateConfig{
					PackageName: "main",
				},
			},
			checksum:   "sha256:abc123",
			wantFields: []string{"Name string", "Version string"},
			wantFuncs:  []string{"func SpecFromMap(", "func (s *Spec) ToMap()"},
			wantErr:    false,
		},
		{
			name: "multiple types",
			types: []ForgeTypeDefinition{
				{
					Name:     "Spec",
					JsonName: "Spec",
					Properties: []ForgeProperty{
						{Name: "Count", JsonName: "count", GoType: "int", Required: false, Description: "Item count"},
						{Name: "Enabled", JsonName: "enabled", GoType: "bool", Required: false, Description: "Is enabled"},
						{Name: "Ratio", JsonName: "ratio", GoType: "float64", Required: false, Description: "A ratio"},
					},
				},
			},
			config: &Config{
				Name: "test-engine",
				Type: EngineTypeBuilder,
				Generate: GenerateConfig{
					PackageName: "main",
				},
			},
			checksum:   "sha256:def456",
			wantFields: []string{"Count int", "Enabled bool", "Ratio float64"},
			wantFuncs:  []string{"func SpecFromMap(", "func (s *Spec) ToMap()"},
			wantErr:    false,
		},
		{
			name: "array and map types",
			types: []ForgeTypeDefinition{
				{
					Name:     "Spec",
					JsonName: "Spec",
					Properties: []ForgeProperty{
						{Name: "Args", JsonName: "args", GoType: "[]string", IsArray: true, ArrayItemType: "string", Description: "Arguments"},
						{Name: "Env", JsonName: "env", GoType: "map[string]string", IsMap: true, MapValueType: "string", Description: "Environment variables"},
					},
				},
			},
			config: &Config{
				Name: "test-engine",
				Type: EngineTypeBuilder,
				Generate: GenerateConfig{
					PackageName: "main",
				},
			},
			checksum:   "sha256:ghi789",
			wantFields: []string{"Args []string", "Env map[string]string"},
			wantFuncs:  []string{"func SpecFromMap(", "func (s *Spec) ToMap()"},
			wantErr:    false,
		},
		{
			name: "reference types",
			types: []ForgeTypeDefinition{
				{
					Name:     "VMResource",
					JsonName: "VMResource",
					Properties: []ForgeProperty{
						{Name: "Name", JsonName: "name", GoType: "string", Required: true},
					},
				},
				{
					Name:     "Spec",
					JsonName: "Spec",
					Properties: []ForgeProperty{
						{Name: "VM", JsonName: "vm", GoType: "VMResource", IsRef: true, RefType: "VMResource"},
					},
				},
			},
			config: &Config{
				Name: "test-engine",
				Type: EngineTypeBuilder,
				Generate: GenerateConfig{
					PackageName: "main",
				},
			},
			checksum:   "sha256:ref123",
			wantFields: []string{"type VMResource struct", "type Spec struct", "VM VMResource"},
			wantFuncs:  []string{"func VMResourceFromMap(", "func SpecFromMap("},
			wantErr:    false,
		},
		{
			name: "array of references",
			types: []ForgeTypeDefinition{
				{
					Name:     "Item",
					JsonName: "Item",
					Properties: []ForgeProperty{
						{Name: "ID", JsonName: "id", GoType: "string", Required: true},
					},
				},
				{
					Name:     "Spec",
					JsonName: "Spec",
					Properties: []ForgeProperty{
						{Name: "Items", JsonName: "items", GoType: "[]Item", IsArray: true, IsArrayOfRef: true, ArrayItemType: "Item"},
					},
				},
			},
			config: &Config{
				Name: "test-engine",
				Type: EngineTypeBuilder,
				Generate: GenerateConfig{
					PackageName: "main",
				},
			},
			checksum:   "sha256:arrref123",
			wantFields: []string{"type Item struct", "type Spec struct", "Items []Item"},
			wantFuncs:  []string{"func ItemFromMap(", "func SpecFromMap("},
			wantErr:    false,
		},
		{
			name:  "empty types",
			types: []ForgeTypeDefinition{},
			config: &Config{
				Name: "test-engine",
				Type: EngineTypeBuilder,
				Generate: GenerateConfig{
					PackageName: "main",
				},
			},
			checksum:   "sha256:empty",
			wantFields: []string{},
			wantFuncs:  []string{},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateSpecFileFromTypes(tt.types, tt.config, tt.checksum)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateSpecFileFromTypes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			code := string(got)

			// Check that the code contains expected fields
			for _, field := range tt.wantFields {
				if !strings.Contains(code, field) {
					t.Errorf("Generated code missing field %q\nCode:\n%s", field, code)
				}
			}

			// Check that the code contains expected functions
			for _, fn := range tt.wantFuncs {
				if !strings.Contains(code, fn) {
					t.Errorf("Generated code missing function %q\nCode:\n%s", fn, code)
				}
			}

			// Check that the code contains the checksum
			if !strings.Contains(code, tt.checksum) {
				t.Errorf("Generated code missing checksum %q", tt.checksum)
			}

			// Verify the generated code compiles
			if len(code) > 0 {
				fset := token.NewFileSet()
				_, parseErr := parser.ParseFile(fset, "spec.go", got, parser.AllErrors)
				if parseErr != nil {
					t.Errorf("Generated code does not compile: %v\nCode:\n%s", parseErr, code)
				}
			}
		})
	}
}

// TestGenerateValidateFileFromTypes tests the new ForgeTypeDefinition-based validation generation path.
func TestGenerateValidateFileFromTypes(t *testing.T) {
	tests := []struct {
		name      string
		types     []ForgeTypeDefinition
		config    *Config
		checksum  string
		wantFuncs []string
		wantErr   bool
	}{
		{
			name: "basic validation",
			types: []ForgeTypeDefinition{
				{
					Name:     "Spec",
					JsonName: "Spec",
					Properties: []ForgeProperty{
						{Name: "Name", JsonName: "name", GoType: "string", Required: true},
						{Name: "Count", JsonName: "count", GoType: "int", Required: false},
					},
				},
			},
			config: &Config{
				Name: "test-engine",
				Type: EngineTypeBuilder,
				Generate: GenerateConfig{
					PackageName: "main",
				},
			},
			checksum:  "sha256:val123",
			wantFuncs: []string{"func ValidateSpec(", "func Validate(", "func ValidateMap("},
			wantErr:   false,
		},
		{
			name: "multiple types validation",
			types: []ForgeTypeDefinition{
				{
					Name:     "Item",
					JsonName: "Item",
					Properties: []ForgeProperty{
						{Name: "ID", JsonName: "id", GoType: "string", Required: true},
					},
				},
				{
					Name:     "Spec",
					JsonName: "Spec",
					Properties: []ForgeProperty{
						{Name: "Item", JsonName: "item", GoType: "Item", IsRef: true, RefType: "Item"},
					},
				},
			},
			config: &Config{
				Name: "test-engine",
				Type: EngineTypeBuilder,
				Generate: GenerateConfig{
					PackageName: "main",
				},
			},
			checksum:  "sha256:multival123",
			wantFuncs: []string{"func ValidateSpec(", "func ValidateItem("},
			wantErr:   false,
		},
		{
			name:  "empty types",
			types: []ForgeTypeDefinition{},
			config: &Config{
				Name: "test-engine",
				Type: EngineTypeBuilder,
				Generate: GenerateConfig{
					PackageName: "main",
				},
			},
			checksum:  "sha256:empty",
			wantFuncs: []string{},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateValidateFileFromTypes(tt.types, tt.config, tt.checksum)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateValidateFileFromTypes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			code := string(got)

			// Check that the code contains expected functions
			for _, fn := range tt.wantFuncs {
				if !strings.Contains(code, fn) {
					t.Errorf("Generated code missing function %q\nCode:\n%s", fn, code)
				}
			}

			// Check that the code contains the checksum
			if !strings.Contains(code, tt.checksum) {
				t.Errorf("Generated code missing checksum %q", tt.checksum)
			}

			// Verify the generated code compiles
			if len(code) > 0 {
				fset := token.NewFileSet()
				_, parseErr := parser.ParseFile(fset, "spec.go", got, parser.AllErrors)
				if parseErr != nil {
					t.Errorf("Generated code does not compile: %v\nCode:\n%s", parseErr, code)
				}
			}
		})
	}
}

// TestNeedsFmtImportForTypes tests the needsFmtImportForTypes helper function.
func TestNeedsFmtImportForTypes(t *testing.T) {
	tests := []struct {
		name  string
		types []ForgeTypeDefinition
		want  bool
	}{
		{
			name:  "empty types",
			types: []ForgeTypeDefinition{},
			want:  false,
		},
		{
			name: "simple types need fmt",
			types: []ForgeTypeDefinition{
				{
					Name: "Spec",
					Properties: []ForgeProperty{
						{Name: "Name", GoType: "string"},
					},
				},
			},
			want: true,
		},
		{
			name: "reference type needs fmt",
			types: []ForgeTypeDefinition{
				{
					Name: "Spec",
					Properties: []ForgeProperty{
						{Name: "VM", GoType: "VMResource", IsRef: true, RefType: "VMResource"},
					},
				},
			},
			want: true,
		},
		{
			name: "array of ref needs fmt",
			types: []ForgeTypeDefinition{
				{
					Name: "Spec",
					Properties: []ForgeProperty{
						{Name: "Items", GoType: "[]Item", IsArrayOfRef: true},
					},
				},
			},
			want: true,
		},
		{
			name: "enum type skipped",
			types: []ForgeTypeDefinition{
				{
					Name:       "Status",
					IsEnum:     true,
					EnumValues: []string{"active", "inactive"},
				},
			},
			want: false,
		},
		{
			name: "union type skipped",
			types: []ForgeTypeDefinition{
				{
					Name:          "MyUnion",
					IsUnion:       true,
					UnionVariants: []string{"TypeA", "TypeB"},
				},
			},
			want: false,
		},
		{
			name: "array needs fmt",
			types: []ForgeTypeDefinition{
				{
					Name: "Spec",
					Properties: []ForgeProperty{
						{Name: "Tags", GoType: "[]string", IsArray: true},
					},
				},
			},
			want: true,
		},
		{
			name: "map needs fmt",
			types: []ForgeTypeDefinition{
				{
					Name: "Spec",
					Properties: []ForgeProperty{
						{Name: "Labels", GoType: "map[string]string", IsMap: true},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsFmtImportForTypes(tt.types)
			if got != tt.want {
				t.Errorf("needsFmtImportForTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNeedsFmtImportForValidation tests the needsFmtImportForValidation helper function.
func TestNeedsFmtImportForValidation(t *testing.T) {
	tests := []struct {
		name  string
		types []ForgeTypeDefinition
		want  bool
	}{
		{
			name:  "empty types",
			types: []ForgeTypeDefinition{},
			want:  false,
		},
		{
			name: "simple types do not need fmt for validation",
			types: []ForgeTypeDefinition{
				{
					Name: "Spec",
					Properties: []ForgeProperty{
						{Name: "Name", GoType: "string"},
					},
				},
			},
			want: false,
		},
		{
			name: "array of ref needs fmt for validation",
			types: []ForgeTypeDefinition{
				{
					Name: "Spec",
					Properties: []ForgeProperty{
						{Name: "Items", GoType: "[]Item", IsArrayOfRef: true},
					},
				},
			},
			want: true,
		},
		{
			name: "enum type skipped",
			types: []ForgeTypeDefinition{
				{
					Name:       "Status",
					IsEnum:     true,
					EnumValues: []string{"active", "inactive"},
				},
			},
			want: false,
		},
		{
			name: "union type skipped",
			types: []ForgeTypeDefinition{
				{
					Name:          "MyUnion",
					IsUnion:       true,
					UnionVariants: []string{"TypeA", "TypeB"},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsFmtImportForValidation(tt.types)
			if got != tt.want {
				t.Errorf("needsFmtImportForValidation() = %v, want %v", got, tt.want)
			}
		})
	}
}
