//go:build integration

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
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// TestIsStandardLibrary tests the standard library detection logic.
func TestIsStandardLibrary(t *testing.T) {
	tests := []struct {
		name     string
		pkgPath  string
		expected bool
	}{
		{"fmt is stdlib", "fmt", true},
		{"encoding/json is stdlib", "encoding/json", true},
		{"net/http is stdlib", "net/http", true},
		{"github.com/foo/bar is NOT stdlib", "github.com/foo/bar", false},
		{"golang.org/x/mod is NOT stdlib", "golang.org/x/mod", false},
		{"C is NOT stdlib (cgo)", "C", false},
		{"internal is stdlib", "internal", true},
		{"vendor is stdlib", "vendor", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStandardLibrary(tt.pkgPath)
			if result != tt.expected {
				t.Errorf("isStandardLibrary(%q) = %v, want %v", tt.pkgPath, result, tt.expected)
			}
		})
	}
}

// TestExtractImports tests import extraction from AST.
func TestExtractImports(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	testCode := `package main

import (
	"fmt"
	"encoding/json"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

func main() {
	fmt.Println("hello")
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Parse the file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, testFile, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}

	// Extract imports
	imports := extractImports(file)

	expectedImports := []string{"fmt", "encoding/json", "github.com/alexandremahdhaoui/forge/pkg/mcptypes"}
	if len(imports) != len(expectedImports) {
		t.Errorf("Expected %d imports, got %d", len(expectedImports), len(imports))
	}

	for i, imp := range imports {
		if imp != expectedImports[i] {
			t.Errorf("Import %d: expected %q, got %q", i, expectedImports[i], imp)
		}
	}
}

// TestFindGoMod tests finding go.mod in parent directories.
func TestFindGoMod(t *testing.T) {
	// Use the real forge project structure
	// Start from this file and find go.mod
	thisFile, err := filepath.Abs("detect_test.go")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	goModPath, err := findGoMod(thisFile)
	if err != nil {
		t.Fatalf("Failed to find go.mod: %v", err)
	}

	// Verify it's actually a go.mod file
	if filepath.Base(goModPath) != "go.mod" {
		t.Errorf("Expected go.mod, got %s", filepath.Base(goModPath))
	}

	// Verify it exists
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		t.Errorf("go.mod not found at %s", goModPath)
	}
}

// TestGetFileTimestamp tests timestamp retrieval.
func TestGetFileTimestamp(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	err := os.WriteFile(testFile, []byte("package main"), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	timestamp, err := getFileTimestamp(testFile)
	if err != nil {
		t.Fatalf("Failed to get timestamp: %v", err)
	}

	// Verify timestamp format (RFC3339)
	_, err = time.Parse(time.RFC3339, timestamp)
	if err != nil {
		t.Errorf("Invalid timestamp format %q: %v", timestamp, err)
	}

	// Verify timestamp is in UTC
	if !strings.HasSuffix(timestamp, "Z") {
		t.Errorf("Timestamp should be in UTC (ending with Z), got %q", timestamp)
	}
}

// TestDetectDependencies_NoImports tests a function with no imports.
func TestDetectDependencies_NoImports(t *testing.T) {
	// Create a temporary project structure
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module example.com/test

go 1.23
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Create test file with no imports
	testFile := filepath.Join(tmpDir, "main.go")
	testCode := `package main

func main() {
	x := 42
	_ = x
}
`
	err = os.WriteFile(testFile, []byte(testCode), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test dependency detection
	input := mcptypes.DetectDependenciesInput{
		FilePath: testFile,
		FuncName: "main",
	}

	output, err := DetectDependencies(input)
	if err != nil {
		t.Fatalf("DetectDependencies failed: %v", err)
	}

	// Should have 1 dependency (go.mod)
	if len(output.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency (go.mod), got %d", len(output.Dependencies))
	}

	// Verify it's go.mod
	if len(output.Dependencies) > 0 {
		dep := output.Dependencies[0]
		if dep.Type != "file" {
			t.Errorf("Expected type 'file' for go.mod, got %q", dep.Type)
		}
		if filepath.Base(dep.FilePath) != "go.mod" {
			t.Errorf("Expected go.mod, got %s", filepath.Base(dep.FilePath))
		}
	}
}

// TestDetectDependencies_StdlibOnly tests a function with only stdlib imports.
func TestDetectDependencies_StdlibOnly(t *testing.T) {
	// Create a temporary project structure
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module example.com/test

go 1.23
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Create test file with stdlib imports only
	testFile := filepath.Join(tmpDir, "main.go")
	testCode := `package main

import (
	"fmt"
	"encoding/json"
)

func main() {
	fmt.Println("hello")
	json.Marshal(nil)
}
`
	err = os.WriteFile(testFile, []byte(testCode), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test dependency detection
	input := mcptypes.DetectDependenciesInput{
		FilePath: testFile,
		FuncName: "main",
	}

	output, err := DetectDependencies(input)
	if err != nil {
		t.Fatalf("DetectDependencies failed: %v", err)
	}

	// Should have 1 dependency (go.mod, stdlib is excluded)
	if len(output.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency (go.mod, stdlib excluded), got %d", len(output.Dependencies))
	}

	// Verify it's go.mod
	if len(output.Dependencies) > 0 {
		dep := output.Dependencies[0]
		if dep.Type != "file" {
			t.Errorf("Expected type 'file' for go.mod, got %q", dep.Type)
		}
		if filepath.Base(dep.FilePath) != "go.mod" {
			t.Errorf("Expected go.mod, got %s", filepath.Base(dep.FilePath))
		}
	}
}

// TestDetectDependencies_ExternalPackage tests a function with external imports.
func TestDetectDependencies_ExternalPackage(t *testing.T) {
	// Create a temporary project structure
	tmpDir := t.TempDir()

	// Create go.mod with external dependency
	goModContent := `module example.com/test

go 1.23

require (
	github.com/stretchr/testify v1.8.4
	golang.org/x/mod v0.30.0
)
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Create test file with external imports
	testFile := filepath.Join(tmpDir, "main.go")
	testCode := `package main

import (
	"github.com/stretchr/testify/assert"
	"golang.org/x/mod/modfile"
)

func main() {
	assert.NotNil(nil, "test")
	modfile.Parse("", nil, nil)
}
`
	err = os.WriteFile(testFile, []byte(testCode), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test dependency detection
	input := mcptypes.DetectDependenciesInput{
		FilePath: testFile,
		FuncName: "main",
	}

	output, err := DetectDependencies(input)
	if err != nil {
		t.Fatalf("DetectDependencies failed: %v", err)
	}

	// Should have 3 dependencies: go.mod + 2 external packages
	if len(output.Dependencies) != 3 {
		t.Errorf("Expected 3 dependencies (go.mod + 2 external), got %d", len(output.Dependencies))
	}

	// First dependency should be go.mod
	if len(output.Dependencies) > 0 {
		dep := output.Dependencies[0]
		if dep.Type != "file" {
			t.Errorf("Expected type 'file' for go.mod, got %q", dep.Type)
		}
		if filepath.Base(dep.FilePath) != "go.mod" {
			t.Errorf("Expected go.mod, got %s", filepath.Base(dep.FilePath))
		}
	}

	// Verify external package dependencies (skip go.mod at index 0)
	for i := 1; i < len(output.Dependencies); i++ {
		dep := output.Dependencies[i]
		if dep.Type != "externalPackage" {
			t.Errorf("Expected type 'externalPackage', got %q", dep.Type)
		}

		if dep.ExternalPackage == "github.com/stretchr/testify/assert" {
			// This is a subpackage, should match parent version
			if dep.Semver != "v1.8.4" {
				t.Errorf("Expected version v1.8.4 for testify, got %s", dep.Semver)
			}
		} else if dep.ExternalPackage == "golang.org/x/mod/modfile" {
			// This is a subpackage, should match parent version
			if dep.Semver != "v0.30.0" {
				t.Errorf("Expected version v0.30.0 for x/mod, got %s", dep.Semver)
			}
		} else {
			t.Errorf("Unexpected package: %s", dep.ExternalPackage)
		}
	}
}

// TestDetectDependencies_LocalPackage tests a function with local package imports.
func TestDetectDependencies_LocalPackage(t *testing.T) {
	// Create a temporary project structure
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module example.com/test

go 1.23
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Create a local package
	pkgDir := filepath.Join(tmpDir, "pkg", "helper")
	err = os.MkdirAll(pkgDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create pkg dir: %v", err)
	}

	helperFile := filepath.Join(pkgDir, "helper.go")
	helperCode := `package helper

func Helper() string {
	return "helper"
}
`
	err = os.WriteFile(helperFile, []byte(helperCode), 0o644)
	if err != nil {
		t.Fatalf("Failed to write helper file: %v", err)
	}

	// Create main file that imports local package
	testFile := filepath.Join(tmpDir, "main.go")
	testCode := `package main

import (
	"example.com/test/pkg/helper"
)

func main() {
	helper.Helper()
}
`
	err = os.WriteFile(testFile, []byte(testCode), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test dependency detection
	input := mcptypes.DetectDependenciesInput{
		FilePath: testFile,
		FuncName: "main",
	}

	output, err := DetectDependencies(input)
	if err != nil {
		t.Fatalf("DetectDependencies failed: %v", err)
	}

	// Should have 2 file dependencies: go.mod + helper.go
	if len(output.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies (go.mod + helper.go), got %d", len(output.Dependencies))
	}

	// First dependency should be go.mod
	if len(output.Dependencies) > 0 {
		dep := output.Dependencies[0]
		if dep.Type != "file" {
			t.Errorf("Expected type 'file' for go.mod, got %q", dep.Type)
		}
		if filepath.Base(dep.FilePath) != "go.mod" {
			t.Errorf("Expected go.mod, got %s", filepath.Base(dep.FilePath))
		}
	}

	// Second dependency should be helper.go
	if len(output.Dependencies) > 1 {
		dep := output.Dependencies[1]
		if dep.Type != "file" {
			t.Errorf("Expected type 'file', got %q", dep.Type)
		}

		// Verify file path is absolute
		if !filepath.IsAbs(dep.FilePath) {
			t.Errorf("Expected absolute path, got %q", dep.FilePath)
		}

		// Verify file path points to helper.go
		if filepath.Base(dep.FilePath) != "helper.go" {
			t.Errorf("Expected helper.go, got %s", filepath.Base(dep.FilePath))
		}

		// Verify timestamp format
		_, err = time.Parse(time.RFC3339, dep.Timestamp)
		if err != nil {
			t.Errorf("Invalid timestamp format: %v", err)
		}
	}
}

// TestDetectDependencies_TransitiveDependencies tests transitive dependencies.
func TestDetectDependencies_TransitiveDependencies(t *testing.T) {
	// Create a temporary project structure: A -> B -> C
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module example.com/test

go 1.23
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Create package C
	pkgCDir := filepath.Join(tmpDir, "pkg", "c")
	err = os.MkdirAll(pkgCDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create pkg/c dir: %v", err)
	}

	cFile := filepath.Join(pkgCDir, "c.go")
	cCode := `package c

func C() string {
	return "c"
}
`
	err = os.WriteFile(cFile, []byte(cCode), 0o644)
	if err != nil {
		t.Fatalf("Failed to write c.go: %v", err)
	}

	// Create package B (imports C)
	pkgBDir := filepath.Join(tmpDir, "pkg", "b")
	err = os.MkdirAll(pkgBDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create pkg/b dir: %v", err)
	}

	bFile := filepath.Join(pkgBDir, "b.go")
	bCode := `package b

import "example.com/test/pkg/c"

func B() string {
	return c.C()
}
`
	err = os.WriteFile(bFile, []byte(bCode), 0o644)
	if err != nil {
		t.Fatalf("Failed to write b.go: %v", err)
	}

	// Create main file (imports B)
	testFile := filepath.Join(tmpDir, "main.go")
	testCode := `package main

import "example.com/test/pkg/b"

func main() {
	b.B()
}
`
	err = os.WriteFile(testFile, []byte(testCode), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test dependency detection
	input := mcptypes.DetectDependenciesInput{
		FilePath: testFile,
		FuncName: "main",
	}

	output, err := DetectDependencies(input)
	if err != nil {
		t.Fatalf("DetectDependencies failed: %v", err)
	}

	// Should have 3 file dependencies: go.mod + b.go + c.go
	if len(output.Dependencies) != 3 {
		t.Errorf("Expected 3 dependencies (go.mod + transitive), got %d", len(output.Dependencies))
	}

	// First dependency should be go.mod
	if len(output.Dependencies) > 0 {
		dep := output.Dependencies[0]
		if dep.Type != "file" {
			t.Errorf("Expected type 'file' for go.mod, got %q", dep.Type)
		}
		if filepath.Base(dep.FilePath) != "go.mod" {
			t.Errorf("Expected go.mod, got %s", filepath.Base(dep.FilePath))
		}
	}

	// Verify both b.go and c.go are present (skip go.mod at index 0)
	foundB := false
	foundC := false
	for i := 1; i < len(output.Dependencies); i++ {
		dep := output.Dependencies[i]
		if dep.Type != "file" {
			t.Errorf("Expected type 'file', got %q", dep.Type)
		}
		baseName := filepath.Base(dep.FilePath)
		if baseName == "b.go" {
			foundB = true
		} else if baseName == "c.go" {
			foundC = true
		}
	}

	if !foundB {
		t.Error("Expected b.go in dependencies")
	}
	if !foundC {
		t.Error("Expected c.go in dependencies (transitive)")
	}
}

// TestDetectDependencies_CircularDependency tests circular dependency handling.
func TestDetectDependencies_CircularDependency(t *testing.T) {
	// Create a temporary project structure: A -> B -> A
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module example.com/test

go 1.23
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Create package A (imports B)
	pkgADir := filepath.Join(tmpDir, "pkg", "a")
	err = os.MkdirAll(pkgADir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create pkg/a dir: %v", err)
	}

	aFile := filepath.Join(pkgADir, "a.go")
	aCode := `package a

import "example.com/test/pkg/b"

func A() string {
	return b.B()
}
`
	err = os.WriteFile(aFile, []byte(aCode), 0o644)
	if err != nil {
		t.Fatalf("Failed to write a.go: %v", err)
	}

	// Create package B (imports A - circular!)
	pkgBDir := filepath.Join(tmpDir, "pkg", "b")
	err = os.MkdirAll(pkgBDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create pkg/b dir: %v", err)
	}

	bFile := filepath.Join(pkgBDir, "b.go")
	bCode := `package b

import "example.com/test/pkg/a"

func B() string {
	return a.A()
}
`
	err = os.WriteFile(bFile, []byte(bCode), 0o644)
	if err != nil {
		t.Fatalf("Failed to write b.go: %v", err)
	}

	// Test dependency detection starting from A
	input := mcptypes.DetectDependenciesInput{
		FilePath: aFile,
		FuncName: "A",
	}

	// Should NOT hang or panic - memoization prevents infinite loop
	output, err := DetectDependencies(input)
	if err != nil {
		t.Fatalf("DetectDependencies failed: %v", err)
	}

	// Should have 2 file dependencies: go.mod + b.go
	// When we start from A, we mark A as visited, then process B
	// B tries to import A, but A is already visited, so we skip it
	// Therefore, we get go.mod + B in dependencies (A is the starting point, not a dependency)
	if len(output.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies (go.mod + circular handled), got %d", len(output.Dependencies))
		for i, dep := range output.Dependencies {
			t.Logf("Dependency %d: %s", i, filepath.Base(dep.FilePath))
		}
	}

	// First dependency should be go.mod
	if len(output.Dependencies) > 0 {
		dep := output.Dependencies[0]
		if dep.Type != "file" {
			t.Errorf("Expected type 'file' for go.mod, got %q", dep.Type)
		}
		if filepath.Base(dep.FilePath) != "go.mod" {
			t.Errorf("Expected go.mod, got %s", filepath.Base(dep.FilePath))
		}
	}

	// Second dependency should be b.go
	if len(output.Dependencies) > 1 {
		dep := output.Dependencies[1]
		if filepath.Base(dep.FilePath) != "b.go" {
			t.Errorf("Expected b.go, got %s", filepath.Base(dep.FilePath))
		}
	}

	// Verify no duplicates
	seen := make(map[string]bool)
	for _, dep := range output.Dependencies {
		if seen[dep.FilePath] {
			t.Errorf("Duplicate dependency: %s", dep.FilePath)
		}
		seen[dep.FilePath] = true
	}
}

// TestDetectDependencies_FunctionNotFound tests error when function doesn't exist.
func TestDetectDependencies_FunctionNotFound(t *testing.T) {
	// Create a temporary project structure
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module example.com/test

go 1.23
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tmpDir, "main.go")
	testCode := `package main

func main() {
}
`
	err = os.WriteFile(testFile, []byte(testCode), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test dependency detection with non-existent function
	input := mcptypes.DetectDependenciesInput{
		FilePath: testFile,
		FuncName: "nonExistent",
	}

	_, err = DetectDependencies(input)
	if err == nil {
		t.Fatal("Expected error for non-existent function, got nil")
	}

	if !strings.Contains(err.Error(), "function nonExistent not found") {
		t.Errorf("Expected 'function nonExistent not found' error, got: %v", err)
	}
}

// TestDetectDependencies_NonExistentFile tests error when file doesn't exist.
func TestDetectDependencies_NonExistentFile(t *testing.T) {
	input := mcptypes.DetectDependenciesInput{
		FilePath: "/this/path/does/not/exist/file.go",
		FuncName: "main",
	}

	_, err := DetectDependencies(input)
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}

	// Verify error message mentions file not found
	if !strings.Contains(err.Error(), "no such file") &&
		!strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected file not found error, got: %v", err)
	}
}
