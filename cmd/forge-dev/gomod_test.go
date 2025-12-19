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
)

func TestFindGoMod(t *testing.T) {
	t.Run("go.mod in current directory", func(t *testing.T) {
		// Create temp directory with go.mod
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte("module example.com/test\n"), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		result, err := FindGoMod(tmpDir)
		if err != nil {
			t.Fatalf("FindGoMod failed: %v", err)
		}
		if result != tmpDir {
			t.Errorf("FindGoMod = %q, want %q", result, tmpDir)
		}
	})

	t.Run("go.mod in parent directory", func(t *testing.T) {
		// Create temp directory structure: tmpDir/go.mod, tmpDir/sub/
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte("module example.com/test\n"), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		subDir := filepath.Join(tmpDir, "sub")
		if err := os.Mkdir(subDir, 0o755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}

		result, err := FindGoMod(subDir)
		if err != nil {
			t.Fatalf("FindGoMod failed: %v", err)
		}
		if result != tmpDir {
			t.Errorf("FindGoMod = %q, want %q", result, tmpDir)
		}
	})

	t.Run("go.mod in grandparent directory", func(t *testing.T) {
		// Create temp directory structure: tmpDir/go.mod, tmpDir/a/b/
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte("module example.com/test\n"), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		deepDir := filepath.Join(tmpDir, "a", "b")
		if err := os.MkdirAll(deepDir, 0o755); err != nil {
			t.Fatalf("failed to create deep directory: %v", err)
		}

		result, err := FindGoMod(deepDir)
		if err != nil {
			t.Fatalf("FindGoMod failed: %v", err)
		}
		if result != tmpDir {
			t.Errorf("FindGoMod = %q, want %q", result, tmpDir)
		}
	})

	t.Run("no go.mod found", func(t *testing.T) {
		// Create temp directory without go.mod
		tmpDir := t.TempDir()

		_, err := FindGoMod(tmpDir)
		if err == nil {
			t.Error("FindGoMod should return error when go.mod not found")
		}
		if !strings.Contains(err.Error(), "cannot find go.mod") {
			t.Errorf("error message should mention cannot find go.mod, got: %v", err)
		}
	})
}

func TestParseModulePath(t *testing.T) {
	t.Run("simple module path", func(t *testing.T) {
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")
		content := "module example.com/test\n"
		if err := os.WriteFile(goModPath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		result, err := ParseModulePath(goModPath)
		if err != nil {
			t.Fatalf("ParseModulePath failed: %v", err)
		}
		if result != "example.com/test" {
			t.Errorf("ParseModulePath = %q, want %q", result, "example.com/test")
		}
	})

	t.Run("multi-level module path", func(t *testing.T) {
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")
		content := "module github.com/org/project/sub\n"
		if err := os.WriteFile(goModPath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		result, err := ParseModulePath(goModPath)
		if err != nil {
			t.Fatalf("ParseModulePath failed: %v", err)
		}
		if result != "github.com/org/project/sub" {
			t.Errorf("ParseModulePath = %q, want %q", result, "github.com/org/project/sub")
		}
	})

	t.Run("module path with trailing whitespace", func(t *testing.T) {
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")
		content := "module example.com/test   \n"
		if err := os.WriteFile(goModPath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		result, err := ParseModulePath(goModPath)
		if err != nil {
			t.Fatalf("ParseModulePath failed: %v", err)
		}
		if result != "example.com/test" {
			t.Errorf("ParseModulePath = %q, want %q", result, "example.com/test")
		}
	})

	t.Run("module path with comment", func(t *testing.T) {
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")
		content := "module example.com/test // some comment\n"
		if err := os.WriteFile(goModPath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		result, err := ParseModulePath(goModPath)
		if err != nil {
			t.Fatalf("ParseModulePath failed: %v", err)
		}
		if result != "example.com/test" {
			t.Errorf("ParseModulePath = %q, want %q", result, "example.com/test")
		}
	})

	t.Run("module path without trailing newline", func(t *testing.T) {
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")
		content := "module example.com/test"
		if err := os.WriteFile(goModPath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		result, err := ParseModulePath(goModPath)
		if err != nil {
			t.Fatalf("ParseModulePath failed: %v", err)
		}
		if result != "example.com/test" {
			t.Errorf("ParseModulePath = %q, want %q", result, "example.com/test")
		}
	})

	t.Run("no module line", func(t *testing.T) {
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")
		content := "go 1.21\n"
		if err := os.WriteFile(goModPath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		_, err := ParseModulePath(goModPath)
		if err == nil {
			t.Error("ParseModulePath should return error when no module line found")
		}
		if !strings.Contains(err.Error(), "no module line found") {
			t.Errorf("error message should mention no module line found, got: %v", err)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := ParseModulePath("/nonexistent/go.mod")
		if err == nil {
			t.Error("ParseModulePath should return error for nonexistent file")
		}
	})
}

func TestResolveSpecTypesContext(t *testing.T) {
	t.Run("nil config returns nil", func(t *testing.T) {
		result, err := ResolveSpecTypesContext("/some/dir", nil)
		if err != nil {
			t.Fatalf("ResolveSpecTypesContext failed: %v", err)
		}
		if result != nil {
			t.Error("ResolveSpecTypesContext should return nil for nil config")
		}
	})

	t.Run("disabled config returns nil", func(t *testing.T) {
		config := &SpecTypesConfig{
			Enabled:     false,
			OutputPath:  "pkg/api/v1",
			PackageName: "v1",
		}

		result, err := ResolveSpecTypesContext("/some/dir", config)
		if err != nil {
			t.Fatalf("ResolveSpecTypesContext failed: %v", err)
		}
		if result != nil {
			t.Error("ResolveSpecTypesContext should return nil for disabled config")
		}
	})

	t.Run("computes correct paths", func(t *testing.T) {
		// Create temp directory structure: tmpDir/go.mod, tmpDir/cmd/engine/
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte("module github.com/user/project\n"), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		srcDir := filepath.Join(tmpDir, "cmd", "engine")
		if err := os.MkdirAll(srcDir, 0o755); err != nil {
			t.Fatalf("failed to create srcDir: %v", err)
		}

		config := &SpecTypesConfig{
			Enabled:     true,
			OutputPath:  "pkg/api/v1",
			PackageName: "v1",
		}

		result, err := ResolveSpecTypesContext(srcDir, config)
		if err != nil {
			t.Fatalf("ResolveSpecTypesContext failed: %v", err)
		}
		if result == nil {
			t.Fatal("ResolveSpecTypesContext should return non-nil for enabled config")
		}

		// Check ImportPath
		expectedImportPath := "github.com/user/project/pkg/api/v1"
		if result.ImportPath != expectedImportPath {
			t.Errorf("ImportPath = %q, want %q", result.ImportPath, expectedImportPath)
		}

		// Check PackageName
		if result.PackageName != "v1" {
			t.Errorf("PackageName = %q, want %q", result.PackageName, "v1")
		}

		// Check Prefix
		if result.Prefix != "v1." {
			t.Errorf("Prefix = %q, want %q", result.Prefix, "v1.")
		}

		// Check OutputDir
		expectedOutputDir := filepath.Join(tmpDir, "pkg", "api", "v1")
		if result.OutputDir != expectedOutputDir {
			t.Errorf("OutputDir = %q, want %q", result.OutputDir, expectedOutputDir)
		}
	})

	t.Run("error when go.mod not found", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := &SpecTypesConfig{
			Enabled:     true,
			OutputPath:  "pkg/api/v1",
			PackageName: "v1",
		}

		_, err := ResolveSpecTypesContext(tmpDir, config)
		if err == nil {
			t.Error("ResolveSpecTypesContext should return error when go.mod not found")
		}
	})

	t.Run("nested output path", func(t *testing.T) {
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte("module github.com/user/project\n"), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		config := &SpecTypesConfig{
			Enabled:     true,
			OutputPath:  "pkg/api/v1/alpha",
			PackageName: "alpha",
		}

		result, err := ResolveSpecTypesContext(tmpDir, config)
		if err != nil {
			t.Fatalf("ResolveSpecTypesContext failed: %v", err)
		}

		expectedImportPath := "github.com/user/project/pkg/api/v1/alpha"
		if result.ImportPath != expectedImportPath {
			t.Errorf("ImportPath = %q, want %q", result.ImportPath, expectedImportPath)
		}

		expectedOutputDir := filepath.Join(tmpDir, "pkg", "api", "v1", "alpha")
		if result.OutputDir != expectedOutputDir {
			t.Errorf("OutputDir = %q, want %q", result.OutputDir, expectedOutputDir)
		}
	})
}
