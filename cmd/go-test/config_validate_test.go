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

func TestValidateMap_ValidSpec(t *testing.T) {
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

	output := ValidateMap(spec)

	if !output.Valid {
		t.Errorf("ValidateMap() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("ValidateMap() errors = %v, want none", output.Errors)
	}
}

func TestValidateMap_EmptySpec(t *testing.T) {
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
			output := ValidateMap(tt.spec)

			if !output.Valid {
				t.Errorf("ValidateMap() valid = %v, want true", output.Valid)
			}
			if len(output.Errors) != 0 {
				t.Errorf("ValidateMap() errors = %v, want none", output.Errors)
			}
		})
	}
}

func TestValidateMap_InvalidPackagesType(t *testing.T) {
	// packages is not an array (it's a string)
	spec := map[string]interface{}{
		"packages": "invalid-not-an-array",
	}

	output := ValidateMap(spec)

	if output.Valid {
		t.Errorf("ValidateMap() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("ValidateMap() errors count = %d, want 1", len(output.Errors))
	}
	// Error comes from FromMap parsing
	if output.Errors[0].Field != "spec" {
		t.Errorf("ValidateMap() error field = %q, want %q", output.Errors[0].Field, "spec")
	}
}

func TestValidateMap_InvalidTagsType(t *testing.T) {
	// tags is not an array (it's an int)
	spec := map[string]interface{}{
		"tags": 123,
	}

	output := ValidateMap(spec)

	if output.Valid {
		t.Errorf("ValidateMap() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("ValidateMap() errors count = %d, want 1", len(output.Errors))
	}
}

func TestValidateMap_InvalidTimeoutType(t *testing.T) {
	// timeout is not a string (it's an int)
	spec := map[string]interface{}{
		"timeout": 30,
	}

	output := ValidateMap(spec)

	if output.Valid {
		t.Errorf("ValidateMap() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("ValidateMap() errors count = %d, want 1", len(output.Errors))
	}
}

func TestValidateMap_InvalidRaceType(t *testing.T) {
	// race is not a bool (it's a string)
	spec := map[string]interface{}{
		"race": "true",
	}

	output := ValidateMap(spec)

	if output.Valid {
		t.Errorf("ValidateMap() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("ValidateMap() errors count = %d, want 1", len(output.Errors))
	}
}

func TestValidateMap_InvalidCoverType(t *testing.T) {
	// cover is not a bool (it's a string)
	spec := map[string]interface{}{
		"cover": "false",
	}

	output := ValidateMap(spec)

	if output.Valid {
		t.Errorf("ValidateMap() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("ValidateMap() errors count = %d, want 1", len(output.Errors))
	}
}

func TestValidateMap_InvalidCoverprofileType(t *testing.T) {
	// coverprofile is not a string (it's a bool)
	spec := map[string]interface{}{
		"coverprofile": true,
	}

	output := ValidateMap(spec)

	if output.Valid {
		t.Errorf("ValidateMap() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("ValidateMap() errors count = %d, want 1", len(output.Errors))
	}
}

func TestValidateMap_InvalidArgsType(t *testing.T) {
	// args is not an array (it's a string)
	spec := map[string]interface{}{
		"args": "invalid-not-an-array",
	}

	output := ValidateMap(spec)

	if output.Valid {
		t.Errorf("ValidateMap() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("ValidateMap() errors count = %d, want 1", len(output.Errors))
	}
}

func TestValidateMap_InvalidEnvType(t *testing.T) {
	// env is not a map (it's a string)
	spec := map[string]interface{}{
		"env": "invalid-not-a-map",
	}

	output := ValidateMap(spec)

	if output.Valid {
		t.Errorf("ValidateMap() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("ValidateMap() errors count = %d, want 1", len(output.Errors))
	}
}

func TestValidateMap_InvalidPackagesElement(t *testing.T) {
	// packages array contains a non-string element
	spec := map[string]interface{}{
		"packages": []interface{}{"./cmd/...", 123, "./pkg/..."},
	}

	output := ValidateMap(spec)

	if output.Valid {
		t.Errorf("ValidateMap() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("ValidateMap() errors count = %d, want 1", len(output.Errors))
	}
}

func TestValidateMap_InvalidTagsElement(t *testing.T) {
	// tags array contains a non-string element
	spec := map[string]interface{}{
		"tags": []interface{}{"unit", true, "e2e"},
	}

	output := ValidateMap(spec)

	if output.Valid {
		t.Errorf("ValidateMap() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("ValidateMap() errors count = %d, want 1", len(output.Errors))
	}
}

func TestValidateMap_InvalidArgsElement(t *testing.T) {
	// args array contains a non-string element
	spec := map[string]interface{}{
		"args": []interface{}{"-v", 123, "-count=1"},
	}

	output := ValidateMap(spec)

	if output.Valid {
		t.Errorf("ValidateMap() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("ValidateMap() errors count = %d, want 1", len(output.Errors))
	}
}

func TestValidateMap_InvalidEnvValue(t *testing.T) {
	// env map contains a non-string value
	spec := map[string]interface{}{
		"env": map[string]interface{}{
			"CGO_ENABLED": "0",
			"GOOS":        123, // invalid: number instead of string
		},
	}

	output := ValidateMap(spec)

	if output.Valid {
		t.Errorf("ValidateMap() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("ValidateMap() errors count = %d, want 1", len(output.Errors))
	}
}

func TestValidateMap_ValidPackagesOnly(t *testing.T) {
	// Valid packages without other fields
	spec := map[string]interface{}{
		"packages": []interface{}{"./cmd/...", "./pkg/..."},
	}

	output := ValidateMap(spec)

	if !output.Valid {
		t.Errorf("ValidateMap() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("ValidateMap() errors = %v, want none", output.Errors)
	}
}

func TestValidateMap_ValidTagsOnly(t *testing.T) {
	// Valid tags without other fields
	spec := map[string]interface{}{
		"tags": []interface{}{"unit", "integration"},
	}

	output := ValidateMap(spec)

	if !output.Valid {
		t.Errorf("ValidateMap() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("ValidateMap() errors = %v, want none", output.Errors)
	}
}

func TestValidateMap_ValidBooleanFields(t *testing.T) {
	// Valid boolean fields only
	spec := map[string]interface{}{
		"race":  false,
		"cover": true,
	}

	output := ValidateMap(spec)

	if !output.Valid {
		t.Errorf("ValidateMap() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("ValidateMap() errors = %v, want none", output.Errors)
	}
}

func TestValidateMap_ValidStringFields(t *testing.T) {
	// Valid string fields only
	spec := map[string]interface{}{
		"timeout":      "10m",
		"coverprofile": "/tmp/coverage.out",
	}

	output := ValidateMap(spec)

	if !output.Valid {
		t.Errorf("ValidateMap() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("ValidateMap() errors = %v, want none", output.Errors)
	}
}

func TestValidateMap_ValidEnvOnly(t *testing.T) {
	// Valid env without other fields
	spec := map[string]interface{}{
		"env": map[string]interface{}{
			"CGO_ENABLED": "0",
		},
	}

	output := ValidateMap(spec)

	if !output.Valid {
		t.Errorf("ValidateMap() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("ValidateMap() errors = %v, want none", output.Errors)
	}
}

func TestValidateMap_ValidArgsOnly(t *testing.T) {
	// Valid args without other fields
	spec := map[string]interface{}{
		"args": []interface{}{"-v", "-count=1"},
	}

	output := ValidateMap(spec)

	if !output.Valid {
		t.Errorf("ValidateMap() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("ValidateMap() errors = %v, want none", output.Errors)
	}
}

func TestValidateMap_InvalidCoverprofileNotString(t *testing.T) {
	// coverprofile is an array instead of string
	spec := map[string]interface{}{
		"coverprofile": []interface{}{"coverage1.out", "coverage2.out"},
	}

	output := ValidateMap(spec)

	if output.Valid {
		t.Errorf("ValidateMap() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("ValidateMap() errors count = %d, want 1", len(output.Errors))
	}
}

// Additional tests for FromMap and Spec conversion

func TestFromMap_ValidSpec(t *testing.T) {
	m := map[string]interface{}{
		"packages":     []interface{}{"./cmd/...", "./pkg/..."},
		"tags":         []interface{}{"unit"},
		"timeout":      "30m",
		"race":         true,
		"cover":        true,
		"coverprofile": "coverage.out",
		"args":         []interface{}{"-v"},
		"env": map[string]interface{}{
			"CGO_ENABLED": "0",
		},
	}

	spec, err := FromMap(m)
	if err != nil {
		t.Fatalf("FromMap() error = %v, want nil", err)
	}

	if len(spec.Packages) != 2 {
		t.Errorf("FromMap() packages = %v, want 2 elements", spec.Packages)
	}
	if len(spec.Tags) != 1 {
		t.Errorf("FromMap() tags = %v, want 1 element", spec.Tags)
	}
	if spec.Timeout != "30m" {
		t.Errorf("FromMap() timeout = %v, want 30m", spec.Timeout)
	}
	if !spec.Race {
		t.Errorf("FromMap() race = %v, want true", spec.Race)
	}
	if !spec.Cover {
		t.Errorf("FromMap() cover = %v, want true", spec.Cover)
	}
	if spec.Coverprofile != "coverage.out" {
		t.Errorf("FromMap() coverprofile = %v, want coverage.out", spec.Coverprofile)
	}
	if len(spec.Args) != 1 {
		t.Errorf("FromMap() args = %v, want 1 element", spec.Args)
	}
	if len(spec.Env) != 1 {
		t.Errorf("FromMap() env = %v, want 1 element", spec.Env)
	}
}

func TestSpec_ToMap(t *testing.T) {
	spec := &Spec{
		Packages:     []string{"./cmd/..."},
		Tags:         []string{"unit"},
		Timeout:      "30m",
		Race:         true,
		Cover:        true,
		Coverprofile: "coverage.out",
		Args:         []string{"-v"},
		Env:          map[string]string{"CGO_ENABLED": "0"},
	}

	m := spec.ToMap()

	if m["packages"] == nil {
		t.Error("ToMap() packages is nil, want non-nil")
	}
	if m["tags"] == nil {
		t.Error("ToMap() tags is nil, want non-nil")
	}
	if m["timeout"] != "30m" {
		t.Errorf("ToMap() timeout = %v, want 30m", m["timeout"])
	}
	if m["race"] != true {
		t.Errorf("ToMap() race = %v, want true", m["race"])
	}
	if m["cover"] != true {
		t.Errorf("ToMap() cover = %v, want true", m["cover"])
	}
	if m["coverprofile"] != "coverage.out" {
		t.Errorf("ToMap() coverprofile = %v, want coverage.out", m["coverprofile"])
	}
	if m["args"] == nil {
		t.Error("ToMap() args is nil, want non-nil")
	}
	if m["env"] == nil {
		t.Error("ToMap() env is nil, want non-nil")
	}
}

func TestFromMap_NilMap(t *testing.T) {
	spec, err := FromMap(nil)
	if err != nil {
		t.Fatalf("FromMap(nil) error = %v, want nil", err)
	}
	if spec == nil {
		t.Fatal("FromMap(nil) returned nil spec, want non-nil")
	}
}

func TestSpec_ToMap_Nil(t *testing.T) {
	var spec *Spec = nil
	m := spec.ToMap()
	if m != nil {
		t.Errorf("nil.ToMap() = %v, want nil", m)
	}
}
