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

// TestBuildBatchesKeepDeclarationOrderAcrossEngines pins the fix for the
// cross-engine ordering race: gen (engine A) declared before build
// (engine B) must run before it, and a return to engine A opens a new
// group rather than merging into the first.
func TestBuildBatchesKeepDeclarationOrderAcrossEngines(t *testing.T) {
	sequence := []struct{ engine, name string }{
		{"forge://forge-dev", "gen-a"},
		{"forge://forge-dev", "gen-b"},
		{"forge://go-build", "a"},
		{"forge://forge-dev", "gen-c"},
		{"forge://go-build", "c"},
	}

	var groups []engineGroup
	for _, s := range sequence {
		groups = appendSpecToGroups(groups, s.engine, map[string]any{"name": s.name})
	}

	want := []struct {
		engine string
		names  []string
	}{
		{"forge://forge-dev", []string{"gen-a", "gen-b"}},
		{"forge://go-build", []string{"a"}},
		{"forge://forge-dev", []string{"gen-c"}},
		{"forge://go-build", []string{"c"}},
	}

	if len(groups) != len(want) {
		t.Fatalf("want %d groups in declaration order, got %d: %+v", len(want), len(groups), groups)
	}

	for i, w := range want {
		if groups[i].engine != w.engine || len(groups[i].specs) != len(w.names) {
			t.Fatalf("group %d: want %s with %d specs, got %s with %d", i, w.engine, len(w.names), groups[i].engine, len(groups[i].specs))
		}

		for j, name := range w.names {
			if groups[i].specs[j]["name"] != name {
				t.Errorf("group %d spec %d: want %s, got %v", i, j, name, groups[i].specs[j]["name"])
			}
		}
	}
}
