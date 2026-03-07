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
	"os"
	"path/filepath"
	"testing"
)

func TestReadSpec_CUConfig(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		expectNil   bool
		compoURL    string
		fileCount   int
	}{
		{
			name: "cu_section_present",
			yamlContent: `
name: test-project
artifactStorePath: .forge/artifacts.yaml
cu:
  compoURL: https://github.com/org/workspace
  managedFiles:
    - go.work
    - "*/go.mod"
`,
			expectNil: false,
			compoURL:  "https://github.com/org/workspace",
			fileCount: 2,
		},
		{
			name: "cu_section_absent",
			yamlContent: `
name: test-project
artifactStorePath: .forge/artifacts.yaml
`,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			yamlPath := filepath.Join(tmpDir, "forge.yaml")

			if err := os.WriteFile(yamlPath, []byte(tt.yamlContent), 0o644); err != nil {
				t.Fatalf("Failed to write test YAML: %v", err)
			}

			spec, err := ReadSpecFromPath(yamlPath)
			if err != nil {
				t.Fatalf("ReadSpecFromPath() error = %v, want nil", err)
			}

			if tt.expectNil {
				if spec.CU != nil {
					t.Errorf("Expected spec.CU to be nil, got %+v", spec.CU)
				}
				return
			}

			if spec.CU == nil {
				t.Fatalf("Expected spec.CU to be non-nil")
			}

			if spec.CU.CompoURL != tt.compoURL {
				t.Errorf("CompoURL = %q, want %q", spec.CU.CompoURL, tt.compoURL)
			}

			if len(spec.CU.ManagedFiles) != tt.fileCount {
				t.Fatalf("ManagedFiles length = %d, want %d", len(spec.CU.ManagedFiles), tt.fileCount)
			}

			expectedFiles := []string{"go.work", "*/go.mod"}
			for i, f := range expectedFiles {
				if spec.CU.ManagedFiles[i] != f {
					t.Errorf("ManagedFiles[%d] = %q, want %q", i, spec.CU.ManagedFiles[i], f)
				}
			}
		})
	}
}
