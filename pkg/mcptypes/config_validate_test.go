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

package mcptypes

import (
	"encoding/json"
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

// -----------------------------------------------------------------------------
// JSON Round-Trip Tests for Types
// -----------------------------------------------------------------------------

// TestConfigValidateInput_JSONRoundTrip tests JSON marshal/unmarshal for ConfigValidateInput
func TestConfigValidateInput_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input ConfigValidateInput
	}{
		{
			name:  "Empty struct",
			input: ConfigValidateInput{},
		},
		{
			name: "Spec only",
			input: ConfigValidateInput{
				Spec: map[string]interface{}{
					"command": "go test",
					"args":    []interface{}{"-v", "-race"},
				},
			},
		},
		{
			name: "All fields populated",
			input: ConfigValidateInput{
				Spec: map[string]interface{}{
					"command": "go test",
				},
				ForgeSpec: &forge.Spec{
					Name: "test-project",
				},
				ConfigPath: "/path/to/forge.yaml",
				DirectoryParams: &DirectoryParams{
					TmpDir:   "/tmp/test",
					BuildDir: "/build",
					RootDir:  "/root",
				},
				SpecType: "build",
				SpecName: "my-app",
			},
		},
		{
			name: "Nil ForgeSpec and DirectoryParams",
			input: ConfigValidateInput{
				Spec:       map[string]interface{}{"key": "value"},
				ConfigPath: "/path/to/config",
				SpecType:   "test",
				SpecName:   "unit",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Test unmarshaling
			var unmarshaled ConfigValidateInput
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify key fields
			if unmarshaled.ConfigPath != tt.input.ConfigPath {
				t.Errorf("ConfigPath mismatch: got %s, want %s", unmarshaled.ConfigPath, tt.input.ConfigPath)
			}
			if unmarshaled.SpecType != tt.input.SpecType {
				t.Errorf("SpecType mismatch: got %s, want %s", unmarshaled.SpecType, tt.input.SpecType)
			}
			if unmarshaled.SpecName != tt.input.SpecName {
				t.Errorf("SpecName mismatch: got %s, want %s", unmarshaled.SpecName, tt.input.SpecName)
			}

			// Verify Spec
			if tt.input.Spec != nil && unmarshaled.Spec == nil {
				t.Error("Spec was nil after unmarshal, expected non-nil")
			}

			// Verify ForgeSpec
			if tt.input.ForgeSpec != nil {
				if unmarshaled.ForgeSpec == nil {
					t.Error("ForgeSpec was nil after unmarshal, expected non-nil")
				} else if unmarshaled.ForgeSpec.Name != tt.input.ForgeSpec.Name {
					t.Errorf("ForgeSpec.Name mismatch: got %s, want %s", unmarshaled.ForgeSpec.Name, tt.input.ForgeSpec.Name)
				}
			}

			// Verify DirectoryParams
			if tt.input.DirectoryParams != nil {
				if unmarshaled.DirectoryParams == nil {
					t.Error("DirectoryParams was nil after unmarshal, expected non-nil")
				} else if unmarshaled.DirectoryParams.TmpDir != tt.input.DirectoryParams.TmpDir {
					t.Errorf("DirectoryParams.TmpDir mismatch: got %s, want %s",
						unmarshaled.DirectoryParams.TmpDir, tt.input.DirectoryParams.TmpDir)
				}
			}
		})
	}
}

// TestConfigValidateOutput_JSONRoundTrip tests JSON marshal/unmarshal for ConfigValidateOutput
func TestConfigValidateOutput_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		output ConfigValidateOutput
	}{
		{
			name: "Valid output",
			output: ConfigValidateOutput{
				Valid: true,
			},
		},
		{
			name: "Invalid with errors",
			output: ConfigValidateOutput{
				Valid: false,
				Errors: []ValidationError{
					{
						Field:    "spec.command",
						Message:  "required field is missing",
						Engine:   "go://generic-builder",
						SpecType: "build",
						SpecName: "my-app",
					},
				},
			},
		},
		{
			name: "Valid with warnings",
			output: ConfigValidateOutput{
				Valid: true,
				Warnings: []ValidationWarning{
					{
						Field:   "spec.timeout",
						Message: "default timeout of 10m will be used",
					},
				},
			},
		},
		{
			name: "Invalid with infra error",
			output: ConfigValidateOutput{
				Valid:      false,
				InfraError: "engine process failed to spawn: exit code 1",
			},
		},
		{
			name: "Multiple errors and warnings",
			output: ConfigValidateOutput{
				Valid: false,
				Errors: []ValidationError{
					{Field: "spec.args", Message: "expected []string, got int"},
					{Field: "spec.env", Message: "expected map[string]string, got []interface{}"},
				},
				Warnings: []ValidationWarning{
					{Message: "deprecated field 'oldField' is being used"},
				},
			},
		},
		{
			name: "Empty errors and warnings slices",
			output: ConfigValidateOutput{
				Valid:    true,
				Errors:   []ValidationError{},
				Warnings: []ValidationWarning{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Test unmarshaling
			var unmarshaled ConfigValidateOutput
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify fields
			if unmarshaled.Valid != tt.output.Valid {
				t.Errorf("Valid mismatch: got %v, want %v", unmarshaled.Valid, tt.output.Valid)
			}
			if unmarshaled.InfraError != tt.output.InfraError {
				t.Errorf("InfraError mismatch: got %s, want %s", unmarshaled.InfraError, tt.output.InfraError)
			}
			if len(unmarshaled.Errors) != len(tt.output.Errors) {
				t.Errorf("Errors count mismatch: got %d, want %d", len(unmarshaled.Errors), len(tt.output.Errors))
			}
			if len(unmarshaled.Warnings) != len(tt.output.Warnings) {
				t.Errorf("Warnings count mismatch: got %d, want %d", len(unmarshaled.Warnings), len(tt.output.Warnings))
			}

			// Verify first error if exists
			if len(tt.output.Errors) > 0 && len(unmarshaled.Errors) > 0 {
				if unmarshaled.Errors[0].Field != tt.output.Errors[0].Field {
					t.Errorf("First error Field mismatch: got %s, want %s",
						unmarshaled.Errors[0].Field, tt.output.Errors[0].Field)
				}
				if unmarshaled.Errors[0].Message != tt.output.Errors[0].Message {
					t.Errorf("First error Message mismatch: got %s, want %s",
						unmarshaled.Errors[0].Message, tt.output.Errors[0].Message)
				}
			}
		})
	}
}

// TestValidationError_JSONRoundTrip tests JSON marshal/unmarshal for ValidationError
func TestValidationError_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		error ValidationError
	}{
		{
			name: "All fields populated",
			error: ValidationError{
				Field:    "spec.args[0]",
				Message:  "expected string, got int",
				Engine:   "go://go-build",
				SpecType: "build",
				SpecName: "my-app",
			},
		},
		{
			name: "Required fields only",
			error: ValidationError{
				Field:   "spec.command",
				Message: "required field is missing",
			},
		},
		{
			name: "Empty field path",
			error: ValidationError{
				Field:   "",
				Message: "infrastructure error: engine failed",
				Engine:  "go://testenv",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.error)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Test unmarshaling
			var unmarshaled ValidationError
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify all fields
			if unmarshaled.Field != tt.error.Field {
				t.Errorf("Field mismatch: got %s, want %s", unmarshaled.Field, tt.error.Field)
			}
			if unmarshaled.Message != tt.error.Message {
				t.Errorf("Message mismatch: got %s, want %s", unmarshaled.Message, tt.error.Message)
			}
			if unmarshaled.Engine != tt.error.Engine {
				t.Errorf("Engine mismatch: got %s, want %s", unmarshaled.Engine, tt.error.Engine)
			}
			if unmarshaled.SpecType != tt.error.SpecType {
				t.Errorf("SpecType mismatch: got %s, want %s", unmarshaled.SpecType, tt.error.SpecType)
			}
			if unmarshaled.SpecName != tt.error.SpecName {
				t.Errorf("SpecName mismatch: got %s, want %s", unmarshaled.SpecName, tt.error.SpecName)
			}
		})
	}
}

// TestValidationWarning_JSONRoundTrip tests JSON marshal/unmarshal for ValidationWarning
func TestValidationWarning_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		warning ValidationWarning
	}{
		{
			name: "With field",
			warning: ValidationWarning{
				Field:   "spec.timeout",
				Message: "using default timeout of 10m",
			},
		},
		{
			name: "Without field",
			warning: ValidationWarning{
				Message: "deprecated configuration format detected",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.warning)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Test unmarshaling
			var unmarshaled ValidationWarning
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify fields
			if unmarshaled.Field != tt.warning.Field {
				t.Errorf("Field mismatch: got %s, want %s", unmarshaled.Field, tt.warning.Field)
			}
			if unmarshaled.Message != tt.warning.Message {
				t.Errorf("Message mismatch: got %s, want %s", unmarshaled.Message, tt.warning.Message)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Validation Helper Tests
// -----------------------------------------------------------------------------

// TestValidateString_ValidString tests that ValidateString returns value and nil for valid strings
func TestValidateString_ValidString(t *testing.T) {
	tests := []struct {
		name     string
		spec     map[string]interface{}
		field    string
		expected string
	}{
		{
			name:     "Simple string",
			spec:     map[string]interface{}{"command": "go test"},
			field:    "command",
			expected: "go test",
		},
		{
			name:     "Empty string is valid",
			spec:     map[string]interface{}{"path": ""},
			field:    "path",
			expected: "",
		},
		{
			name:     "String with special characters",
			spec:     map[string]interface{}{"pattern": "*.go"},
			field:    "pattern",
			expected: "*.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateString(tt.spec, tt.field)
			if err != nil {
				t.Errorf("Expected no error, got: %+v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestValidateString_Missing tests that ValidateString returns "", nil for missing fields (optional)
func TestValidateString_Missing(t *testing.T) {
	spec := map[string]interface{}{"other": "value"}
	result, err := ValidateString(spec, "missing")
	if err != nil {
		t.Errorf("Expected no error for missing optional field, got: %+v", err)
	}
	if result != "" {
		t.Errorf("Expected empty string for missing field, got: %q", result)
	}
}

// TestValidateString_WrongType tests that ValidateString returns "", ValidationError for wrong types
func TestValidateString_WrongType(t *testing.T) {
	tests := []struct {
		name  string
		spec  map[string]interface{}
		field string
	}{
		{
			name:  "Integer instead of string",
			spec:  map[string]interface{}{"port": 8080},
			field: "port",
		},
		{
			name:  "Boolean instead of string",
			spec:  map[string]interface{}{"enabled": true},
			field: "enabled",
		},
		{
			name:  "Array instead of string",
			spec:  map[string]interface{}{"args": []interface{}{"a", "b"}},
			field: "args",
		},
		{
			name:  "Map instead of string",
			spec:  map[string]interface{}{"env": map[string]interface{}{"KEY": "value"}},
			field: "env",
		},
		{
			name:  "Float instead of string",
			spec:  map[string]interface{}{"version": 1.5},
			field: "version",
		},
		{
			name:  "Nil value",
			spec:  map[string]interface{}{"nullable": nil},
			field: "nullable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateString(tt.spec, tt.field)
			if err == nil {
				t.Errorf("Expected ValidationError, got nil")
			}
			if result != "" {
				t.Errorf("Expected empty string on error, got: %q", result)
			}
			if err != nil && err.Field != "spec."+tt.field {
				t.Errorf("Expected field 'spec.%s', got: %s", tt.field, err.Field)
			}
		})
	}
}

// TestValidateStringRequired_ValidString tests that ValidateStringRequired returns value for valid strings
func TestValidateStringRequired_ValidString(t *testing.T) {
	tests := []struct {
		name     string
		spec     map[string]interface{}
		field    string
		expected string
	}{
		{
			name:     "Simple string",
			spec:     map[string]interface{}{"command": "go test"},
			field:    "command",
			expected: "go test",
		},
		{
			name:     "String with spaces",
			spec:     map[string]interface{}{"name": "my app"},
			field:    "name",
			expected: "my app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateStringRequired(tt.spec, tt.field)
			if err != nil {
				t.Errorf("Expected no error, got: %+v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestValidateStringRequired_Missing tests that ValidateStringRequired returns error for missing fields
func TestValidateStringRequired_Missing(t *testing.T) {
	spec := map[string]interface{}{"other": "value"}
	result, err := ValidateStringRequired(spec, "command")

	if err == nil {
		t.Error("Expected ValidationError for missing required field, got nil")
	}
	if result != "" {
		t.Errorf("Expected empty string on error, got: %q", result)
	}
	if err != nil && err.Field != "spec.command" {
		t.Errorf("Expected field 'spec.command', got: %s", err.Field)
	}
	if err != nil && err.Message != "required field is missing" {
		t.Errorf("Expected message about missing field, got: %s", err.Message)
	}
}

// TestValidateStringRequired_EmptyString tests that ValidateStringRequired returns error for empty strings
func TestValidateStringRequired_EmptyString(t *testing.T) {
	spec := map[string]interface{}{"command": ""}
	result, err := ValidateStringRequired(spec, "command")

	if err == nil {
		t.Error("Expected ValidationError for empty required field, got nil")
	}
	if result != "" {
		t.Errorf("Expected empty string on error, got: %q", result)
	}
	if err != nil && err.Message != "required field cannot be empty" {
		t.Errorf("Expected message about empty field, got: %s", err.Message)
	}
}

// TestValidateStringRequired_WrongType tests that ValidateStringRequired returns error for wrong types
func TestValidateStringRequired_WrongType(t *testing.T) {
	spec := map[string]interface{}{"command": 123}
	result, err := ValidateStringRequired(spec, "command")

	if err == nil {
		t.Error("Expected ValidationError for wrong type, got nil")
	}
	if result != "" {
		t.Errorf("Expected empty string on error, got: %q", result)
	}
}

// TestValidateStringSlice_ValidSlice tests that ValidateStringSlice returns slice for valid arrays
func TestValidateStringSlice_ValidSlice(t *testing.T) {
	tests := []struct {
		name     string
		spec     map[string]interface{}
		field    string
		expected []string
	}{
		{
			name:     "JSON-style array ([]interface{})",
			spec:     map[string]interface{}{"args": []interface{}{"-v", "-race"}},
			field:    "args",
			expected: []string{"-v", "-race"},
		},
		{
			name:     "Go-style array ([]string)",
			spec:     map[string]interface{}{"tags": []string{"unit", "integration"}},
			field:    "tags",
			expected: []string{"unit", "integration"},
		},
		{
			name:     "Empty array",
			spec:     map[string]interface{}{"args": []interface{}{}},
			field:    "args",
			expected: []string{},
		},
		{
			name:     "Single element",
			spec:     map[string]interface{}{"packages": []interface{}{"./..."}},
			field:    "packages",
			expected: []string{"./..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateStringSlice(tt.spec, tt.field)
			if err != nil {
				t.Errorf("Expected no error, got: %+v", err)
			}
			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
			}
			for i, v := range tt.expected {
				if i < len(result) && result[i] != v {
					t.Errorf("Element %d: expected %q, got %q", i, v, result[i])
				}
			}
		})
	}
}

// TestValidateStringSlice_Missing tests that ValidateStringSlice returns nil, nil for missing fields
func TestValidateStringSlice_Missing(t *testing.T) {
	spec := map[string]interface{}{"other": "value"}
	result, err := ValidateStringSlice(spec, "args")
	if err != nil {
		t.Errorf("Expected no error for missing optional field, got: %+v", err)
	}
	if result != nil {
		t.Errorf("Expected nil for missing field, got: %v", result)
	}
}

// TestValidateStringSlice_WrongType tests that ValidateStringSlice returns error for wrong types
func TestValidateStringSlice_WrongType(t *testing.T) {
	tests := []struct {
		name  string
		spec  map[string]interface{}
		field string
	}{
		{
			name:  "String instead of array",
			spec:  map[string]interface{}{"args": "single-value"},
			field: "args",
		},
		{
			name:  "Integer instead of array",
			spec:  map[string]interface{}{"args": 123},
			field: "args",
		},
		{
			name:  "Map instead of array",
			spec:  map[string]interface{}{"args": map[string]interface{}{"key": "value"}},
			field: "args",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateStringSlice(tt.spec, tt.field)
			if err == nil {
				t.Error("Expected ValidationError, got nil")
			}
			if result != nil {
				t.Errorf("Expected nil on error, got: %v", result)
			}
			if err != nil && err.Field != "spec."+tt.field {
				t.Errorf("Expected field 'spec.%s', got: %s", tt.field, err.Field)
			}
		})
	}
}

// TestValidateStringSlice_ElementWrongType tests that ValidateStringSlice returns error for non-string elements
func TestValidateStringSlice_ElementWrongType(t *testing.T) {
	tests := []struct {
		name          string
		spec          map[string]interface{}
		field         string
		expectedField string
	}{
		{
			name:          "Integer element at index 0",
			spec:          map[string]interface{}{"args": []interface{}{123, "valid"}},
			field:         "args",
			expectedField: "spec.args[0]",
		},
		{
			name:          "Integer element at index 1",
			spec:          map[string]interface{}{"args": []interface{}{"valid", 456}},
			field:         "args",
			expectedField: "spec.args[1]",
		},
		{
			name:          "Boolean element",
			spec:          map[string]interface{}{"tags": []interface{}{"unit", true}},
			field:         "tags",
			expectedField: "spec.tags[1]",
		},
		{
			name:          "Map element",
			spec:          map[string]interface{}{"packages": []interface{}{map[string]interface{}{"key": "value"}}},
			field:         "packages",
			expectedField: "spec.packages[0]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateStringSlice(tt.spec, tt.field)
			if err == nil {
				t.Error("Expected ValidationError, got nil")
			}
			if result != nil {
				t.Errorf("Expected nil on error, got: %v", result)
			}
			if err != nil && err.Field != tt.expectedField {
				t.Errorf("Expected field %q, got: %s", tt.expectedField, err.Field)
			}
		})
	}
}

// TestValidateStringMap_ValidMap tests that ValidateStringMap returns map for valid maps
func TestValidateStringMap_ValidMap(t *testing.T) {
	tests := []struct {
		name     string
		spec     map[string]interface{}
		field    string
		expected map[string]string
	}{
		{
			name: "JSON-style map (map[string]interface{})",
			spec: map[string]interface{}{
				"env": map[string]interface{}{
					"GO_ENV":      "test",
					"GOPROXY":     "direct",
					"GOOS":        "linux",
					"GOARCH":      "amd64",
					"CGO_ENABLED": "0",
				},
			},
			field: "env",
			expected: map[string]string{
				"GO_ENV":      "test",
				"GOPROXY":     "direct",
				"GOOS":        "linux",
				"GOARCH":      "amd64",
				"CGO_ENABLED": "0",
			},
		},
		{
			name: "Go-style map (map[string]string)",
			spec: map[string]interface{}{
				"env": map[string]string{
					"KEY1": "value1",
					"KEY2": "value2",
				},
			},
			field: "env",
			expected: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
		},
		{
			name:     "Empty map",
			spec:     map[string]interface{}{"env": map[string]interface{}{}},
			field:    "env",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateStringMap(tt.spec, tt.field)
			if err != nil {
				t.Errorf("Expected no error, got: %+v", err)
			}
			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("Key %q: expected %q, got %q", k, v, result[k])
				}
			}
		})
	}
}

// TestValidateStringMap_Missing tests that ValidateStringMap returns nil, nil for missing fields
func TestValidateStringMap_Missing(t *testing.T) {
	spec := map[string]interface{}{"other": "value"}
	result, err := ValidateStringMap(spec, "env")
	if err != nil {
		t.Errorf("Expected no error for missing optional field, got: %+v", err)
	}
	if result != nil {
		t.Errorf("Expected nil for missing field, got: %v", result)
	}
}

// TestValidateStringMap_WrongType tests that ValidateStringMap returns error for wrong types
func TestValidateStringMap_WrongType(t *testing.T) {
	tests := []struct {
		name  string
		spec  map[string]interface{}
		field string
	}{
		{
			name:  "String instead of map",
			spec:  map[string]interface{}{"env": "KEY=value"},
			field: "env",
		},
		{
			name:  "Integer instead of map",
			spec:  map[string]interface{}{"env": 123},
			field: "env",
		},
		{
			name:  "Array instead of map",
			spec:  map[string]interface{}{"env": []interface{}{"KEY=value"}},
			field: "env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateStringMap(tt.spec, tt.field)
			if err == nil {
				t.Error("Expected ValidationError, got nil")
			}
			if result != nil {
				t.Errorf("Expected nil on error, got: %v", result)
			}
			if err != nil && err.Field != "spec."+tt.field {
				t.Errorf("Expected field 'spec.%s', got: %s", tt.field, err.Field)
			}
		})
	}
}

// TestValidateStringMap_ValueWrongType tests that ValidateStringMap returns error for non-string values
func TestValidateStringMap_ValueWrongType(t *testing.T) {
	tests := []struct {
		name          string
		spec          map[string]interface{}
		field         string
		expectedField string
	}{
		{
			name: "Integer value",
			spec: map[string]interface{}{
				"env": map[string]interface{}{"PORT": 8080},
			},
			field:         "env",
			expectedField: "spec.env.PORT",
		},
		{
			name: "Boolean value",
			spec: map[string]interface{}{
				"env": map[string]interface{}{"ENABLED": true},
			},
			field:         "env",
			expectedField: "spec.env.ENABLED",
		},
		{
			name: "Array value",
			spec: map[string]interface{}{
				"env": map[string]interface{}{"TAGS": []interface{}{"a", "b"}},
			},
			field:         "env",
			expectedField: "spec.env.TAGS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateStringMap(tt.spec, tt.field)
			if err == nil {
				t.Error("Expected ValidationError, got nil")
			}
			if result != nil {
				t.Errorf("Expected nil on error, got: %v", result)
			}
			if err != nil && err.Field != tt.expectedField {
				t.Errorf("Expected field %q, got: %s", tt.expectedField, err.Field)
			}
		})
	}
}

// TestValidateBool_Valid tests that ValidateBool returns bool for valid booleans
func TestValidateBool_Valid(t *testing.T) {
	tests := []struct {
		name     string
		spec     map[string]interface{}
		field    string
		expected bool
	}{
		{
			name:     "True value",
			spec:     map[string]interface{}{"enabled": true},
			field:    "enabled",
			expected: true,
		},
		{
			name:     "False value",
			spec:     map[string]interface{}{"disabled": false},
			field:    "disabled",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateBool(tt.spec, tt.field)
			if err != nil {
				t.Errorf("Expected no error, got: %+v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestValidateBool_Missing tests that ValidateBool returns false, nil for missing fields
func TestValidateBool_Missing(t *testing.T) {
	spec := map[string]interface{}{"other": "value"}
	result, err := ValidateBool(spec, "enabled")
	if err != nil {
		t.Errorf("Expected no error for missing optional field, got: %+v", err)
	}
	if result != false {
		t.Errorf("Expected false for missing field, got: %v", result)
	}
}

// TestValidateBool_WrongType tests that ValidateBool returns error for wrong types
func TestValidateBool_WrongType(t *testing.T) {
	tests := []struct {
		name  string
		spec  map[string]interface{}
		field string
	}{
		{
			name:  "String 'true' instead of bool",
			spec:  map[string]interface{}{"enabled": "true"},
			field: "enabled",
		},
		{
			name:  "Integer 1 instead of bool",
			spec:  map[string]interface{}{"enabled": 1},
			field: "enabled",
		},
		{
			name:  "Integer 0 instead of bool",
			spec:  map[string]interface{}{"enabled": 0},
			field: "enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateBool(tt.spec, tt.field)
			if err == nil {
				t.Error("Expected ValidationError, got nil")
			}
			if result != false {
				t.Errorf("Expected false on error, got: %v", result)
			}
			if err != nil && err.Field != "spec."+tt.field {
				t.Errorf("Expected field 'spec.%s', got: %s", tt.field, err.Field)
			}
		})
	}
}

// TestValidateInt_ValidInt tests that ValidateInt returns int for valid integers
func TestValidateInt_ValidInt(t *testing.T) {
	tests := []struct {
		name     string
		spec     map[string]interface{}
		field    string
		expected int
	}{
		{
			name:     "Integer value (Go int)",
			spec:     map[string]interface{}{"port": 8080},
			field:    "port",
			expected: 8080,
		},
		{
			name:     "Float64 value (JSON number)",
			spec:     map[string]interface{}{"count": float64(42)},
			field:    "count",
			expected: 42,
		},
		{
			name:     "Zero value",
			spec:     map[string]interface{}{"offset": 0},
			field:    "offset",
			expected: 0,
		},
		{
			name:     "Negative value",
			spec:     map[string]interface{}{"delta": -10},
			field:    "delta",
			expected: -10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateInt(tt.spec, tt.field)
			if err != nil {
				t.Errorf("Expected no error, got: %+v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestValidateInt_Missing tests that ValidateInt returns 0, nil for missing fields
func TestValidateInt_Missing(t *testing.T) {
	spec := map[string]interface{}{"other": "value"}
	result, err := ValidateInt(spec, "port")
	if err != nil {
		t.Errorf("Expected no error for missing optional field, got: %+v", err)
	}
	if result != 0 {
		t.Errorf("Expected 0 for missing field, got: %d", result)
	}
}

// TestValidateInt_WrongType tests that ValidateInt returns error for wrong types
func TestValidateInt_WrongType(t *testing.T) {
	tests := []struct {
		name  string
		spec  map[string]interface{}
		field string
	}{
		{
			name:  "String number instead of int",
			spec:  map[string]interface{}{"port": "8080"},
			field: "port",
		},
		{
			name:  "Boolean instead of int",
			spec:  map[string]interface{}{"count": true},
			field: "count",
		},
		{
			name:  "Array instead of int",
			spec:  map[string]interface{}{"port": []interface{}{8080}},
			field: "port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateInt(tt.spec, tt.field)
			if err == nil {
				t.Error("Expected ValidationError, got nil")
			}
			if result != 0 {
				t.Errorf("Expected 0 on error, got: %d", result)
			}
			if err != nil && err.Field != "spec."+tt.field {
				t.Errorf("Expected field 'spec.%s', got: %s", tt.field, err.Field)
			}
		})
	}
}

// TestValidateInt_FromFloat64 tests that ValidateInt correctly converts JSON float64 to int
func TestValidateInt_FromFloat64(t *testing.T) {
	// Simulating JSON unmarshal behavior where numbers become float64
	jsonData := []byte(`{"port": 8080}`)
	var spec map[string]interface{}
	if err := json.Unmarshal(jsonData, &spec); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	result, err := ValidateInt(spec, "port")
	if err != nil {
		t.Errorf("Expected no error, got: %+v", err)
	}
	if result != 8080 {
		t.Errorf("Expected 8080, got %d", result)
	}
}
