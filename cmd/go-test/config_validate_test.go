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
	// Valid spec with all fields
	spec := map[string]interface{}{
		"packages":     []interface{}{"./cmd/...", "./pkg/..."},
		"tags":         []interface{}{"unit", "integration"},
		"timeout":      "30m",
		"race":         true,
		"cover":        true,
		"coverprofile": "coverage.out",
		"args":         []interface{}{"-v", "-count=1"},
		"env": map[string]interface{}{
			"CGO_ENABLED": "0",
			"GOOS":        "linux",
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

func TestConfigValidate_InvalidPackagesType(t *testing.T) {
	// packages is not an array (it's a string)
	spec := map[string]interface{}{
		"packages": "invalid-not-an-array",
	}

	output := validateSpec(spec)

	if output.Valid {
		t.Errorf("validateSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.packages" {
		t.Errorf("validateSpec() error field = %q, want %q", output.Errors[0].Field, "spec.packages")
	}
}

func TestConfigValidate_InvalidTagsType(t *testing.T) {
	// tags is not an array (it's an int)
	spec := map[string]interface{}{
		"tags": 123,
	}

	output := validateSpec(spec)

	if output.Valid {
		t.Errorf("validateSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.tags" {
		t.Errorf("validateSpec() error field = %q, want %q", output.Errors[0].Field, "spec.tags")
	}
}

func TestConfigValidate_InvalidTimeoutType(t *testing.T) {
	// timeout is not a string (it's an int)
	spec := map[string]interface{}{
		"timeout": 30,
	}

	output := validateSpec(spec)

	if output.Valid {
		t.Errorf("validateSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.timeout" {
		t.Errorf("validateSpec() error field = %q, want %q", output.Errors[0].Field, "spec.timeout")
	}
}

func TestConfigValidate_InvalidRaceType(t *testing.T) {
	// race is not a bool (it's a string)
	spec := map[string]interface{}{
		"race": "true",
	}

	output := validateSpec(spec)

	if output.Valid {
		t.Errorf("validateSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.race" {
		t.Errorf("validateSpec() error field = %q, want %q", output.Errors[0].Field, "spec.race")
	}
}

func TestConfigValidate_InvalidCoverType(t *testing.T) {
	// cover is not a bool (it's a string)
	spec := map[string]interface{}{
		"cover": "false",
	}

	output := validateSpec(spec)

	if output.Valid {
		t.Errorf("validateSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.cover" {
		t.Errorf("validateSpec() error field = %q, want %q", output.Errors[0].Field, "spec.cover")
	}
}

func TestConfigValidate_InvalidCoverprofileType(t *testing.T) {
	// coverprofile is not a string (it's a bool)
	spec := map[string]interface{}{
		"coverprofile": true,
	}

	output := validateSpec(spec)

	if output.Valid {
		t.Errorf("validateSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.coverprofile" {
		t.Errorf("validateSpec() error field = %q, want %q", output.Errors[0].Field, "spec.coverprofile")
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

func TestConfigValidate_InvalidPackagesElement(t *testing.T) {
	// packages array contains a non-string element
	spec := map[string]interface{}{
		"packages": []interface{}{"./cmd/...", 123, "./pkg/..."},
	}

	output := validateSpec(spec)

	if output.Valid {
		t.Errorf("validateSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateSpec() errors count = %d, want 1", len(output.Errors))
	}
	// The error should point to the specific array index
	if output.Errors[0].Field != "spec.packages[1]" {
		t.Errorf("validateSpec() error field = %q, want %q", output.Errors[0].Field, "spec.packages[1]")
	}
}

func TestConfigValidate_InvalidTagsElement(t *testing.T) {
	// tags array contains a non-string element
	spec := map[string]interface{}{
		"tags": []interface{}{"unit", true, "e2e"},
	}

	output := validateSpec(spec)

	if output.Valid {
		t.Errorf("validateSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateSpec() errors count = %d, want 1", len(output.Errors))
	}
	// The error should point to the specific array index
	if output.Errors[0].Field != "spec.tags[1]" {
		t.Errorf("validateSpec() error field = %q, want %q", output.Errors[0].Field, "spec.tags[1]")
	}
}

func TestConfigValidate_InvalidArgsElement(t *testing.T) {
	// args array contains a non-string element
	spec := map[string]interface{}{
		"args": []interface{}{"-v", 123, "-count=1"},
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
			"CGO_ENABLED": "0",
			"GOOS":        123, // invalid: number instead of string
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
	if output.Errors[0].Field != "spec.env.GOOS" {
		t.Errorf("validateSpec() error field = %q, want %q", output.Errors[0].Field, "spec.env.GOOS")
	}
}

func TestConfigValidate_MultipleErrors(t *testing.T) {
	// Multiple fields are invalid
	spec := map[string]interface{}{
		"packages": "invalid-not-an-array",
		"tags":     123,
		"timeout":  456,
		"race":     "invalid",
		"cover":    "invalid",
		"args":     "invalid-not-an-array",
		"env":      "invalid-not-a-map",
	}

	output := validateSpec(spec)

	if output.Valid {
		t.Errorf("validateSpec() valid = %v, want false", output.Valid)
	}
	// We should have 7 errors (one for each invalid field)
	if len(output.Errors) != 7 {
		t.Errorf("validateSpec() errors count = %d, want 7", len(output.Errors))
	}
}

func TestConfigValidate_ValidPackagesOnly(t *testing.T) {
	// Valid packages without other fields
	spec := map[string]interface{}{
		"packages": []interface{}{"./cmd/...", "./pkg/..."},
	}

	output := validateSpec(spec)

	if !output.Valid {
		t.Errorf("validateSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateSpec() errors = %v, want none", output.Errors)
	}
}

func TestConfigValidate_ValidTagsOnly(t *testing.T) {
	// Valid tags without other fields
	spec := map[string]interface{}{
		"tags": []interface{}{"unit", "integration"},
	}

	output := validateSpec(spec)

	if !output.Valid {
		t.Errorf("validateSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateSpec() errors = %v, want none", output.Errors)
	}
}

func TestConfigValidate_ValidBooleanFields(t *testing.T) {
	// Valid boolean fields only
	spec := map[string]interface{}{
		"race":  false,
		"cover": true,
	}

	output := validateSpec(spec)

	if !output.Valid {
		t.Errorf("validateSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateSpec() errors = %v, want none", output.Errors)
	}
}

func TestConfigValidate_ValidStringFields(t *testing.T) {
	// Valid string fields only
	spec := map[string]interface{}{
		"timeout":      "10m",
		"coverprofile": "/tmp/coverage.out",
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
	// Valid env without other fields
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

func TestConfigValidate_ValidArgsOnly(t *testing.T) {
	// Valid args without other fields
	spec := map[string]interface{}{
		"args": []interface{}{"-v", "-count=1"},
	}

	output := validateSpec(spec)

	if !output.Valid {
		t.Errorf("validateSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateSpec() errors = %v, want none", output.Errors)
	}
}

func TestConfigValidate_InvalidCoverprofileNotString(t *testing.T) {
	// coverprofile is an array instead of string
	spec := map[string]interface{}{
		"coverprofile": []interface{}{"coverage1.out", "coverage2.out"},
	}

	output := validateSpec(spec)

	if output.Valid {
		t.Errorf("validateSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.coverprofile" {
		t.Errorf("validateSpec() error field = %q, want %q", output.Errors[0].Field, "spec.coverprofile")
	}
}
