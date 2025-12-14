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
		"dockerfile": "Dockerfile.custom",
		"context":    "./build",
		"buildArgs": map[string]interface{}{
			"VERSION": "1.0.0",
			"COMMIT":  "abc123",
		},
		"tags":     []interface{}{"myimage:latest", "myimage:v1"},
		"target":   "production",
		"push":     true,
		"registry": "docker.io/myrepo",
	}

	output := validateContainerBuildSpec(spec)

	if !output.Valid {
		t.Errorf("validateContainerBuildSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateContainerBuildSpec() errors = %v, want none", output.Errors)
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
			output := validateContainerBuildSpec(tt.spec)

			if !output.Valid {
				t.Errorf("validateContainerBuildSpec() valid = %v, want true", output.Valid)
			}
			if len(output.Errors) != 0 {
				t.Errorf("validateContainerBuildSpec() errors = %v, want none", output.Errors)
			}
		})
	}
}

func TestConfigValidate_InvalidDockerfileType(t *testing.T) {
	// dockerfile is not a string (it's a number)
	spec := map[string]interface{}{
		"dockerfile": 123,
	}

	output := validateContainerBuildSpec(spec)

	if output.Valid {
		t.Errorf("validateContainerBuildSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateContainerBuildSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.dockerfile" {
		t.Errorf("validateContainerBuildSpec() error field = %q, want %q", output.Errors[0].Field, "spec.dockerfile")
	}
}

func TestConfigValidate_InvalidContextType(t *testing.T) {
	// context is not a string (it's a boolean)
	spec := map[string]interface{}{
		"context": true,
	}

	output := validateContainerBuildSpec(spec)

	if output.Valid {
		t.Errorf("validateContainerBuildSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateContainerBuildSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.context" {
		t.Errorf("validateContainerBuildSpec() error field = %q, want %q", output.Errors[0].Field, "spec.context")
	}
}

func TestConfigValidate_InvalidBuildArgsType(t *testing.T) {
	// buildArgs is not a map (it's a string)
	spec := map[string]interface{}{
		"buildArgs": "invalid-not-a-map",
	}

	output := validateContainerBuildSpec(spec)

	if output.Valid {
		t.Errorf("validateContainerBuildSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateContainerBuildSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.buildArgs" {
		t.Errorf("validateContainerBuildSpec() error field = %q, want %q", output.Errors[0].Field, "spec.buildArgs")
	}
}

func TestConfigValidate_InvalidBuildArgsValue(t *testing.T) {
	// buildArgs map contains a non-string value
	spec := map[string]interface{}{
		"buildArgs": map[string]interface{}{
			"VERSION": "1.0.0",
			"COUNT":   123, // invalid: number instead of string
		},
	}

	output := validateContainerBuildSpec(spec)

	if output.Valid {
		t.Errorf("validateContainerBuildSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateContainerBuildSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.buildArgs.COUNT" {
		t.Errorf("validateContainerBuildSpec() error field = %q, want %q", output.Errors[0].Field, "spec.buildArgs.COUNT")
	}
}

func TestConfigValidate_InvalidTagsType(t *testing.T) {
	// tags is not an array (it's a string)
	spec := map[string]interface{}{
		"tags": "invalid-not-an-array",
	}

	output := validateContainerBuildSpec(spec)

	if output.Valid {
		t.Errorf("validateContainerBuildSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateContainerBuildSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.tags" {
		t.Errorf("validateContainerBuildSpec() error field = %q, want %q", output.Errors[0].Field, "spec.tags")
	}
}

func TestConfigValidate_InvalidTagsElement(t *testing.T) {
	// tags array contains a non-string element
	spec := map[string]interface{}{
		"tags": []interface{}{"myimage:latest", 123, "myimage:v1"},
	}

	output := validateContainerBuildSpec(spec)

	if output.Valid {
		t.Errorf("validateContainerBuildSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateContainerBuildSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.tags[1]" {
		t.Errorf("validateContainerBuildSpec() error field = %q, want %q", output.Errors[0].Field, "spec.tags[1]")
	}
}

func TestConfigValidate_InvalidTargetType(t *testing.T) {
	// target is not a string (it's a number)
	spec := map[string]interface{}{
		"target": 456,
	}

	output := validateContainerBuildSpec(spec)

	if output.Valid {
		t.Errorf("validateContainerBuildSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateContainerBuildSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.target" {
		t.Errorf("validateContainerBuildSpec() error field = %q, want %q", output.Errors[0].Field, "spec.target")
	}
}

func TestConfigValidate_InvalidPushType(t *testing.T) {
	// push is not a bool (it's a string)
	spec := map[string]interface{}{
		"push": "yes",
	}

	output := validateContainerBuildSpec(spec)

	if output.Valid {
		t.Errorf("validateContainerBuildSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateContainerBuildSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.push" {
		t.Errorf("validateContainerBuildSpec() error field = %q, want %q", output.Errors[0].Field, "spec.push")
	}
}

func TestConfigValidate_InvalidRegistryType(t *testing.T) {
	// registry is not a string (it's an array)
	spec := map[string]interface{}{
		"registry": []interface{}{"docker.io", "ghcr.io"},
	}

	output := validateContainerBuildSpec(spec)

	if output.Valid {
		t.Errorf("validateContainerBuildSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateContainerBuildSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.registry" {
		t.Errorf("validateContainerBuildSpec() error field = %q, want %q", output.Errors[0].Field, "spec.registry")
	}
}

func TestConfigValidate_MultipleErrors(t *testing.T) {
	// Multiple fields with invalid types
	spec := map[string]interface{}{
		"dockerfile": 123,            // invalid: number instead of string
		"buildArgs":  "not-a-map",    // invalid: string instead of map
		"tags":       "not-an-array", // invalid: string instead of array
	}

	output := validateContainerBuildSpec(spec)

	if output.Valid {
		t.Errorf("validateContainerBuildSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 3 {
		t.Errorf("validateContainerBuildSpec() errors count = %d, want 3", len(output.Errors))
	}
}

func TestConfigValidate_PartialValidSpec(t *testing.T) {
	tests := []struct {
		name string
		spec map[string]interface{}
	}{
		{
			name: "only dockerfile",
			spec: map[string]interface{}{
				"dockerfile": "Dockerfile",
			},
		},
		{
			name: "only buildArgs",
			spec: map[string]interface{}{
				"buildArgs": map[string]interface{}{
					"VERSION": "1.0.0",
				},
			},
		},
		{
			name: "only tags",
			spec: map[string]interface{}{
				"tags": []interface{}{"myimage:latest"},
			},
		},
		{
			name: "push false",
			spec: map[string]interface{}{
				"push": false,
			},
		},
		{
			name: "context and target",
			spec: map[string]interface{}{
				"context": ".",
				"target":  "builder",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := validateContainerBuildSpec(tt.spec)

			if !output.Valid {
				t.Errorf("validateContainerBuildSpec() valid = %v, want true, errors: %v", output.Valid, output.Errors)
			}
			if len(output.Errors) != 0 {
				t.Errorf("validateContainerBuildSpec() errors = %v, want none", output.Errors)
			}
		})
	}
}
