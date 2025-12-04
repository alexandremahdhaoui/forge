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

// Package integration contains end-to-end integration tests for forge components.
package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

// TestLazyRebuildMocks tests the lazy rebuild workflow for mock generation.
// This test verifies that the lazy rebuild logic works correctly with dependencies
// that match what the go-gen-mocks-dep-detector would produce.
//
// The test runs within the forge repository and uses an existing build target
// (go-lint) as a proxy for testing the lazy rebuild logic with mock-like dependencies.
// This avoids the complexity of setting up a separate project with mockery.
//
// Test scenario:
// 1. Build an artifact (first build)
// 2. Add mock-style dependencies (.mockery.yaml, go.mod, interface files)
// 3. Build again (no changes) -> SKIPPED
// 4. Touch interface file dependency
// 5. Build again -> REBUILT (dependency changed)
// 6. Update dependencies, build again -> SKIPPED
func TestLazyRebuildMocks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Change to repository root (same pattern as build_lazy_integration_test.go)
	repoRoot := "../.."
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Failed to change to repo root: %v", err)
	}

	// Build forge binary
	forgeBin := "./build/bin/forge"
	cmd := exec.Command("go", "build", "-o", forgeBin, "./cmd/forge")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build forge: %v", err)
	}

	// Use a dedicated artifact store path for this test to avoid interference
	artifactStorePath := ".forge/test-lazy-mocks-artifact-store.yaml"
	defer func() { _ = os.Remove(artifactStorePath) }()
	_ = os.Remove(artifactStorePath)

	// Create a temporary forge.yaml for this test
	testForgeYaml := `name: lazy-mocks-test
artifactStorePath: ` + artifactStorePath + `

build:
  - name: test-mocks-artifact
    src: ./cmd/go-lint
    dest: ./build/bin
    engine: go://go-build
`
	testForgeYamlPath := "forge-test-lazy-mocks.yaml"
	if err := os.WriteFile(testForgeYamlPath, []byte(testForgeYaml), 0o644); err != nil {
		t.Fatalf("Failed to write test forge.yaml: %v", err)
	}
	defer func() { _ = os.Remove(testForgeYamlPath) }()

	// Step 1: First build
	t.Log("Step 1: First build")
	cmd = exec.Command(forgeBin, "--config", testForgeYamlPath, "build", "test-mocks-artifact")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\nOutput: %s", err, string(output))
	}
	outputStr := string(output)
	if !strings.Contains(outputStr, "Building test-mocks-artifact") {
		t.Errorf("Expected 'Building' message, got: %s", outputStr)
	}

	// Step 2: Add mock-style dependencies
	t.Log("Step 2: Adding mock-style dependencies to artifact store")
	store, err := forge.ReadArtifactStore(artifactStorePath)
	if err != nil {
		t.Fatalf("Failed to read artifact store: %v", err)
	}

	artifact, err := forge.GetLatestArtifact(store, "test-mocks-artifact")
	if err != nil {
		t.Fatalf("Artifact not found in store: %v", err)
	}

	// Simulate mock-style dependencies:
	// - .mockery.yaml (we'll use go.mod as proxy since it exists)
	// - go.mod
	// - interface files (we'll use cmd/go-lint/main.go as proxy)
	mockeryConfigProxy := "go.mod" // Using go.mod as .mockery.yaml proxy
	interfaceFileProxy := filepath.Join("cmd", "go-lint", "main.go")

	mockeryInfo, err := os.Stat(mockeryConfigProxy)
	if err != nil {
		t.Fatalf("Failed to stat %s: %v", mockeryConfigProxy, err)
	}
	absMockeryConfig, _ := filepath.Abs(mockeryConfigProxy)

	interfaceInfo, err := os.Stat(interfaceFileProxy)
	if err != nil {
		t.Fatalf("Failed to stat %s: %v", interfaceFileProxy, err)
	}
	absInterfaceFile, _ := filepath.Abs(interfaceFileProxy)

	// Add dependencies simulating what go-gen-mocks-dep-detector would produce
	artifact.Dependencies = []forge.ArtifactDependency{
		{
			Type:      forge.DependencyTypeFile,
			FilePath:  absMockeryConfig,
			Timestamp: mockeryInfo.ModTime().UTC().Format(time.RFC3339),
		},
		{
			Type:      forge.DependencyTypeFile,
			FilePath:  absInterfaceFile,
			Timestamp: interfaceInfo.ModTime().UTC().Format(time.RFC3339),
		},
	}
	artifact.DependencyDetectorEngine = "go://go-gen-mocks-dep-detector"
	forge.AddOrUpdateArtifact(&store, artifact)

	if err := forge.WriteArtifactStore(artifactStorePath, store); err != nil {
		t.Fatalf("Failed to write artifact store: %v", err)
	}
	t.Logf("Dependencies tracked: %d", len(artifact.Dependencies))

	// Step 3: Second build - should skip
	t.Log("Step 3: Second build - should skip (unchanged)")
	cmd = exec.Command(forgeBin, "--config", testForgeYamlPath, "build", "test-mocks-artifact")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\nOutput: %s", err, string(output))
	}
	outputStr = string(output)
	if !strings.Contains(outputStr, "Skipping test-mocks-artifact") {
		t.Errorf("Expected 'Skipping' message, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "unchanged") {
		t.Errorf("Expected 'unchanged' in output, got: %s", outputStr)
	}

	// Step 4: Touch interface file dependency
	t.Log("Step 4: Touch interface file dependency")
	time.Sleep(time.Second) // Ensure timestamp changes
	now := time.Now()
	if err := os.Chtimes(absInterfaceFile, now, now); err != nil {
		t.Fatalf("Failed to touch file: %v", err)
	}

	// Step 5: Third build - should rebuild
	t.Log("Step 5: Third build - should rebuild (dependency changed)")
	cmd = exec.Command(forgeBin, "--config", testForgeYamlPath, "build", "test-mocks-artifact")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\nOutput: %s", err, string(output))
	}
	outputStr = string(output)
	if !strings.Contains(outputStr, "Building test-mocks-artifact") {
		t.Errorf("Expected 'Building' message after dependency change, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "dependency") || !strings.Contains(outputStr, "modified") {
		t.Errorf("Expected 'dependency ... modified' in output, got: %s", outputStr)
	}

	// Step 6: Update dependencies after rebuild, then build again - should skip
	t.Log("Step 6: Update dependencies and build again - should skip")
	store, err = forge.ReadArtifactStore(artifactStorePath)
	if err != nil {
		t.Fatalf("Failed to read artifact store: %v", err)
	}
	artifact, err = forge.GetLatestArtifact(store, "test-mocks-artifact")
	if err != nil {
		t.Fatalf("Artifact not found after rebuild: %v", err)
	}

	// Update with current timestamps
	mockeryInfo, _ = os.Stat(mockeryConfigProxy)
	interfaceInfo, _ = os.Stat(interfaceFileProxy)
	artifact.Dependencies = []forge.ArtifactDependency{
		{
			Type:      forge.DependencyTypeFile,
			FilePath:  absMockeryConfig,
			Timestamp: mockeryInfo.ModTime().UTC().Format(time.RFC3339),
		},
		{
			Type:      forge.DependencyTypeFile,
			FilePath:  absInterfaceFile,
			Timestamp: interfaceInfo.ModTime().UTC().Format(time.RFC3339),
		},
	}
	artifact.DependencyDetectorEngine = "go://go-gen-mocks-dep-detector"
	forge.AddOrUpdateArtifact(&store, artifact)
	if err := forge.WriteArtifactStore(artifactStorePath, store); err != nil {
		t.Fatalf("Failed to write artifact store: %v", err)
	}

	cmd = exec.Command(forgeBin, "--config", testForgeYamlPath, "build", "test-mocks-artifact")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\nOutput: %s", err, string(output))
	}
	outputStr = string(output)
	if !strings.Contains(outputStr, "Skipping test-mocks-artifact") {
		t.Errorf("Expected 'Skipping' message after update, got: %s", outputStr)
	}

	t.Log("All lazy rebuild scenarios for mocks passed")
}
