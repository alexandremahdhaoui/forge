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

import "testing"

func TestNormalizePortAllocEnvKey(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected string
	}{
		{
			name:     "hyphenated identifier",
			id:       "shaper-e2e-api",
			expected: "PORTALLOC_SHAPER_E2E_API",
		},
		{
			name:     "hyphenated identifier for mtls",
			id:       "shaper-e2e-mtls",
			expected: "PORTALLOC_SHAPER_E2E_MTLS",
		},
		{
			name:     "underscored identifier",
			id:       "my_service",
			expected: "PORTALLOC_MY_SERVICE",
		},
		{
			name:     "simple identifier",
			id:       "simple",
			expected: "PORTALLOC_SIMPLE",
		},
		{
			name:     "mixed case identifier",
			id:       "MixedCase-id",
			expected: "PORTALLOC_MIXEDCASE_ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizePortAllocEnvKey(tt.id)
			if got != tt.expected {
				t.Errorf("NormalizePortAllocEnvKey(%q) = %q, want %q", tt.id, got, tt.expected)
			}
		})
	}
}
