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

package forge

import (
	"testing"
)

// TestParseDependsOn tests the ParseDependsOn function with various valid and invalid inputs
func TestParseDependsOn(t *testing.T) {
	tests := []struct {
		name        string
		spec        map[string]interface{}
		want        []DependsOnSpec
		wantErr     bool
		errContains string
		description string
	}{
		{
			name: "valid_single_detector",
			spec: map[string]interface{}{
				"dependsOn": []interface{}{
					map[string]interface{}{
						"engine": "go://go-dependency-detector",
						"spec": map[string]interface{}{
							"pattern": "*.go",
						},
					},
				},
			},
			want: []DependsOnSpec{
				{
					Engine: "go://go-dependency-detector",
					Spec: map[string]interface{}{
						"pattern": "*.go",
					},
				},
			},
			wantErr:     false,
			description: "Valid dependsOn with single detector and spec field",
		},
		{
			name:        "missing_dependsOn",
			spec:        map[string]interface{}{},
			want:        nil,
			wantErr:     false,
			description: "Missing dependsOn field should return (nil, nil) - not an error",
		},
		{
			name: "invalid_dependsOn_type",
			spec: map[string]interface{}{
				"dependsOn": "not-an-array",
			},
			want:        nil,
			wantErr:     true,
			errContains: "dependsOn must be an array",
			description: "dependsOn field is not an array",
		},
		{
			name: "invalid_item_type",
			spec: map[string]interface{}{
				"dependsOn": []interface{}{
					"not-an-object",
				},
			},
			want:        nil,
			wantErr:     true,
			errContains: "dependsOn[0] must be an object",
			description: "Item in dependsOn array is not an object",
		},
		{
			name: "missing_engine_field",
			spec: map[string]interface{}{
				"dependsOn": []interface{}{
					map[string]interface{}{
						"spec": map[string]interface{}{
							"pattern": "*.go",
						},
					},
				},
			},
			want:        nil,
			wantErr:     true,
			errContains: "dependsOn[0].engine is required",
			description: "Missing engine field in dependsOn item",
		},
		{
			name: "empty_engine_field",
			spec: map[string]interface{}{
				"dependsOn": []interface{}{
					map[string]interface{}{
						"engine": "",
						"spec": map[string]interface{}{
							"pattern": "*.go",
						},
					},
				},
			},
			want:        nil,
			wantErr:     true,
			errContains: "dependsOn[0].engine is required",
			description: "Empty engine field in dependsOn item",
		},
		{
			name: "multiple_detectors",
			spec: map[string]interface{}{
				"dependsOn": []interface{}{
					map[string]interface{}{
						"engine": "go://detector-one",
						"spec": map[string]interface{}{
							"config": "value1",
						},
					},
					map[string]interface{}{
						"engine": "go://detector-two",
						"spec": map[string]interface{}{
							"config": "value2",
						},
					},
				},
			},
			want: []DependsOnSpec{
				{
					Engine: "go://detector-one",
					Spec: map[string]interface{}{
						"config": "value1",
					},
				},
				{
					Engine: "go://detector-two",
					Spec: map[string]interface{}{
						"config": "value2",
					},
				},
			},
			wantErr:     false,
			description: "Multiple dependency detectors",
		},
		{
			name: "valid_without_spec_field",
			spec: map[string]interface{}{
				"dependsOn": []interface{}{
					map[string]interface{}{
						"engine": "go://simple-detector",
					},
				},
			},
			want: []DependsOnSpec{
				{
					Engine: "go://simple-detector",
					Spec:   nil,
				},
			},
			wantErr:     false,
			description: "Valid detector without spec field (spec is optional)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDependsOn(tt.spec)

			// Check error expectation
			if tt.wantErr {
				if err == nil {
					t.Errorf("%s: ParseDependsOn() error = nil, want error containing %q",
						tt.description, tt.errContains)
					return
				}
				if !containsSubstring(err.Error(), tt.errContains) {
					t.Errorf("%s: ParseDependsOn() error = %q, want error containing %q",
						tt.description, err.Error(), tt.errContains)
				}
				return
			}

			// No error expected
			if err != nil {
				t.Errorf("%s: ParseDependsOn() unexpected error = %v",
					tt.description, err)
				return
			}

			// Compare results
			if !equalDependsOnSpecs(got, tt.want) {
				t.Errorf("%s: ParseDependsOn() = %+v, want %+v",
					tt.description, got, tt.want)
			}
		})
	}
}

// equalDependsOnSpecs compares two slices of DependsOnSpec for equality
func equalDependsOnSpecs(a, b []DependsOnSpec) bool {
	if len(a) != len(b) {
		return false
	}

	// Both nil
	if a == nil && b == nil {
		return true
	}

	// One nil, one not
	if (a == nil) != (b == nil) {
		return false
	}

	for i := range a {
		if a[i].Engine != b[i].Engine {
			return false
		}

		// Compare spec maps
		if !equalMaps(a[i].Spec, b[i].Spec) {
			return false
		}
	}

	return true
}

// equalMaps compares two map[string]interface{} for equality
func equalMaps(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}

	// Both nil
	if a == nil && b == nil {
		return true
	}

	// One nil, one not
	if (a == nil) != (b == nil) {
		return false
	}

	for key, aVal := range a {
		bVal, exists := b[key]
		if !exists {
			return false
		}

		// Simple comparison - works for strings and basic types
		if aVal != bVal {
			return false
		}
	}

	return true
}
