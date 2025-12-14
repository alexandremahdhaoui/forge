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

	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

func TestDeduplicateDependencies(t *testing.T) {
	tests := []struct {
		name     string
		input    []forge.ArtifactDependency
		expected int // expected number of unique dependencies
	}{
		{
			name:     "empty slice",
			input:    []forge.ArtifactDependency{},
			expected: 0,
		},
		{
			name: "no duplicates",
			input: []forge.ArtifactDependency{
				{Type: "file", FilePath: "/path/to/file1.go"},
				{Type: "file", FilePath: "/path/to/file2.go"},
				{Type: "externalPackage", ExternalPackage: "github.com/foo/bar"},
			},
			expected: 3,
		},
		{
			name: "duplicate files",
			input: []forge.ArtifactDependency{
				{Type: "file", FilePath: "/path/to/file1.go", Timestamp: "2025-11-23T10:00:00Z"},
				{Type: "file", FilePath: "/path/to/file1.go", Timestamp: "2025-11-23T11:00:00Z"}, // duplicate
				{Type: "file", FilePath: "/path/to/file2.go"},
			},
			expected: 2,
		},
		{
			name: "duplicate external packages",
			input: []forge.ArtifactDependency{
				{Type: "externalPackage", ExternalPackage: "github.com/foo/bar", Semver: "v1.0.0"},
				{Type: "externalPackage", ExternalPackage: "github.com/foo/bar", Semver: "v1.1.0"}, // duplicate
				{Type: "externalPackage", ExternalPackage: "github.com/baz/qux"},
			},
			expected: 2,
		},
		{
			name: "mixed duplicates",
			input: []forge.ArtifactDependency{
				{Type: "file", FilePath: "/path/to/file1.go"},
				{Type: "file", FilePath: "/path/to/file1.go"}, // duplicate
				{Type: "externalPackage", ExternalPackage: "github.com/foo/bar"},
				{Type: "externalPackage", ExternalPackage: "github.com/foo/bar"}, // duplicate
				{Type: "file", FilePath: "/path/to/file2.go"},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicateDependencies(tt.input)
			if len(result) != tt.expected {
				t.Errorf("deduplicateDependencies() returned %d dependencies, expected %d", len(result), tt.expected)
			}

			// Verify no duplicates in result
			seen := make(map[string]bool)
			for _, dep := range result {
				var key string
				if dep.Type == "file" {
					key = "file:" + dep.FilePath
				} else {
					key = "external:" + dep.ExternalPackage
				}

				if seen[key] {
					t.Errorf("duplicate dependency found in result: %s", key)
				}
				seen[key] = true
			}
		})
	}
}
