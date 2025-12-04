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

// Package forgepath provides utilities for locating the forge source repository
// and constructing commands to execute forge tools via `go run`.
package forgepath

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const (
	forgeModule     = "github.com/alexandremahdhaoui/forge"
	forgeRepoEnvVar = "FORGE_REPO_PATH"
)

var (
	// Cache for forge repository path to avoid repeated filesystem/command operations
	cachedForgeRepoPath string
	cachedForgeRepoErr  error
	cacheOnce           sync.Once
)

// FindForgeRepo locates the forge source repository using multiple detection methods.
// It checks in the following order:
// 1. FORGE_REPO_PATH environment variable
// 2. Go module cache using `go list -m -f '{{.Dir}}' github.com/alexandremahdhaoui/forge`
// 3. Walking up from os.Executable() to find forge repository
//
// Returns the absolute path to the forge repository or an error if not found.
func FindForgeRepo() (string, error) {
	cacheOnce.Do(func() {
		cachedForgeRepoPath, cachedForgeRepoErr = findForgeRepoUncached()
	})
	return cachedForgeRepoPath, cachedForgeRepoErr
}

// findForgeRepoUncached performs the actual forge repository detection without caching.
func findForgeRepoUncached() (string, error) {
	// Method 1: Check FORGE_REPO_PATH environment variable
	if envPath := os.Getenv(forgeRepoEnvVar); envPath != "" {
		absPath, err := filepath.Abs(envPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve FORGE_REPO_PATH: %w", err)
		}
		if IsForgeRepo(absPath) {
			return absPath, nil
		}
		return "", fmt.Errorf("FORGE_REPO_PATH points to non-forge directory: %s", absPath)
	}

	// Method 2: Use `go list` to find the module in Go's module cache
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", forgeModule)
	output, err := cmd.Output()
	if err == nil {
		modulePath := strings.TrimSpace(string(output))
		if modulePath != "" && IsForgeRepo(modulePath) {
			return modulePath, nil
		}
	}

	// Method 3: Walk up from os.Executable() to find forge repository
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve executable symlinks: %w", err)
	}

	// Walk up the directory tree
	dir := filepath.Dir(execPath)
	for {
		if IsForgeRepo(dir) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("forge repository not found (checked: env var, go list, executable path)")
}

// IsForgeRepo checks if the given directory is the forge repository.
// It verifies by checking:
// 1. go.mod exists and contains the forge module path
// 2. cmd/forge/main.go exists (main forge CLI)
func IsForgeRepo(dir string) bool {
	// Check if go.mod exists and contains forge module
	goModPath := filepath.Join(dir, "go.mod")
	goModContent, err := os.ReadFile(goModPath)
	if err != nil {
		return false
	}

	// Check if go.mod declares the forge module
	if !strings.Contains(string(goModContent), forgeModule) {
		return false
	}

	// Check if cmd/forge/main.go exists
	forgeMainPath := filepath.Join(dir, "cmd", "forge", "main.go")
	if _, err := os.Stat(forgeMainPath); err != nil {
		return false
	}

	return true
}

// BuildGoRunCommand constructs the command arguments for executing a forge MCP server
// via `go run`. The returned slice is suitable for use with exec.Command("go", args...).
//
// Environment Variables (checked in order of preference when FORGE_RUN_LOCAL_ENABLED=true):
// - FORGE_RUN_LOCAL_ENABLED: Set to "true" to run from local source using ./cmd/{packageName}
// - FORGE_RUN_LOCAL_BASEDIR: Base directory for forge repo when running locally
// - FORGE_REPO_PATH: Legacy base directory variable (for backward compatibility)
//
// Behavior:
//   - If FORGE_RUN_LOCAL_ENABLED=true:
//     → Use `go run -C {basedir} ./cmd/{packageName}`
//   - Otherwise:
//     → Use `go run github.com/alexandremahdhaoui/forge/cmd/{packageName}@{forgeVersion}`
//
// Using @version syntax ensures go run uses forge's own dependencies from its go.mod/go.sum,
// not the consuming project's dependencies. This prevents dependency conflicts when forge
// is used as a library in other projects.
//
// Example usage:
//
//	args, err := BuildGoRunCommand("testenv-kind", "v0.9.0")
//	// Returns: ["run", "github.com/alexandremahdhaoui/forge/cmd/testenv-kind@v0.9.0"]
//	cmd := exec.Command("go", args...)
func BuildGoRunCommand(packageName, forgeVersion string) ([]string, error) {
	if packageName == "" {
		return nil, fmt.Errorf("package name cannot be empty")
	}
	if forgeVersion == "" {
		return nil, fmt.Errorf("forge version cannot be empty")
	}

	// Check if local development mode should be used
	// ONLY enabled when FORGE_RUN_LOCAL_ENABLED=true
	localEnabled := os.Getenv("FORGE_RUN_LOCAL_ENABLED")
	useLocalMode := localEnabled == "true"

	if useLocalMode {
		// Check for base directory in order of preference:
		// 1. FORGE_RUN_LOCAL_BASEDIR (explicit override)
		// 2. FORGE_REPO_PATH (legacy, used by tests)
		// 3. FindForgeRepo() - searches current dir, module cache, executable path
		baseDir := os.Getenv("FORGE_RUN_LOCAL_BASEDIR")
		if baseDir == "" {
			baseDir = os.Getenv("FORGE_REPO_PATH")
		}
		if baseDir == "" {
			// Try to find forge repo automatically
			foundRepo, err := FindForgeRepo()
			if err == nil {
				baseDir = foundRepo
			}
		}

		if baseDir == "" {
			return nil, fmt.Errorf("FORGE_RUN_LOCAL_ENABLED=true but cannot find forge repository. Set FORGE_RUN_LOCAL_BASEDIR=/path/to/forge or FORGE_REPO_PATH=/path/to/forge")
		}

		// Use -C flag to run from forge directory
		pkgPath := fmt.Sprintf("./cmd/%s", packageName)
		return []string{"-C", baseDir, "run", pkgPath}, nil
	}

	// Production mode: use versioned module syntax
	// This ensures the tool runs with its own dependencies
	// Strip dirty suffixes for module resolution
	// git describe uses "-dirty", build info uses "+dirty"
	moduleVersion := forgeVersion
	moduleVersion = strings.TrimSuffix(moduleVersion, "-dirty")
	moduleVersion = strings.TrimSuffix(moduleVersion, "+dirty")
	return []string{"run", fmt.Sprintf("%s/cmd/%s@%s", forgeModule, packageName, moduleVersion)}, nil
}
