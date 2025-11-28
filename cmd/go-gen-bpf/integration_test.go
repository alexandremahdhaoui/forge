//go:build integration

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// checkClangBPFSupport verifies that clang is available and has BPF target support.
// Returns nil if clang with BPF support is available, otherwise returns an error.
func checkClangBPFSupport() error {
	// Check clang is available
	if _, err := exec.LookPath("clang"); err != nil {
		return err
	}

	// Verify clang can compile for BPF target
	cmd := exec.Command("clang", "-target", "bpf", "-c", "-x", "c", "-o", "/dev/null", "-")
	cmd.Stdin = strings.NewReader("int main() { return 0; }")
	return cmd.Run()
}

// TestBuildIntegration tests the full BPF build flow with real bpf2go execution.
func TestBuildIntegration(t *testing.T) {
	// Check for required tools - FAIL if not available (per project requirements)
	if err := checkClangBPFSupport(); err != nil {
		t.Fatalf("clang with BPF target support is required: %v", err)
	}

	// Create temp directory structure
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "minimal.c")
	destDir := filepath.Join(tmpDir, "generated")

	// Create minimal valid BPF C source file
	// This is the simplest possible BPF program that bpf2go can compile
	bpfContent := `// SPDX-License-Identifier: GPL-2.0
// Minimal BPF program for testing

// Minimal placeholder to satisfy bpf2go
char _license[] __attribute__((section("license"))) = "GPL";

// Empty program section
__attribute__((section("socket")))
int minimal() {
    return 0;
}
`
	if err := os.WriteFile(srcFile, []byte(bpfContent), 0o644); err != nil {
		t.Fatalf("Failed to write BPF source file: %v", err)
	}

	// Call build() with valid BuildInput
	input := mcptypes.BuildInput{
		Name: "test-bpf",
		Src:  srcFile,
		Dest: destDir,
		Spec: map[string]any{
			"ident": "minimal",
		},
	}

	artifact, err := build(context.Background(), input)
	if err != nil {
		t.Fatalf("build() failed: %v", err)
	}

	// Verify artifact has correct name
	if artifact.Name != "test-bpf" {
		t.Errorf("artifact.Name = %q, want %q", artifact.Name, "test-bpf")
	}

	// Verify artifact type is "bpf"
	if artifact.Type != "bpf" {
		t.Errorf("artifact.Type = %q, want %q", artifact.Type, "bpf")
	}

	// Verify artifact has DependencyDetectorEngine set
	if artifact.DependencyDetectorEngine != "go://go-gen-bpf" {
		t.Errorf("artifact.DependencyDetectorEngine = %q, want %q", artifact.DependencyDetectorEngine, "go://go-gen-bpf")
	}

	// Verify dependencies populated with .c file
	if len(artifact.Dependencies) == 0 {
		t.Error("artifact.Dependencies is empty, expected at least one dependency")
	} else {
		// Check that the source file is in dependencies
		foundSourceFile := false
		for _, dep := range artifact.Dependencies {
			if dep.Type != "file" {
				t.Errorf("dependency.Type = %q, want %q", dep.Type, "file")
			}
			if strings.HasSuffix(dep.FilePath, "minimal.c") {
				foundSourceFile = true
			}
			if dep.Timestamp == "" {
				t.Error("dependency.Timestamp is empty")
			}
		}
		if !foundSourceFile {
			t.Error("minimal.c not found in artifact.Dependencies")
		}
	}

	// Verify generated files exist (bpf2go generates _bpfel.go and _bpfeb.go)
	generatedGoFile := filepath.Join(destDir, "zz_generated_bpfel.go")
	if _, err := os.Stat(generatedGoFile); os.IsNotExist(err) {
		t.Errorf("Generated Go file not found at %s", generatedGoFile)
	}

	// Verify artifact location matches dest
	if artifact.Location != destDir {
		t.Errorf("artifact.Location = %q, want %q", artifact.Location, destDir)
	}

	// Verify artifact timestamp is set
	if artifact.Timestamp == "" {
		t.Error("artifact.Timestamp is empty")
	}
}

// TestBuildIntegrationMissingSrc tests that build fails for missing source file.
func TestBuildIntegrationMissingSrc(t *testing.T) {
	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, "generated")

	input := mcptypes.BuildInput{
		Name: "test-missing-src",
		Src:  "", // Missing source
		Dest: destDir,
		Spec: map[string]any{
			"ident": "test",
		},
	}

	_, err := build(context.Background(), input)
	if err == nil {
		t.Error("build() should fail when src is empty")
	}
	if !strings.Contains(err.Error(), "src is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestBuildIntegrationMissingDest tests that build fails for missing destination.
func TestBuildIntegrationMissingDest(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "test.c")
	if err := os.WriteFile(srcFile, []byte("// test"), 0o644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	input := mcptypes.BuildInput{
		Name: "test-missing-dest",
		Src:  srcFile,
		Dest: "", // Missing destination
		Spec: map[string]any{
			"ident": "test",
		},
	}

	_, err := build(context.Background(), input)
	if err == nil {
		t.Error("build() should fail when dest is empty")
	}
	if !strings.Contains(err.Error(), "dest is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestBuildIntegrationMissingIdent tests that build fails when ident is missing.
func TestBuildIntegrationMissingIdent(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "test.c")
	destDir := filepath.Join(tmpDir, "generated")

	if err := os.WriteFile(srcFile, []byte("// test"), 0o644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	input := mcptypes.BuildInput{
		Name: "test-missing-ident",
		Src:  srcFile,
		Dest: destDir,
		Spec: map[string]any{}, // Missing ident
	}

	_, err := build(context.Background(), input)
	if err == nil {
		t.Error("build() should fail when ident is missing")
	}
	if !strings.Contains(err.Error(), "ident") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestBuildIntegrationSourceNotFound tests that build fails for non-existent source.
func TestBuildIntegrationSourceNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, "generated")

	input := mcptypes.BuildInput{
		Name: "test-not-found",
		Src:  filepath.Join(tmpDir, "nonexistent.c"),
		Dest: destDir,
		Spec: map[string]any{
			"ident": "test",
		},
	}

	_, err := build(context.Background(), input)
	if err == nil {
		t.Error("build() should fail when source file doesn't exist")
	}
	if !strings.Contains(err.Error(), "source file not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestBuildIntegrationSourceIsDirectory tests that build fails when source is a directory.
func TestBuildIntegrationSourceIsDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "srcdir")
	destDir := filepath.Join(tmpDir, "generated")

	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	input := mcptypes.BuildInput{
		Name: "test-src-is-dir",
		Src:  srcDir, // Directory instead of file
		Dest: destDir,
		Spec: map[string]any{
			"ident": "test",
		},
	}

	_, err := build(context.Background(), input)
	if err == nil {
		t.Error("build() should fail when source is a directory")
	}
	if !strings.Contains(err.Error(), "must be a file, not directory") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestBuildIntegrationWithAllOptions tests build with all spec options configured.
func TestBuildIntegrationWithAllOptions(t *testing.T) {
	// Check for required tools - FAIL if not available (per project requirements)
	if err := checkClangBPFSupport(); err != nil {
		t.Fatalf("clang with BPF target support is required: %v", err)
	}

	// Create temp directory structure
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "full_options.c")
	destDir := filepath.Join(tmpDir, "output")

	// Create minimal valid BPF C source file
	bpfContent := `// SPDX-License-Identifier: GPL-2.0
char _license[] __attribute__((section("license"))) = "GPL";
__attribute__((section("socket")))
int full_options() { return 0; }
`
	if err := os.WriteFile(srcFile, []byte(bpfContent), 0o644); err != nil {
		t.Fatalf("Failed to write BPF source file: %v", err)
	}

	// Call build() with all options specified
	input := mcptypes.BuildInput{
		Name: "test-full-options",
		Src:  srcFile,
		Dest: destDir,
		Spec: map[string]any{
			"ident":         "fullopts",
			"bpf2goVersion": "latest",
			"goPackage":     "bpftest",
			"outputStem":    "gen_bpf",
			"tags":          []any{"linux"},
		},
	}

	artifact, err := build(context.Background(), input)
	if err != nil {
		t.Fatalf("build() failed: %v", err)
	}

	// Verify artifact
	if artifact.Name != "test-full-options" {
		t.Errorf("artifact.Name = %q, want %q", artifact.Name, "test-full-options")
	}

	// Verify generated files with custom output stem exist
	generatedGoFile := filepath.Join(destDir, "gen_bpf_bpfel.go")
	if _, err := os.Stat(generatedGoFile); os.IsNotExist(err) {
		t.Errorf("Generated Go file not found at %s", generatedGoFile)
	}
}
