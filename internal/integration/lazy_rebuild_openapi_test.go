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

// TestLazyRebuildOpenAPI tests the lazy rebuild workflow for OpenAPI code generation.
// This test verifies that the lazy rebuild logic works correctly with dependencies
// that match what the go-gen-openapi-dep-detector would produce.
//
// The test runs within the forge repository and uses an existing build target
// as a proxy for testing the lazy rebuild logic with OpenAPI-style dependencies.
// This avoids the complexity of setting up a separate project with oapi-codegen.
//
// Test scenario:
// 1. Build an artifact (first build)
// 2. Add OpenAPI-style dependencies (spec files)
// 3. Build again (no changes) -> SKIPPED
// 4. Touch spec file dependency
// 5. Build again -> REBUILT (dependency changed)
// 6. Update dependencies, build again -> SKIPPED
func TestLazyRebuildOpenAPI(t *testing.T) {
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
	artifactStorePath := ".forge/test-lazy-openapi-artifact-store.yaml"
	defer func() { _ = os.Remove(artifactStorePath) }()
	_ = os.Remove(artifactStorePath)

	// Create a temporary forge.yaml for this test
	testForgeYaml := `name: lazy-openapi-test
artifactStorePath: ` + artifactStorePath + `

build:
  - name: test-openapi-artifact
    src: ./cmd/go-format
    dest: ./build/bin
    engine: go://go-build
`
	testForgeYamlPath := "forge-test-lazy-openapi.yaml"
	if err := os.WriteFile(testForgeYamlPath, []byte(testForgeYaml), 0o644); err != nil {
		t.Fatalf("Failed to write test forge.yaml: %v", err)
	}
	defer func() { _ = os.Remove(testForgeYamlPath) }()

	// Step 1: First build
	t.Log("Step 1: First build")
	cmd = exec.Command(forgeBin, "--config", testForgeYamlPath, "build", "test-openapi-artifact")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\nOutput: %s", err, string(output))
	}
	outputStr := string(output)
	if !strings.Contains(outputStr, "Building test-openapi-artifact") {
		t.Errorf("Expected 'Building' message, got: %s", outputStr)
	}

	// Step 2: Add OpenAPI-style dependencies
	t.Log("Step 2: Adding OpenAPI-style dependencies to artifact store")
	store, err := forge.ReadArtifactStore(artifactStorePath)
	if err != nil {
		t.Fatalf("Failed to read artifact store: %v", err)
	}

	artifact, err := forge.GetLatestArtifact(store, "test-openapi-artifact")
	if err != nil {
		t.Fatalf("Artifact not found in store: %v", err)
	}

	// Simulate OpenAPI-style dependencies:
	// - OpenAPI spec files (we'll use cmd/go-format/build.go as proxy for spec file)
	specFileProxy := filepath.Join("cmd", "go-format", "build.go")

	specInfo, err := os.Stat(specFileProxy)
	if err != nil {
		t.Fatalf("Failed to stat %s: %v", specFileProxy, err)
	}
	absSpecFile, _ := filepath.Abs(specFileProxy)

	// Add dependencies simulating what go-gen-openapi-dep-detector would produce
	artifact.Dependencies = []forge.ArtifactDependency{
		{
			Type:      forge.DependencyTypeFile,
			FilePath:  absSpecFile,
			Timestamp: specInfo.ModTime().UTC().Format(time.RFC3339),
		},
	}
	artifact.DependencyDetectorEngine = "go://go-gen-openapi-dep-detector"
	forge.AddOrUpdateArtifact(&store, artifact)

	if err := forge.WriteArtifactStore(artifactStorePath, store); err != nil {
		t.Fatalf("Failed to write artifact store: %v", err)
	}
	t.Logf("Dependencies tracked: %d", len(artifact.Dependencies))

	// Step 3: Second build - should skip
	t.Log("Step 3: Second build - should skip (unchanged)")
	cmd = exec.Command(forgeBin, "--config", testForgeYamlPath, "build", "test-openapi-artifact")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\nOutput: %s", err, string(output))
	}
	outputStr = string(output)
	if !strings.Contains(outputStr, "Skipping test-openapi-artifact") {
		t.Errorf("Expected 'Skipping' message, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "unchanged") {
		t.Errorf("Expected 'unchanged' in output, got: %s", outputStr)
	}

	// Step 4: Touch spec file dependency
	t.Log("Step 4: Touch spec file dependency")
	time.Sleep(time.Second) // Ensure timestamp changes
	now := time.Now()
	if err := os.Chtimes(absSpecFile, now, now); err != nil {
		t.Fatalf("Failed to touch file: %v", err)
	}

	// Step 5: Third build - should rebuild
	t.Log("Step 5: Third build - should rebuild (dependency changed)")
	cmd = exec.Command(forgeBin, "--config", testForgeYamlPath, "build", "test-openapi-artifact")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\nOutput: %s", err, string(output))
	}
	outputStr = string(output)
	if !strings.Contains(outputStr, "Building test-openapi-artifact") {
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
	artifact, err = forge.GetLatestArtifact(store, "test-openapi-artifact")
	if err != nil {
		t.Fatalf("Artifact not found after rebuild: %v", err)
	}

	// Update with current timestamps
	specInfo, _ = os.Stat(specFileProxy)
	artifact.Dependencies = []forge.ArtifactDependency{
		{
			Type:      forge.DependencyTypeFile,
			FilePath:  absSpecFile,
			Timestamp: specInfo.ModTime().UTC().Format(time.RFC3339),
		},
	}
	artifact.DependencyDetectorEngine = "go://go-gen-openapi-dep-detector"
	forge.AddOrUpdateArtifact(&store, artifact)
	if err := forge.WriteArtifactStore(artifactStorePath, store); err != nil {
		t.Fatalf("Failed to write artifact store: %v", err)
	}

	cmd = exec.Command(forgeBin, "--config", testForgeYamlPath, "build", "test-openapi-artifact")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\nOutput: %s", err, string(output))
	}
	outputStr = string(output)
	if !strings.Contains(outputStr, "Skipping test-openapi-artifact") {
		t.Errorf("Expected 'Skipping' message after update, got: %s", outputStr)
	}

	t.Log("All lazy rebuild scenarios for OpenAPI passed")
}
