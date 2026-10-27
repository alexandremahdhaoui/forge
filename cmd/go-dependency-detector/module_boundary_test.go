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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

func writeBoundaryFixture(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}

	return path
}

func TestDetectDependencies_AnImportFromASiblingModuleSharingTheModulePrefixIsCheckedAgainstGoMod(t *testing.T) {
	tmpDir := t.TempDir()
	writeBoundaryFixture(t, tmpDir, "go.mod", "module example.com/test\n\ngo 1.23\n")
	mainFile := writeBoundaryFixture(t, tmpDir, "main.go", "package main\n\nimport \"example.com/test-spec/pkg\"\n\nfunc main() { pkg.Run() }\n")

	_, err := DetectDependencies(mcptypes.DetectDependenciesInput{FilePath: mainFile, FuncName: "main"})
	if err == nil {
		t.Fatal("example.com/test-spec/pkg was walked as a package of example.com/test instead of refused as a module go.mod does not require")
	}
	if !strings.Contains(err.Error(), "package example.com/test-spec/pkg not found in go.mod") {
		t.Fatalf("expected the refusal to name example.com/test-spec/pkg, got %v", err)
	}
}

func TestDetectDependencies_AReplaceOntoASiblingModuleSharingTheModulePrefixIsNotWalkedAsLocal(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	writeBoundaryFixture(t, tmpDir, "go.mod", "module example.com/test\n\ngo 1.23\n\nrequire example.com/other v1.0.0\n\nreplace example.com/other => example.com/test-spec v1.0.0\n")
	mainFile := writeBoundaryFixture(t, tmpDir, "main.go", "package main\n\nimport \"example.com/other\"\n\nfunc main() { other.Run() }\n")
	writeBoundaryFixture(t, tmpDir, filepath.Join("example.com", "test-spec", "spec.go"), "package other\n\nfunc Run() {}\n")

	output, err := DetectDependencies(mcptypes.DetectDependenciesInput{FilePath: mainFile, FuncName: "main"})
	if err != nil {
		t.Fatalf("DetectDependencies failed: %v", err)
	}

	for _, dep := range output.Dependencies {
		if filepath.Base(dep.Path) == "spec.go" {
			t.Fatalf("the replacement module example.com/test-spec was read as a directory of example.com/test: %+v", output.Dependencies)
		}
	}
}
