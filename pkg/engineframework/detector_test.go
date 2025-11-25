//go:build unit

package engineframework

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFindDetector_InPATH(t *testing.T) {
	// "ls" should be in PATH on all Unix systems
	path, err := FindDetector("ls")
	if err != nil {
		t.Fatalf("FindDetector('ls') returned error: %v", err)
	}
	if path == "" {
		t.Fatal("FindDetector('ls') returned empty path")
	}

	// Verify it's a valid path
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("FindDetector returned path that doesn't exist: %s", path)
	}
}

func TestFindDetector_InBuildBin(t *testing.T) {
	// Create a temporary build/bin directory with a fake detector
	tmpDir := t.TempDir()
	buildBinDir := filepath.Join(tmpDir, "build", "bin")
	if err := os.MkdirAll(buildBinDir, 0o755); err != nil {
		t.Fatalf("failed to create build/bin: %v", err)
	}

	// Create a fake detector binary
	detectorName := "test-detector"
	detectorPath := filepath.Join(buildBinDir, detectorName)
	if err := os.WriteFile(detectorPath, []byte("#!/bin/sh\necho test"), 0o755); err != nil {
		t.Fatalf("failed to create fake detector: %v", err)
	}

	// Change to temp directory so ./build/bin is relative to it
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	// FindDetector should find it
	path, err := FindDetector(detectorName)
	if err != nil {
		t.Fatalf("FindDetector(%q) returned error: %v", detectorName, err)
	}

	// Should be an absolute path
	if !filepath.IsAbs(path) {
		t.Errorf("FindDetector returned non-absolute path: %s", path)
	}

	// Should point to our fake detector
	expectedPath := filepath.Join(tmpDir, "build", "bin", detectorName)
	if path != expectedPath {
		t.Errorf("FindDetector returned %q, expected %q", path, expectedPath)
	}
}

func TestFindDetector_NotFound(t *testing.T) {
	// Use a name that definitely doesn't exist
	nonExistentDetector := "this-detector-definitely-does-not-exist-123456789"

	// Ensure it's not in PATH
	if _, err := exec.LookPath(nonExistentDetector); err == nil {
		t.Skipf("Unexpectedly found %q in PATH", nonExistentDetector)
	}

	// FindDetector should return an error
	path, err := FindDetector(nonExistentDetector)
	if err == nil {
		t.Fatalf("FindDetector(%q) should have returned error, got path: %s", nonExistentDetector, path)
	}

	// Error message should mention PATH and ./build/bin
	errMsg := err.Error()
	if errMsg != nonExistentDetector+" not found in PATH or ./build/bin" {
		t.Errorf("FindDetector error message = %q, want %q", errMsg, nonExistentDetector+" not found in PATH or ./build/bin")
	}
}

func TestFindDetector_PATHTakesPrecedence(t *testing.T) {
	// This test verifies that PATH is checked before ./build/bin
	// We use "ls" which is in PATH, and verify FindDetector finds it
	// even if we're in a directory with a ./build/bin/ls

	tmpDir := t.TempDir()
	buildBinDir := filepath.Join(tmpDir, "build", "bin")
	if err := os.MkdirAll(buildBinDir, 0o755); err != nil {
		t.Fatalf("failed to create build/bin: %v", err)
	}

	// Create a fake "ls" in build/bin
	fakeLs := filepath.Join(buildBinDir, "ls")
	if err := os.WriteFile(fakeLs, []byte("#!/bin/sh\necho fake"), 0o755); err != nil {
		t.Fatalf("failed to create fake ls: %v", err)
	}

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	// FindDetector should find the real "ls" from PATH, not our fake one
	path, err := FindDetector("ls")
	if err != nil {
		t.Fatalf("FindDetector('ls') returned error: %v", err)
	}

	// Should NOT be our fake ls
	absExpected, _ := filepath.Abs(fakeLs)
	if path == absExpected {
		t.Errorf("FindDetector should prefer PATH over ./build/bin, but returned ./build/bin/ls")
	}
}

// Note: CallDetector tests would require a real MCP server or mocking.
// The following tests document the expected behavior but are skipped
// because they require integration test infrastructure.

func TestCallDetector_Integration(t *testing.T) {
	// This is an integration test that requires the go-dependency-detector
	// to be built and available. Skip in unit tests.
	t.Skip("Integration test - requires go-dependency-detector binary")

	// If this test is enabled, it would:
	// 1. Find go-dependency-detector
	// 2. Call it with a test file
	// 3. Verify dependencies are returned
}
