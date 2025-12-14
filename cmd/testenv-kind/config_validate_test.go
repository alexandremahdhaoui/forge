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

func TestConfigValidate_ValidSpecAllFields(t *testing.T) {
	// Valid spec with all fields populated
	spec := map[string]interface{}{
		"name":        "my-cluster",
		"image":       "kindest/node:v1.30.0",
		"config":      "/path/to/kind-config.yaml",
		"waitTimeout": "5m",
		"retain":      true,
	}

	output := validateKindSpec(spec)

	if !output.Valid {
		t.Errorf("validateKindSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateKindSpec() errors = %v, want none", output.Errors)
	}
}

func TestConfigValidate_EmptySpec(t *testing.T) {
	// Empty spec should be valid (all fields are optional)
	tests := []struct {
		name string
		spec map[string]interface{}
	}{
		{
			name: "nil spec",
			spec: nil,
		},
		{
			name: "empty spec",
			spec: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := validateKindSpec(tt.spec)

			if !output.Valid {
				t.Errorf("validateKindSpec() valid = %v, want true", output.Valid)
			}
			if len(output.Errors) != 0 {
				t.Errorf("validateKindSpec() errors = %v, want none", output.Errors)
			}
		})
	}
}

func TestConfigValidate_InvalidNameType(t *testing.T) {
	// name is not a string (it's an integer)
	spec := map[string]interface{}{
		"name": 123,
	}

	output := validateKindSpec(spec)

	if output.Valid {
		t.Errorf("validateKindSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateKindSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.name" {
		t.Errorf("validateKindSpec() error field = %q, want %q", output.Errors[0].Field, "spec.name")
	}
}

func TestConfigValidate_InvalidImageType(t *testing.T) {
	// image is not a string (it's a map)
	spec := map[string]interface{}{
		"image": map[string]interface{}{"name": "kindest/node", "tag": "v1.30.0"},
	}

	output := validateKindSpec(spec)

	if output.Valid {
		t.Errorf("validateKindSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateKindSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.image" {
		t.Errorf("validateKindSpec() error field = %q, want %q", output.Errors[0].Field, "spec.image")
	}
}

func TestConfigValidate_InvalidConfigType(t *testing.T) {
	// config is not a string (it's an array)
	spec := map[string]interface{}{
		"config": []interface{}{"/path/to/config1.yaml", "/path/to/config2.yaml"},
	}

	output := validateKindSpec(spec)

	if output.Valid {
		t.Errorf("validateKindSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateKindSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.config" {
		t.Errorf("validateKindSpec() error field = %q, want %q", output.Errors[0].Field, "spec.config")
	}
}

func TestConfigValidate_InvalidWaitTimeoutType(t *testing.T) {
	// waitTimeout is not a string (it's a number)
	spec := map[string]interface{}{
		"waitTimeout": 300,
	}

	output := validateKindSpec(spec)

	if output.Valid {
		t.Errorf("validateKindSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateKindSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.waitTimeout" {
		t.Errorf("validateKindSpec() error field = %q, want %q", output.Errors[0].Field, "spec.waitTimeout")
	}
}

func TestConfigValidate_InvalidRetainType(t *testing.T) {
	// retain is not a bool (it's a string)
	spec := map[string]interface{}{
		"retain": "true",
	}

	output := validateKindSpec(spec)

	if output.Valid {
		t.Errorf("validateKindSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateKindSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.retain" {
		t.Errorf("validateKindSpec() error field = %q, want %q", output.Errors[0].Field, "spec.retain")
	}
}

func TestConfigValidate_MultipleErrors(t *testing.T) {
	// Multiple invalid fields
	spec := map[string]interface{}{
		"name":        123,             // invalid: int instead of string
		"image":       true,            // invalid: bool instead of string
		"config":      []interface{}{}, // invalid: array instead of string
		"waitTimeout": 300,             // invalid: int instead of string
		"retain":      "yes",           // invalid: string instead of bool
	}

	output := validateKindSpec(spec)

	if output.Valid {
		t.Errorf("validateKindSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 5 {
		t.Errorf("validateKindSpec() errors count = %d, want 5", len(output.Errors))
	}
}

func TestConfigValidate_ValidPartialSpec(t *testing.T) {
	// Valid spec with only some fields populated
	tests := []struct {
		name string
		spec map[string]interface{}
	}{
		{
			name: "name only",
			spec: map[string]interface{}{
				"name": "my-cluster",
			},
		},
		{
			name: "image only",
			spec: map[string]interface{}{
				"image": "kindest/node:v1.30.0",
			},
		},
		{
			name: "config only",
			spec: map[string]interface{}{
				"config": "/path/to/config.yaml",
			},
		},
		{
			name: "waitTimeout only",
			spec: map[string]interface{}{
				"waitTimeout": "10m",
			},
		},
		{
			name: "retain true",
			spec: map[string]interface{}{
				"retain": true,
			},
		},
		{
			name: "retain false",
			spec: map[string]interface{}{
				"retain": false,
			},
		},
		{
			name: "name and image",
			spec: map[string]interface{}{
				"name":  "my-cluster",
				"image": "kindest/node:v1.30.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := validateKindSpec(tt.spec)

			if !output.Valid {
				t.Errorf("validateKindSpec() valid = %v, want true", output.Valid)
			}
			if len(output.Errors) != 0 {
				t.Errorf("validateKindSpec() errors = %v, want none", output.Errors)
			}
		})
	}
}

func TestConfigValidate_InvalidRetainIntegerTypes(t *testing.T) {
	// retain with integer values (should be bool)
	tests := []struct {
		name string
		spec map[string]interface{}
	}{
		{
			name: "retain as integer 1",
			spec: map[string]interface{}{
				"retain": 1,
			},
		},
		{
			name: "retain as integer 0",
			spec: map[string]interface{}{
				"retain": 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := validateKindSpec(tt.spec)

			if output.Valid {
				t.Errorf("validateKindSpec() valid = %v, want false", output.Valid)
			}
			if len(output.Errors) != 1 {
				t.Fatalf("validateKindSpec() errors count = %d, want 1", len(output.Errors))
			}
			if output.Errors[0].Field != "spec.retain" {
				t.Errorf("validateKindSpec() error field = %q, want %q", output.Errors[0].Field, "spec.retain")
			}
		})
	}
}

func TestConfigValidate_EmptyStringValues(t *testing.T) {
	// Empty strings are valid (optional fields)
	spec := map[string]interface{}{
		"name":        "",
		"image":       "",
		"config":      "",
		"waitTimeout": "",
	}

	output := validateKindSpec(spec)

	if !output.Valid {
		t.Errorf("validateKindSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateKindSpec() errors = %v, want none", output.Errors)
	}
}

func TestConfigValidate_NilFieldValues(t *testing.T) {
	// Nil values should fail validation (type assertion fails)
	tests := []struct {
		name          string
		spec          map[string]interface{}
		expectedField string
	}{
		{
			name:          "nil name",
			spec:          map[string]interface{}{"name": nil},
			expectedField: "spec.name",
		},
		{
			name:          "nil image",
			spec:          map[string]interface{}{"image": nil},
			expectedField: "spec.image",
		},
		{
			name:          "nil config",
			spec:          map[string]interface{}{"config": nil},
			expectedField: "spec.config",
		},
		{
			name:          "nil waitTimeout",
			spec:          map[string]interface{}{"waitTimeout": nil},
			expectedField: "spec.waitTimeout",
		},
		{
			name:          "nil retain",
			spec:          map[string]interface{}{"retain": nil},
			expectedField: "spec.retain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := validateKindSpec(tt.spec)

			if output.Valid {
				t.Errorf("validateKindSpec() valid = %v, want false", output.Valid)
			}
			if len(output.Errors) != 1 {
				t.Fatalf("validateKindSpec() errors count = %d, want 1", len(output.Errors))
			}
			if output.Errors[0].Field != tt.expectedField {
				t.Errorf("validateKindSpec() error field = %q, want %q", output.Errors[0].Field, tt.expectedField)
			}
		})
	}
}

func TestConfigValidate_UnknownFieldsIgnored(t *testing.T) {
	// Unknown fields should be ignored (not validated)
	spec := map[string]interface{}{
		"name":         "my-cluster",
		"unknownField": "some value",
		"anotherOne":   123,
	}

	output := validateKindSpec(spec)

	if !output.Valid {
		t.Errorf("validateKindSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateKindSpec() errors = %v, want none", output.Errors)
	}
}
