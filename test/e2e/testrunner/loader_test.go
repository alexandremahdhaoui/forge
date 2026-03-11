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

package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadTestFiles_ValidDir(t *testing.T) {
	dir := t.TempDir()

	// Write two valid YAML files.
	yaml1 := `testCases:
  - name: "test-one"
    steps:
      - command: version
        mode: cli
`
	yaml2 := `testCases:
  - name: "test-two"
    steps:
      - command: help
        mode: cli
`
	if err := os.WriteFile(filepath.Join(dir, "one.yaml"), []byte(yaml1), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "two.yaml"), []byte(yaml2), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	testFiles, paths, err := LoadTestFiles(dir)
	if err != nil {
		t.Fatalf("LoadTestFiles() unexpected error: %v", err)
	}

	if len(testFiles) != 2 {
		t.Fatalf("expected 2 test files, got %d", len(testFiles))
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}

	// Verify test case names are present across the loaded files.
	names := make(map[string]bool)
	for _, tf := range testFiles {
		for _, tc := range tf.TestCases {
			names[tc.Name] = true
		}
	}
	if !names["test-one"] {
		t.Error("missing test case 'test-one'")
	}
	if !names["test-two"] {
		t.Error("missing test case 'test-two'")
	}
}

func TestLoadTestFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	_, _, err := LoadTestFiles(dir)
	if err == nil {
		t.Fatal("expected error for empty dir, got nil")
	}
	if !strings.Contains(err.Error(), "no YAML files found") {
		t.Errorf("expected 'no YAML files found' error, got: %v", err)
	}
}

func TestLoadTestFiles_InvalidYAML(t *testing.T) {
	dir := t.TempDir()

	// Write an invalid YAML file.
	invalid := `testCases:
  - name: [this is invalid
    steps: }`
	if err := os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte(invalid), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, _, err := LoadTestFiles(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
	if !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected error to contain 'parsing', got: %v", err)
	}
	if !strings.Contains(err.Error(), "bad.yaml") {
		t.Errorf("expected error to contain file name 'bad.yaml', got: %v", err)
	}
}

func TestLoadTestFiles_NestedDirs(t *testing.T) {
	dir := t.TempDir()

	// Create nested directory structure.
	subdir1 := filepath.Join(dir, "build")
	subdir2 := filepath.Join(dir, "system", "sub")
	if err := os.MkdirAll(subdir1, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.MkdirAll(subdir2, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	yaml1 := `testCases:
  - name: "build-test"
    steps:
      - command: build
        mode: cli
`
	yaml2 := `testCases:
  - name: "system-test"
    steps:
      - command: version
        mode: cli
`
	yaml3 := `testCases:
  - name: "deep-nested-test"
    steps:
      - command: help
        mode: cli
`
	if err := os.WriteFile(filepath.Join(subdir1, "basic.yaml"), []byte(yaml1), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "root.yaml"), []byte(yaml2), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subdir2, "deep.yaml"), []byte(yaml3), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Also write a non-YAML file that should be ignored.
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not yaml"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	testFiles, paths, err := LoadTestFiles(dir)
	if err != nil {
		t.Fatalf("LoadTestFiles() unexpected error: %v", err)
	}

	if len(testFiles) != 3 {
		t.Fatalf("expected 3 test files, got %d", len(testFiles))
	}
	if len(paths) != 3 {
		t.Fatalf("expected 3 paths, got %d", len(paths))
	}

	// Verify all test case names are present.
	names := make(map[string]bool)
	for _, tf := range testFiles {
		for _, tc := range tf.TestCases {
			names[tc.Name] = true
		}
	}
	if !names["build-test"] {
		t.Error("missing test case 'build-test'")
	}
	if !names["system-test"] {
		t.Error("missing test case 'system-test'")
	}
	if !names["deep-nested-test"] {
		t.Error("missing test case 'deep-nested-test'")
	}

	// Verify non-YAML file was not loaded.
	for _, p := range paths {
		if strings.HasSuffix(p, ".txt") {
			t.Errorf("non-YAML file was loaded: %s", p)
		}
	}
}
