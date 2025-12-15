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
