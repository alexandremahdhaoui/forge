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

func TestGenerateValidateFile(t *testing.T) {
	tests := []struct {
		name         string
		schema       *SpecSchema
		config       *Config
		checksum     string
		wantFuncs    []string
		wantContains []string
		wantErr      bool
	}{
		{
			name: "basic required field validation",
			schema: &SpecSchema{
				Properties: []PropertySchema{
					{Name: "name", Type: "string", Required: true, Description: "Required name"},
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
			checksum:     "sha256:abc123",
			wantFuncs:    []string{"func Validate(", "func ValidateMap("},
			wantContains: []string{"required field is missing"},
			wantErr:      false,
		},
		{
			name: "enum validation",
			schema: &SpecSchema{
				Properties: []PropertySchema{
					{
						Name:        "mode",
						Type:        "string",
						Required:    false,
						Description: "Operation mode",
						Enum:        []string{"fast", "slow", "balanced"},
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
			checksum:     "sha256:def456",
			wantFuncs:    []string{"func Validate(", "func ValidateMap("},
			wantContains: []string{"must be one of:", "fast", "slow", "balanced"},
			wantErr:      false,
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
			checksum:     "sha256:empty",
			wantFuncs:    []string{"func Validate(", "func ValidateMap("},
			wantContains: []string{},
			wantErr:      false,
		},
		{
			name: "required array field",
			schema: &SpecSchema{
				Properties: []PropertySchema{
					{
						Name:        "args",
						Type:        "array",
						Required:    true,
						Description: "Required arguments",
						Items:       &PropertySchema{Type: "string"},
					},
				},
				Required: []string{"args"},
			},
			config: &Config{
				Name: "test-engine",
				Type: EngineTypeBuilder,
				Generate: GenerateConfig{
					PackageName: "main",
				},
			},
			checksum:     "sha256:arr123",
			wantFuncs:    []string{"func Validate(", "func ValidateMap("},
			wantContains: []string{"required field is missing or empty"},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateValidateFile(tt.schema, tt.config, tt.checksum)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateValidateFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			code := string(got)

			// Check that the code contains expected functions
			for _, fn := range tt.wantFuncs {
				if !strings.Contains(code, fn) {
					t.Errorf("Generated code missing function %q", fn)
				}
			}

			// Check that the code contains expected strings
			for _, s := range tt.wantContains {
				if !strings.Contains(code, s) {
					t.Errorf("Generated code missing %q", s)
				}
			}

			// Check that the code contains the checksum
			if !strings.Contains(code, tt.checksum) {
				t.Errorf("Generated code missing checksum %q", tt.checksum)
			}

			// Verify the generated code compiles
			fset := token.NewFileSet()
			_, parseErr := parser.ParseFile(fset, "validate.go", got, parser.AllErrors)
			if parseErr != nil {
				t.Errorf("Generated code does not compile: %v\nCode:\n%s", parseErr, code)
			}
		})
	}
}

func TestGenerateValidateFile_Compiles(t *testing.T) {
	// Test with a more complex schema to ensure generated code compiles
	schema := &SpecSchema{
		Properties: []PropertySchema{
			{Name: "name", Type: "string", Required: true, Description: "Required name"},
			{Name: "version", Type: "string", Required: false, Description: "Optional version"},
			{
				Name:        "mode",
				Type:        "string",
				Required:    false,
				Description: "Operation mode",
				Enum:        []string{"fast", "slow", "balanced"},
			},
			{
				Name:        "args",
				Type:        "array",
				Required:    true,
				Description: "Required arguments",
				Items:       &PropertySchema{Type: "string"},
			},
			{
				Name:                 "env",
				Type:                 "object",
				Required:             true,
				Description:          "Required environment",
				AdditionalProperties: &PropertySchema{Type: "string"},
			},
		},
		Required: []string{"name", "args", "env"},
	}

	config := &Config{
		Name: "complex-engine",
		Type: EngineTypeBuilder,
		Generate: GenerateConfig{
			PackageName: "main",
		},
	}

	got, err := GenerateValidateFile(schema, config, "sha256:test")
	if err != nil {
		t.Fatalf("GenerateValidateFile() error = %v", err)
	}

	// Verify the generated code compiles
	fset := token.NewFileSet()
	_, parseErr := parser.ParseFile(fset, "validate.go", got, parser.AllErrors)
	if parseErr != nil {
		t.Errorf("Generated code does not compile: %v\nCode:\n%s", parseErr, string(got))
	}
}
