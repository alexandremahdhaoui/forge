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

package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

// ExtractTestID extracts testID from command output.
// testID format: test-<stage>-YYYYMMDD-XXXXXXXX
func ExtractTestID(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "test-") && len(line) > 10 {
			// Verify format
			parts := strings.Split(line, "-")
			if len(parts) >= 4 {
				return line
			}
		}
	}
	return ""
}

// VerifyClusterExists checks if a KIND cluster exists.
// It respects the KIND_BINARY environment variable.
func VerifyClusterExists(testID string) error {
	kindBinary := os.Getenv("KIND_BINARY")
	if kindBinary == "" {
		kindBinary = "kind"
	}

	expectedClusterName := fmt.Sprintf("forge-%s", testID)

	cmd := exec.Command(kindBinary, "get", "clusters")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get clusters: %w\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), expectedClusterName) {
		return fmt.Errorf("cluster %s not found in kind clusters", expectedClusterName)
	}

	return nil
}

// VerifyArtifactStoreHasTestEnv checks if artifact store contains a test environment.
func VerifyArtifactStoreHasTestEnv(testID string) error {
	storePath, err := forge.GetArtifactStorePath(".forge/artifacts.json")
	if err != nil {
		return fmt.Errorf("failed to get artifact store path: %w", err)
	}
	data, err := os.ReadFile(storePath)
	if err != nil {
		return fmt.Errorf("failed to read artifact store: %w", err)
	}

	content := string(data)
	if !strings.Contains(content, testID) {
		return fmt.Errorf("testID %s not found in artifact store", testID)
	}

	// Should contain test environment structure
	if !strings.Contains(content, "testEnvironments") && !strings.Contains(content, "\"id\"") {
		return fmt.Errorf("artifact store missing test environment structure")
	}

	return nil
}

// VerifyArtifactStoreMissingTestEnv checks that artifact store doesn't contain a test environment.
func VerifyArtifactStoreMissingTestEnv(testID string) error {
	storePath, err := forge.GetArtifactStorePath(".forge/artifacts.json")
	if err != nil {
		return fmt.Errorf("failed to get artifact store path: %w", err)
	}
	data, err := os.ReadFile(storePath)
	if err != nil {
		return fmt.Errorf("failed to read artifact store: %w", err)
	}

	content := string(data)
	if strings.Contains(content, testID) {
		return fmt.Errorf("testID %s still found in artifact store after deletion", testID)
	}

	return nil
}

// ForceCleanupTestEnv forcefully cleans up a test environment without artifact store dependency.
// The stage parameter specifies which test stage the environment belongs to (e.g., "integration", "e2e-stub").
// If stage is empty, it defaults to "integration".
func ForceCleanupTestEnv(testID string, stage ...string) error {
	if testID == "" {
		return nil
	}

	// Default to integration stage if not specified
	stageName := "integration"
	if len(stage) > 0 && stage[0] != "" {
		stageName = stage[0]
	}

	var errors []error

	// Only delete KIND cluster for non-stub stages
	if stageName != "e2e-stub" {
		kindBinary := os.Getenv("KIND_BINARY")
		if kindBinary == "" {
			kindBinary = "kind"
		}

		clusterName := fmt.Sprintf("forge-%s", testID)
		fmt.Fprintf(os.Stderr, "Deleting cluster: %s\n", clusterName)
		deleteCmd := exec.Command(kindBinary, "delete", "cluster", "--name", clusterName)
		if err := deleteCmd.Run(); err != nil {
			// Only add error if cluster might exist (ignore "not found" errors)
			errors = append(errors, fmt.Errorf("failed to delete cluster %s: %w", clusterName, err))
		}
	}

	// Delete tmp directory
	rootDir, err := os.Getwd()
	if err == nil {
		tmpDir := filepath.Join(rootDir, "tmp", testID)
		if err := os.RemoveAll(tmpDir); err != nil {
			errors = append(errors, fmt.Errorf("failed to remove tmpDir %s: %w", tmpDir, err))
		}
	}

	// Try to remove from artifact store (best effort)
	cleanupTestEnvViaForge(testID, stageName)

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}
	return nil
}

// cleanupTestEnvViaForge deletes a test environment via the forge CLI.
// This is a helper for ForceCleanupTestEnv.
func cleanupTestEnvViaForge(testID, stage string) {
	if testID == "" {
		return
	}

	if stage == "" {
		stage = "integration"
	}

	// Try to delete via forge
	cmd := exec.Command("./build/bin/forge", "test", "delete-env", stage, testID)
	cmd.Env = os.Environ()
	_ = cmd.Run() // Ignore errors during cleanup
}

// ForceCleanupLeftovers cleans up leftover resources tracked in the local artifact store.
// This ensures we only delete clusters that belong to THIS forge instance,
// preventing accidental deletion of clusters from other forge instances running in parallel.
func ForceCleanupLeftovers() error {
	var errors []error

	// Build a set of tracked cluster names from the local artifact store
	trackedClusters := make(map[string]bool)
	artifactStorePath, err := forge.GetArtifactStorePath(".forge/artifacts.json")
	if err == nil {
		store, err := forge.ReadArtifactStore(artifactStorePath)
		if err == nil {
			for _, env := range store.TestEnvironments {
				if clusterName, ok := env.Metadata["testenv-kind.clusterName"]; ok && clusterName != "" {
					trackedClusters[clusterName] = true
				}
			}
		}
	}

	// Cleanup KIND clusters - ONLY those tracked in the local artifact store
	kindBinary := os.Getenv("KIND_BINARY")
	if kindBinary == "" {
		kindBinary = "kind"
	}

	cmd := exec.Command(kindBinary, "get", "clusters")
	output, err := cmd.CombinedOutput()
	if err == nil {
		clusters := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, cluster := range clusters {
			cluster = strings.TrimSpace(cluster)
			// Only delete clusters that are tracked in our local artifact store
			// This prevents deleting clusters from other forge instances
			if cluster != "" && trackedClusters[cluster] {
				fmt.Fprintf(os.Stderr, "Cleaning up leftover cluster: %s\n", cluster)
				deleteCmd := exec.Command(kindBinary, "delete", "cluster", "--name", cluster)
				if err := deleteCmd.Run(); err != nil {
					errors = append(errors, fmt.Errorf("failed to delete cluster %s: %w", cluster, err))
				}
			}
		}
	}

	// Cleanup tmp directories
	rootDir, err := os.Getwd()
	if err == nil {
		tmpBase := filepath.Join(rootDir, "tmp")
		entries, err := os.ReadDir(tmpBase)
		if err == nil {
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), "test-integration-") || strings.HasPrefix(entry.Name(), "tmp-") {
					dirPath := filepath.Join(tmpBase, entry.Name())
					if err := os.RemoveAll(dirPath); err != nil {
						errors = append(errors, fmt.Errorf("failed to remove %s: %w", dirPath, err))
					}
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}
	return nil
}

// FindForgeBinary locates the forge binary for testing.
// It tries multiple locations:
// 1. build/bin/forge in current directory and parent directories
// 2. Attempts to build forge if go.mod is found
// 3. Searches in PATH
func FindForgeBinary() (string, error) {
	// Try to find forge in build/bin
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Walk up to find forge repository root
	dir := cwd
	for {
		// Check for forge binary
		forgeBin := filepath.Join(dir, "build", "bin", "forge")
		if _, err := os.Stat(forgeBin); err == nil {
			return forgeBin, nil
		}

		// Check if we've reached a forge repo
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			// Found go.mod, try to build forge
			buildCmd := exec.Command("go", "build", "-o", "build/bin/forge", "./cmd/forge")
			buildCmd.Dir = dir
			if err := buildCmd.Run(); err != nil {
				return "", fmt.Errorf("forge binary not found and build failed: %w", err)
			}
			return filepath.Join(dir, "build", "bin", "forge"), nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Try PATH
	if forgePath, err := exec.LookPath("forge"); err == nil {
		return forgePath, nil
	}

	return "", fmt.Errorf("forge binary not found (checked build/bin/forge, attempted build, and PATH)")
}

// FindForgeRepository locates the forge repository root.
// It walks up the directory tree looking for go.mod with the forge module path.
func FindForgeRepository() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Walk up to find forge repository
	dir := cwd
	for {
		// Check for go.mod with forge module
		goModPath := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(goModPath); err == nil {
			if strings.Contains(string(data), "github.com/alexandremahdhaoui/forge") {
				// Also check for cmd/forge/main.go
				if _, err := os.Stat(filepath.Join(dir, "cmd", "forge", "main.go")); err == nil {
					return dir, nil
				}
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("forge repository not found")
}
