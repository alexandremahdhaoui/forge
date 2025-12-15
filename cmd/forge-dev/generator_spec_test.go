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
			got, err := GenerateSpecFile(tt.schema, tt.config, tt.checksum)
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

	got, err := GenerateSpecFile(schema, config, "sha256:test")
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
