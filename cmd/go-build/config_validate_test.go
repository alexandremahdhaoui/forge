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

func TestConfigValidate_ValidSpec(t *testing.T) {
	// Valid spec with both args and env
	spec := map[string]interface{}{
		"args": []interface{}{"-tags=netgo", "-ldflags=-w -s"},
		"env": map[string]interface{}{
			"GOOS":   "linux",
			"GOARCH": "amd64",
		},
	}

	output := validateSpec(spec)

	if !output.Valid {
		t.Errorf("validateSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateSpec() errors = %v, want none", output.Errors)
	}
}

func TestConfigValidate_EmptySpec(t *testing.T) {
	// Empty spec should be valid
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
			output := validateSpec(tt.spec)

			if !output.Valid {
				t.Errorf("validateSpec() valid = %v, want true", output.Valid)
			}
			if len(output.Errors) != 0 {
				t.Errorf("validateSpec() errors = %v, want none", output.Errors)
			}
		})
	}
}

func TestConfigValidate_InvalidArgsType(t *testing.T) {
	// args is not an array (it's a string)
	spec := map[string]interface{}{
		"args": "invalid-not-an-array",
	}

	output := validateSpec(spec)

	if output.Valid {
		t.Errorf("validateSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.args" {
		t.Errorf("validateSpec() error field = %q, want %q", output.Errors[0].Field, "spec.args")
	}
}

func TestConfigValidate_InvalidEnvType(t *testing.T) {
	// env is not a map (it's a string)
	spec := map[string]interface{}{
		"env": "invalid-not-a-map",
	}

	output := validateSpec(spec)

	if output.Valid {
		t.Errorf("validateSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.env" {
		t.Errorf("validateSpec() error field = %q, want %q", output.Errors[0].Field, "spec.env")
	}
}

func TestConfigValidate_InvalidArgsElement(t *testing.T) {
	// args array contains a non-string element
	spec := map[string]interface{}{
		"args": []interface{}{"-tags=netgo", 123, "-ldflags=-w"},
	}

	output := validateSpec(spec)

	if output.Valid {
		t.Errorf("validateSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateSpec() errors count = %d, want 1", len(output.Errors))
	}
	// The error should point to the specific array index
	if output.Errors[0].Field != "spec.args[1]" {
		t.Errorf("validateSpec() error field = %q, want %q", output.Errors[0].Field, "spec.args[1]")
	}
}

func TestConfigValidate_InvalidEnvValue(t *testing.T) {
	// env map contains a non-string value
	spec := map[string]interface{}{
		"env": map[string]interface{}{
			"GOOS":   "linux",
			"GOARCH": 123, // invalid: number instead of string
		},
	}

	output := validateSpec(spec)

	if output.Valid {
		t.Errorf("validateSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateSpec() errors count = %d, want 1", len(output.Errors))
	}
	// The error should point to the specific map key
	if output.Errors[0].Field != "spec.env.GOARCH" {
		t.Errorf("validateSpec() error field = %q, want %q", output.Errors[0].Field, "spec.env.GOARCH")
	}
}

func TestConfigValidate_MultipleErrors(t *testing.T) {
	// Both args and env are invalid
	spec := map[string]interface{}{
		"args": "invalid-not-an-array",
		"env":  "invalid-not-a-map",
	}

	output := validateSpec(spec)

	if output.Valid {
		t.Errorf("validateSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 2 {
		t.Errorf("validateSpec() errors count = %d, want 2", len(output.Errors))
	}
}

func TestConfigValidate_ValidArgsOnly(t *testing.T) {
	// Valid args without env
	spec := map[string]interface{}{
		"args": []interface{}{"-tags=netgo"},
	}

	output := validateSpec(spec)

	if !output.Valid {
		t.Errorf("validateSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateSpec() errors = %v, want none", output.Errors)
	}
}

func TestConfigValidate_ValidEnvOnly(t *testing.T) {
	// Valid env without args
	spec := map[string]interface{}{
		"env": map[string]interface{}{
			"CGO_ENABLED": "0",
		},
	}

	output := validateSpec(spec)

	if !output.Valid {
		t.Errorf("validateSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateSpec() errors = %v, want none", output.Errors)
	}
}
