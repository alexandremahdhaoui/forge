//go:build e2e

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

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexandremahdhaoui/forge/test/e2e/testrunner"
)

var forgeBinary string

func TestMain(m *testing.M) {
	bin, err := findForgeBinary()
	if err != nil {
		fmt.Fprintf(os.Stderr, "finding forge binary: %v\n", err)
		os.Exit(1)
	}
	forgeBinary = bin

	code := m.Run()
	os.Exit(code)
}

// TestE2EDeclarative discovers and runs YAML-based test files from testdata/.
func TestE2EDeclarative(t *testing.T) {
	root, err := projectRoot()
	if err != nil {
		t.Fatalf("resolving project root: %v", err)
	}
	testdataDir := filepath.Join(root, "test", "e2e", "testdata")

	testFiles, filePaths, err := testrunner.LoadTestFiles(testdataDir)
	if err != nil {
		t.Fatalf("loading test files: %v", err)
	}

	for i, tf := range testFiles {
		// Use the relative path from testdata (sans extension) as the subtest name.
		relPath, err := filepath.Rel(testdataDir, filePaths[i])
		if err != nil {
			relPath = filepath.Base(filePaths[i])
		}
		relPath = strings.TrimSuffix(relPath, filepath.Ext(relPath))
		// Normalize path separators for consistent test names.
		relPath = filepath.ToSlash(relPath)

		t.Run(relPath, func(t *testing.T) {
			for _, tc := range tf.TestCases {
				t.Run(tc.Name, func(t *testing.T) {
					data := &testrunner.TemplateData{
						Binary:    forgeBinary,
						Workspace: t.TempDir(),
						CWD:       root,
						Env:       make(map[string]string),
						Steps:     make(map[string]map[string]interface{}),
					}

					if err := testrunner.RunTestCase(data, tc); err != nil {
						t.Fatalf("test case %q failed: %v", tc.Name, err)
					}
				})
			}
		})
	}
}

// findForgeBinary locates the forge binary. It checks:
// 1. FORGE_BINARY environment variable
// 2. ./build/bin/forge relative to project root
// 3. Builds from source into a temp directory
func findForgeBinary() (string, error) {
	// Check environment variable.
	if bin := os.Getenv("FORGE_BINARY"); bin != "" {
		if _, err := os.Stat(bin); err == nil {
			return bin, nil
		}
	}

	// Check local build directory.
	root, err := projectRoot()
	if err == nil {
		localBin := filepath.Join(root, "build", "bin", "forge")
		if _, err := os.Stat(localBin); err == nil {
			return localBin, nil
		}
	}

	// Build from source.
	if root != "" {
		tmpDir, err := os.MkdirTemp("", "forge-e2e-*")
		if err != nil {
			return "", fmt.Errorf("creating temp dir: %w", err)
		}
		binPath := filepath.Join(tmpDir, "forge")
		cmd := exec.Command("go", "build", "-o", binPath, "./cmd/forge")
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("building forge binary: %w\n%s", err, string(out))
		}
		return binPath, nil
	}

	return "", fmt.Errorf("forge binary not found: set FORGE_BINARY or build with 'go run ./cmd/forge build forge'")
}

// projectRoot walks up from the current test file directory to find go.mod.
func projectRoot() (string, error) {
	// Start from the working directory.
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goMod := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goMod); err == nil {
			// Verify this is the forge module.
			data, err := os.ReadFile(goMod)
			if err == nil && strings.Contains(string(data), "module github.com/alexandremahdhaoui/forge") {
				return dir, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found in any parent directory")
		}
		dir = parent
	}
}
