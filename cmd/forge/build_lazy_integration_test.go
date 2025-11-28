//go:build integration

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

// clearArtifactsOnly removes all artifacts from the store while preserving testEnvironments and testReports.
// This allows tests to start fresh with artifacts without interfering with testenv cleanup.
func clearArtifactsOnly(t *testing.T, artifactStorePath string) {
	t.Helper()

	store, err := forge.ReadArtifactStore(artifactStorePath)
	if err != nil {
		// No store exists, nothing to clear
		return
	}

	// Clear only artifacts, preserve testEnvironments and testReports
	store.Artifacts = nil

	if err := forge.WriteArtifactStore(artifactStorePath, store); err != nil {
		t.Logf("Warning: failed to clear artifacts: %v", err)
	}
}

// TestLazyRebuild_WithDependencyTracking tests the complete lazy rebuild workflow
// when dependencies are tracked in the artifact store.
func TestLazyRebuild_WithDependencyTracking(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Change to repository root
	repoRoot := "../.."
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Failed to change to repo root: %v", err)
	}

	// Build forge binary first
	forgeBin := "./build/bin/forge"
	cmd := exec.Command("go", "build", "-o", forgeBin, "./cmd/forge")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build forge: %v", err)
	}

	// Clear only artifacts (preserve testEnvironments for cleanup)
	artifactStorePath := ".forge/artifact-store.yaml"
	clearArtifactsOnly(t, artifactStorePath)

	// Step 1: First build - should build with "no previous build"
	t.Log("Step 1: First build")
	cmd = exec.Command(forgeBin, "build", "go-lint")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\nOutput: %s", err, string(output))
	}
	outputStr := string(output)
	if !strings.Contains(outputStr, "🔨 Building go-lint (no previous build)") {
		t.Errorf("Expected 'no previous build' message, got: %s", outputStr)
	}

	// Step 2: Read artifact store and manually add dependencies to simulate dependency tracking
	t.Log("Step 2: Adding dependencies to artifact store")
	store, err := forge.ReadArtifactStore(artifactStorePath)
	if err != nil {
		t.Fatalf("Failed to read artifact store: %v", err)
	}

	// Find go-lint artifact
	var artifact *forge.Artifact
	for i := range store.Artifacts {
		if store.Artifacts[i].Name == "go-lint" {
			artifact = &store.Artifacts[i]
			break
		}
	}
	if artifact == nil {
		t.Fatal("go-lint artifact not found in store")
	}

	// Add mock dependencies with current timestamps
	mainGo := filepath.Join("cmd", "go-lint", "main.go")
	mainInfo, err := os.Stat(mainGo)
	if err != nil {
		t.Fatalf("Failed to stat main.go: %v", err)
	}
	absMainGo, _ := filepath.Abs(mainGo)

	artifact.Dependencies = []forge.ArtifactDependency{
		{
			Type:      forge.DependencyTypeFile,
			FilePath:  absMainGo,
			Timestamp: mainInfo.ModTime().UTC().Format(time.RFC3339),
		},
	}
	artifact.DependencyDetectorEngine = "go://go-dependency-detector"

	// Write updated store
	if err := forge.WriteArtifactStore(artifactStorePath, store); err != nil {
		t.Fatalf("Failed to write artifact store: %v", err)
	}

	// Step 3: Second build - should skip (unchanged)
	t.Log("Step 3: Second build - should skip")
	cmd = exec.Command(forgeBin, "build", "go-lint")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\nOutput: %s", err, string(output))
	}
	outputStr = string(output)
	if !strings.Contains(outputStr, "⏭  Skipping go-lint (unchanged)") {
		t.Errorf("Expected 'Skipping' message, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "✅ Artifact go-lint is up to date") {
		t.Errorf("Expected 'up to date' message, got: %s", outputStr)
	}

	// Step 4: Touch dependency file - should rebuild
	t.Log("Step 4: Touch dependency - should rebuild")
	time.Sleep(time.Second) // Ensure timestamp changes
	now := time.Now()
	if err := os.Chtimes(absMainGo, now, now); err != nil {
		t.Fatalf("Failed to touch file: %v", err)
	}

	cmd = exec.Command(forgeBin, "build", "go-lint")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\nOutput: %s", err, string(output))
	}
	outputStr = string(output)
	if !strings.Contains(outputStr, "🔨 Building go-lint (dependency") {
		t.Errorf("Expected 'dependency ... modified' message, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "modified)") {
		t.Errorf("Expected 'modified' reason, got: %s", outputStr)
	}

	// Step 5: Force rebuild - should always rebuild
	t.Log("Step 5: Force rebuild")
	cmd = exec.Command(forgeBin, "build", "--force", "go-lint")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\nOutput: %s", err, string(output))
	}
	outputStr = string(output)
	if !strings.Contains(outputStr, "🔨 Building go-lint (force flag set)") {
		t.Errorf("Expected 'force flag set' message, got: %s", outputStr)
	}

	t.Log("✅ All lazy rebuild scenarios passed")
}

// TestLazyRebuild_WithoutDependencyTracking verifies that artifacts without
// dependency tracking always rebuild with appropriate message.
func TestLazyRebuild_WithoutDependencyTracking(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Change to repository root
	repoRoot := "../.."
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Failed to change to repo root: %v", err)
	}

	forgeBin := "./build/bin/forge"
	if _, err := os.Stat(forgeBin); os.IsNotExist(err) {
		t.Skip("forge binary not found, run full build test first")
	}

	// Clear only artifacts (preserve testEnvironments for cleanup)
	artifactStorePath := ".forge/artifact-store.yaml"
	clearArtifactsOnly(t, artifactStorePath)

	// Build once
	cmd := exec.Command(forgeBin, "build", "testenv-kind")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\nOutput: %s", err, string(output))
	}

	// Manually remove dependency tracking from artifact store to simulate old artifacts
	// This simulates an artifact built before dependency tracking was implemented
	store, err := forge.ReadArtifactStore(artifactStorePath)
	if err != nil {
		t.Fatalf("Failed to read artifact store: %v", err)
	}

	// Find testenv-kind artifact and remove its dependency tracking
	for i := range store.Artifacts {
		if store.Artifacts[i].Name == "testenv-kind" {
			store.Artifacts[i].Dependencies = nil
			store.Artifacts[i].DependencyDetectorEngine = ""
			break
		}
	}

	if err := forge.WriteArtifactStore(artifactStorePath, store); err != nil {
		t.Fatalf("Failed to write modified artifact store: %v", err)
	}

	// Build again - should rebuild with "dependencies not tracked"
	cmd = exec.Command(forgeBin, "build", "testenv-kind")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\nOutput: %s", err, string(output))
	}
	outputStr := string(output)
	if !strings.Contains(outputStr, "🔨 Building testenv-kind (dependencies not tracked)") {
		t.Errorf("Expected 'dependencies not tracked' message, got: %s", outputStr)
	}
}

// TestLazyRebuild_ArtifactDeleted tests that a deleted artifact file is rebuilt.
func TestLazyRebuild_ArtifactDeleted(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Change to repository root
	repoRoot := "../.."
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Failed to change to repo root: %v", err)
	}

	// Build forge binary first
	forgeBin := "./build/bin/forge"
	cmd := exec.Command("go", "build", "-o", forgeBin, "./cmd/forge")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build forge: %v", err)
	}

	// Clear only artifacts (preserve testEnvironments for cleanup)
	artifactStorePath := ".forge/artifact-store.yaml"
	clearArtifactsOnly(t, artifactStorePath)

	// Step 1: First build
	t.Log("Step 1: First build")
	cmd = exec.Command(forgeBin, "build", "go-lint")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\nOutput: %s", err, string(output))
	}

	// Step 2: Verify artifact exists
	t.Log("Step 2: Verify artifact exists")
	artifactPath := "./build/bin/go-lint"
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		t.Fatalf("Artifact not found at %s", artifactPath)
	}

	// Step 3: Add dependencies to artifact store (to enable lazy rebuild)
	t.Log("Step 3: Adding dependencies to artifact store")
	store, err := forge.ReadArtifactStore(artifactStorePath)
	if err != nil {
		t.Fatalf("Failed to read artifact store: %v", err)
	}

	var artifact *forge.Artifact
	for i := range store.Artifacts {
		if store.Artifacts[i].Name == "go-lint" {
			artifact = &store.Artifacts[i]
			break
		}
	}
	if artifact == nil {
		t.Fatal("go-lint artifact not found in store")
	}

	// Add mock dependencies
	mainGo := filepath.Join("cmd", "go-lint", "main.go")
	mainInfo, err := os.Stat(mainGo)
	if err != nil {
		t.Fatalf("Failed to stat main.go: %v", err)
	}
	absMainGo, _ := filepath.Abs(mainGo)

	artifact.Dependencies = []forge.ArtifactDependency{
		{
			Type:      forge.DependencyTypeFile,
			FilePath:  absMainGo,
			Timestamp: mainInfo.ModTime().UTC().Format(time.RFC3339),
		},
	}
	artifact.DependencyDetectorEngine = "go://go-dependency-detector"

	if err := forge.WriteArtifactStore(artifactStorePath, store); err != nil {
		t.Fatalf("Failed to write artifact store: %v", err)
	}

	// Step 4: Delete artifact file
	t.Log("Step 4: Delete artifact file")
	if err := os.Remove(artifactPath); err != nil {
		t.Fatalf("Failed to delete artifact: %v", err)
	}

	// Step 5: Run forge build again - should rebuild with "artifact file missing"
	t.Log("Step 5: Rebuild after deletion")
	cmd = exec.Command(forgeBin, "build", "go-lint")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\nOutput: %s", err, string(output))
	}
	outputStr := string(output)
	if !strings.Contains(outputStr, "🔨 Building go-lint (artifact file missing)") {
		t.Errorf("Expected 'artifact file missing' message, got: %s", outputStr)
	}

	// Step 6: Verify artifact was rebuilt
	t.Log("Step 6: Verify artifact rebuilt")
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		t.Fatalf("Artifact was not rebuilt at %s", artifactPath)
	}

	t.Log("✅ Artifact deletion scenario passed")
}

// TestLazyRebuild_ExternalDepChanged tests that modifying go.mod triggers rebuild.
func TestLazyRebuild_ExternalDepChanged(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Change to repository root
	repoRoot := "../.."
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Failed to change to repo root: %v", err)
	}

	// Build forge binary first
	forgeBin := "./build/bin/forge"
	cmd := exec.Command("go", "build", "-o", forgeBin, "./cmd/forge")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build forge: %v", err)
	}

	// Clear only artifacts (preserve testEnvironments for cleanup)
	artifactStorePath := ".forge/artifact-store.yaml"
	clearArtifactsOnly(t, artifactStorePath)

	// Step 1: First build
	t.Log("Step 1: First build")
	cmd = exec.Command(forgeBin, "build", "go-lint")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\nOutput: %s", err, string(output))
	}

	// Step 2: Add dependencies including go.mod to artifact store
	t.Log("Step 2: Adding dependencies including go.mod")
	store, err := forge.ReadArtifactStore(artifactStorePath)
	if err != nil {
		t.Fatalf("Failed to read artifact store: %v", err)
	}

	var artifact *forge.Artifact
	for i := range store.Artifacts {
		if store.Artifacts[i].Name == "go-lint" {
			artifact = &store.Artifacts[i]
			break
		}
	}
	if artifact == nil {
		t.Fatal("go-lint artifact not found in store")
	}

	// Add go.mod as a dependency
	goModPath := "go.mod"
	goModInfo, err := os.Stat(goModPath)
	if err != nil {
		t.Fatalf("Failed to stat go.mod: %v", err)
	}
	absGoMod, _ := filepath.Abs(goModPath)

	mainGo := filepath.Join("cmd", "go-lint", "main.go")
	mainInfo, err := os.Stat(mainGo)
	if err != nil {
		t.Fatalf("Failed to stat main.go: %v", err)
	}
	absMainGo, _ := filepath.Abs(mainGo)

	artifact.Dependencies = []forge.ArtifactDependency{
		{
			Type:      forge.DependencyTypeFile,
			FilePath:  absMainGo,
			Timestamp: mainInfo.ModTime().UTC().Format(time.RFC3339),
		},
		{
			Type:      forge.DependencyTypeFile,
			FilePath:  absGoMod,
			Timestamp: goModInfo.ModTime().UTC().Format(time.RFC3339),
		},
	}
	artifact.DependencyDetectorEngine = "go://go-dependency-detector"

	if err := forge.WriteArtifactStore(artifactStorePath, store); err != nil {
		t.Fatalf("Failed to write artifact store: %v", err)
	}

	// Step 3: Second build - should skip (unchanged)
	t.Log("Step 3: Second build - should skip")
	cmd = exec.Command(forgeBin, "build", "go-lint")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\nOutput: %s", err, string(output))
	}
	outputStr := string(output)
	if !strings.Contains(outputStr, "⏭  Skipping go-lint (unchanged)") {
		t.Errorf("Expected 'Skipping' message, got: %s", outputStr)
	}

	// Step 4: Touch go.mod - should trigger rebuild
	t.Log("Step 4: Touch go.mod - should rebuild")
	time.Sleep(time.Second) // Ensure timestamp changes
	now := time.Now()
	if err := os.Chtimes(absGoMod, now, now); err != nil {
		t.Fatalf("Failed to touch go.mod: %v", err)
	}

	cmd = exec.Command(forgeBin, "build", "go-lint")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\nOutput: %s", err, string(output))
	}
	outputStr = string(output)
	if !strings.Contains(outputStr, "🔨 Building go-lint (dependency") && !strings.Contains(outputStr, "modified)") {
		t.Errorf("Expected 'dependency ... modified' message, got: %s", outputStr)
	}

	t.Log("✅ External dependency change scenario passed")
}

// TestLazyRebuild_MixedChanges tests selective rebuild with multiple artifacts.
func TestLazyRebuild_MixedChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Change to repository root
	repoRoot := "../.."
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Failed to change to repo root: %v", err)
	}

	// Build forge binary first
	forgeBin := "./build/bin/forge"
	cmd := exec.Command("go", "build", "-o", forgeBin, "./cmd/forge")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build forge: %v", err)
	}

	// Clear only artifacts (preserve testEnvironments for cleanup)
	artifactStorePath := ".forge/artifact-store.yaml"
	clearArtifactsOnly(t, artifactStorePath)

	// Step 1: Build two artifacts
	t.Log("Step 1: Build multiple artifacts")
	artifacts := []string{"forge", "go-build"}
	for _, artifactName := range artifacts {
		cmd = exec.Command(forgeBin, "build", artifactName)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("forge build %s failed: %v\nOutput: %s", artifactName, err, string(output))
		}
	}

	// Step 2: Add dependencies to both artifacts
	t.Log("Step 2: Adding dependencies to artifact store")
	store, err := forge.ReadArtifactStore(artifactStorePath)
	if err != nil {
		t.Fatalf("Failed to read artifact store: %v", err)
	}

	// Setup dependencies for each artifact
	artifactDeps := map[string]string{
		"forge":    filepath.Join("cmd", "forge", "main.go"),
		"go-build": filepath.Join("cmd", "go-build", "main.go"),
	}

	for artifactName, depPath := range artifactDeps {
		var artifact *forge.Artifact
		for i := range store.Artifacts {
			if store.Artifacts[i].Name == artifactName {
				artifact = &store.Artifacts[i]
				break
			}
		}
		if artifact == nil {
			t.Fatalf("%s artifact not found in store", artifactName)
		}

		depInfo, err := os.Stat(depPath)
		if err != nil {
			t.Fatalf("Failed to stat %s: %v", depPath, err)
		}
		absDepPath, _ := filepath.Abs(depPath)

		artifact.Dependencies = []forge.ArtifactDependency{
			{
				Type:      forge.DependencyTypeFile,
				FilePath:  absDepPath,
				Timestamp: depInfo.ModTime().UTC().Format(time.RFC3339),
			},
		}
		artifact.DependencyDetectorEngine = "go://go-dependency-detector"
	}

	if err := forge.WriteArtifactStore(artifactStorePath, store); err != nil {
		t.Fatalf("Failed to write artifact store: %v", err)
	}

	// Step 3: Rebuild both - should skip both
	t.Log("Step 3: Rebuild both - should skip both")
	for _, artifactName := range artifacts {
		cmd = exec.Command(forgeBin, "build", artifactName)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("forge build %s failed: %v\nOutput: %s", artifactName, err, string(output))
		}
		outputStr := string(output)
		if !strings.Contains(outputStr, "⏭  Skipping "+artifactName+" (unchanged)") {
			t.Errorf("Expected 'Skipping %s' message, got: %s", artifactName, outputStr)
		}
	}

	// Step 4: Touch only one artifact's dependency
	t.Log("Step 4: Touch only forge's dependency")
	time.Sleep(time.Second) // Ensure timestamp changes
	now := time.Now()
	forgeDep, _ := filepath.Abs(artifactDeps["forge"])
	if err := os.Chtimes(forgeDep, now, now); err != nil {
		t.Fatalf("Failed to touch forge dependency: %v", err)
	}

	// Step 5: Rebuild forge - should rebuild
	t.Log("Step 5: Rebuild forge - should rebuild")
	cmd = exec.Command(forgeBin, "build", "forge")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build forge failed: %v\nOutput: %s", err, string(output))
	}
	outputStr := string(output)

	// Verify forge was rebuilt
	if !strings.Contains(outputStr, "🔨 Building forge (dependency") && !strings.Contains(outputStr, "modified)") {
		t.Errorf("Expected 'Building forge (dependency ... modified)' message, got: %s", outputStr)
	}

	// Step 6: Rebuild go-build - should skip (unchanged)
	t.Log("Step 6: Rebuild go-build - should skip")
	cmd = exec.Command(forgeBin, "build", "go-build")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build go-build failed: %v\nOutput: %s", err, string(output))
	}
	outputStr = string(output)

	// Verify go-build was skipped
	if !strings.Contains(outputStr, "⏭  Skipping go-build (unchanged)") {
		t.Errorf("Expected 'Skipping go-build' message, got: %s", outputStr)
	}

	t.Log("✅ Mixed changes scenario passed")
}
