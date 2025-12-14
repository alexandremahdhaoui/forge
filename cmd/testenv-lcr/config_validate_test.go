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
		"enabled":                   true,
		"namespace":                 "testenv-lcr",
		"imagePullSecretNamespaces": []interface{}{"default", "kube-system"},
		"imagePullSecretName":       "lcr-credentials",
		"images": []interface{}{
			map[string]interface{}{
				"source": "docker.io/library/nginx:latest",
				"dest":   "nginx:latest",
			},
		},
	}

	output := validateLcrSpec(spec)

	if !output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateLcrSpec() errors = %v, want none", output.Errors)
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
			output := validateLcrSpec(tt.spec)

			if !output.Valid {
				t.Errorf("validateLcrSpec() valid = %v, want true", output.Valid)
			}
			if len(output.Errors) != 0 {
				t.Errorf("validateLcrSpec() errors = %v, want none", output.Errors)
			}
		})
	}
}

func TestConfigValidate_InvalidEnabledType(t *testing.T) {
	// enabled is not a bool (it's a string)
	spec := map[string]interface{}{
		"enabled": "true",
	}

	output := validateLcrSpec(spec)

	if output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateLcrSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.enabled" {
		t.Errorf("validateLcrSpec() error field = %q, want %q", output.Errors[0].Field, "spec.enabled")
	}
}

func TestConfigValidate_InvalidEnabledTypeInt(t *testing.T) {
	// enabled is not a bool (it's an int)
	spec := map[string]interface{}{
		"enabled": 1,
	}

	output := validateLcrSpec(spec)

	if output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateLcrSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.enabled" {
		t.Errorf("validateLcrSpec() error field = %q, want %q", output.Errors[0].Field, "spec.enabled")
	}
}

func TestConfigValidate_InvalidNamespaceType(t *testing.T) {
	// namespace is not a string (it's an int)
	spec := map[string]interface{}{
		"namespace": 123,
	}

	output := validateLcrSpec(spec)

	if output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateLcrSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.namespace" {
		t.Errorf("validateLcrSpec() error field = %q, want %q", output.Errors[0].Field, "spec.namespace")
	}
}

func TestConfigValidate_InvalidNamespaceTypeBool(t *testing.T) {
	// namespace is not a string (it's a bool)
	spec := map[string]interface{}{
		"namespace": true,
	}

	output := validateLcrSpec(spec)

	if output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateLcrSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.namespace" {
		t.Errorf("validateLcrSpec() error field = %q, want %q", output.Errors[0].Field, "spec.namespace")
	}
}

func TestConfigValidate_InvalidImagePullSecretNamespacesType(t *testing.T) {
	// imagePullSecretNamespaces is not an array (it's a string)
	spec := map[string]interface{}{
		"imagePullSecretNamespaces": "default",
	}

	output := validateLcrSpec(spec)

	if output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateLcrSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.imagePullSecretNamespaces" {
		t.Errorf("validateLcrSpec() error field = %q, want %q", output.Errors[0].Field, "spec.imagePullSecretNamespaces")
	}
}

func TestConfigValidate_InvalidImagePullSecretNamespacesTypeInt(t *testing.T) {
	// imagePullSecretNamespaces is not an array (it's an int)
	spec := map[string]interface{}{
		"imagePullSecretNamespaces": 123,
	}

	output := validateLcrSpec(spec)

	if output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateLcrSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.imagePullSecretNamespaces" {
		t.Errorf("validateLcrSpec() error field = %q, want %q", output.Errors[0].Field, "spec.imagePullSecretNamespaces")
	}
}

func TestConfigValidate_InvalidImagePullSecretNamespacesElement(t *testing.T) {
	// imagePullSecretNamespaces array contains a non-string element
	spec := map[string]interface{}{
		"imagePullSecretNamespaces": []interface{}{"default", 123, "kube-system"},
	}

	output := validateLcrSpec(spec)

	if output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateLcrSpec() errors count = %d, want 1", len(output.Errors))
	}
	// The error should point to the specific array index
	if output.Errors[0].Field != "spec.imagePullSecretNamespaces[1]" {
		t.Errorf("validateLcrSpec() error field = %q, want %q", output.Errors[0].Field, "spec.imagePullSecretNamespaces[1]")
	}
}

func TestConfigValidate_InvalidImagePullSecretNameType(t *testing.T) {
	// imagePullSecretName is not a string (it's an int)
	spec := map[string]interface{}{
		"imagePullSecretName": 123,
	}

	output := validateLcrSpec(spec)

	if output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateLcrSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.imagePullSecretName" {
		t.Errorf("validateLcrSpec() error field = %q, want %q", output.Errors[0].Field, "spec.imagePullSecretName")
	}
}

func TestConfigValidate_InvalidImagePullSecretNameTypeBool(t *testing.T) {
	// imagePullSecretName is not a string (it's a bool)
	spec := map[string]interface{}{
		"imagePullSecretName": false,
	}

	output := validateLcrSpec(spec)

	if output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateLcrSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.imagePullSecretName" {
		t.Errorf("validateLcrSpec() error field = %q, want %q", output.Errors[0].Field, "spec.imagePullSecretName")
	}
}

func TestConfigValidate_InvalidImagesType(t *testing.T) {
	// images is not an array (it's a string)
	spec := map[string]interface{}{
		"images": "nginx:latest",
	}

	output := validateLcrSpec(spec)

	if output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateLcrSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.images" {
		t.Errorf("validateLcrSpec() error field = %q, want %q", output.Errors[0].Field, "spec.images")
	}
}

func TestConfigValidate_InvalidImagesTypeInt(t *testing.T) {
	// images is not an array (it's an int)
	spec := map[string]interface{}{
		"images": 123,
	}

	output := validateLcrSpec(spec)

	if output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateLcrSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.images" {
		t.Errorf("validateLcrSpec() error field = %q, want %q", output.Errors[0].Field, "spec.images")
	}
}

func TestConfigValidate_InvalidImagesTypeBool(t *testing.T) {
	// images is not an array (it's a bool)
	spec := map[string]interface{}{
		"images": true,
	}

	output := validateLcrSpec(spec)

	if output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateLcrSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.images" {
		t.Errorf("validateLcrSpec() error field = %q, want %q", output.Errors[0].Field, "spec.images")
	}
}

func TestConfigValidate_InvalidImagesTypeMap(t *testing.T) {
	// images is not an array (it's a map)
	spec := map[string]interface{}{
		"images": map[string]interface{}{
			"source": "nginx:latest",
		},
	}

	output := validateLcrSpec(spec)

	if output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want false", output.Valid)
	}
	if len(output.Errors) != 1 {
		t.Fatalf("validateLcrSpec() errors count = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.images" {
		t.Errorf("validateLcrSpec() error field = %q, want %q", output.Errors[0].Field, "spec.images")
	}
}

func TestConfigValidate_MultipleErrors(t *testing.T) {
	// Multiple fields are invalid
	spec := map[string]interface{}{
		"enabled":                   "invalid-not-a-bool",
		"namespace":                 123,
		"imagePullSecretNamespaces": "invalid-not-an-array",
		"imagePullSecretName":       456,
		"images":                    "invalid-not-an-array",
	}

	output := validateLcrSpec(spec)

	if output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want false", output.Valid)
	}
	// We should have 5 errors (one for each invalid field)
	if len(output.Errors) != 5 {
		t.Errorf("validateLcrSpec() errors count = %d, want 5", len(output.Errors))
	}
}

func TestConfigValidate_ValidEnabledOnly(t *testing.T) {
	// Valid enabled without other fields
	spec := map[string]interface{}{
		"enabled": true,
	}

	output := validateLcrSpec(spec)

	if !output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateLcrSpec() errors = %v, want none", output.Errors)
	}
}

func TestConfigValidate_ValidEnabledFalse(t *testing.T) {
	// Valid enabled=false
	spec := map[string]interface{}{
		"enabled": false,
	}

	output := validateLcrSpec(spec)

	if !output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateLcrSpec() errors = %v, want none", output.Errors)
	}
}

func TestConfigValidate_ValidNamespaceOnly(t *testing.T) {
	// Valid namespace without other fields
	spec := map[string]interface{}{
		"namespace": "custom-namespace",
	}

	output := validateLcrSpec(spec)

	if !output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateLcrSpec() errors = %v, want none", output.Errors)
	}
}

func TestConfigValidate_ValidImagePullSecretNamespacesOnly(t *testing.T) {
	// Valid imagePullSecretNamespaces without other fields
	spec := map[string]interface{}{
		"imagePullSecretNamespaces": []interface{}{"default", "kube-system", "my-namespace"},
	}

	output := validateLcrSpec(spec)

	if !output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateLcrSpec() errors = %v, want none", output.Errors)
	}
}

func TestConfigValidate_ValidImagePullSecretNameOnly(t *testing.T) {
	// Valid imagePullSecretName without other fields
	spec := map[string]interface{}{
		"imagePullSecretName": "my-custom-secret",
	}

	output := validateLcrSpec(spec)

	if !output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateLcrSpec() errors = %v, want none", output.Errors)
	}
}

func TestConfigValidate_ValidImagesOnly(t *testing.T) {
	// Valid images without other fields
	spec := map[string]interface{}{
		"images": []interface{}{
			map[string]interface{}{
				"source": "docker.io/library/nginx:latest",
				"dest":   "nginx:latest",
			},
			map[string]interface{}{
				"source": "gcr.io/distroless/base:latest",
				"dest":   "distroless:latest",
			},
		},
	}

	output := validateLcrSpec(spec)

	if !output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateLcrSpec() errors = %v, want none", output.Errors)
	}
}

func TestConfigValidate_ValidEmptyImages(t *testing.T) {
	// Valid empty images array
	spec := map[string]interface{}{
		"images": []interface{}{},
	}

	output := validateLcrSpec(spec)

	if !output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateLcrSpec() errors = %v, want none", output.Errors)
	}
}

func TestConfigValidate_ValidEmptyImagePullSecretNamespaces(t *testing.T) {
	// Valid empty imagePullSecretNamespaces array
	spec := map[string]interface{}{
		"imagePullSecretNamespaces": []interface{}{},
	}

	output := validateLcrSpec(spec)

	if !output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateLcrSpec() errors = %v, want none", output.Errors)
	}
}

func TestConfigValidate_ValidEmptyNamespace(t *testing.T) {
	// Valid empty namespace string (will use default)
	spec := map[string]interface{}{
		"namespace": "",
	}

	output := validateLcrSpec(spec)

	if !output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateLcrSpec() errors = %v, want none", output.Errors)
	}
}

func TestConfigValidate_ValidEmptyImagePullSecretName(t *testing.T) {
	// Valid empty imagePullSecretName string (will use default)
	spec := map[string]interface{}{
		"imagePullSecretName": "",
	}

	output := validateLcrSpec(spec)

	if !output.Valid {
		t.Errorf("validateLcrSpec() valid = %v, want true", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("validateLcrSpec() errors = %v, want none", output.Errors)
	}
}
