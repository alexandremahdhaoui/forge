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
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"golang.org/x/mod/modfile"
	"gopkg.in/yaml.v3"
)

// MockeryConfig represents the parsed mockery configuration file.
type MockeryConfig struct {
	Packages map[string]MockeryPackageConfig `yaml:"packages"`
}

// MockeryPackageConfig represents package configuration in mockery.
type MockeryPackageConfig struct {
	Interfaces map[string]interface{} `yaml:"interfaces"`
}

// ----------------------------------------------------- CONFIG DISCOVERY -------------------------------------------- //

// findMockeryConfig discovers the mockery configuration file path.
// Discovery order:
//  1. MOCKERY_CONFIG_PATH environment variable
//  2. .mockery.yaml in workDir
//  3. .mockery.yml in workDir
//  4. mockery.yaml in workDir
//  5. mockery.yml in workDir
func findMockeryConfig(workDir string) (string, error) {
	// 1. Check MOCKERY_CONFIG_PATH environment variable
	if envPath := os.Getenv("MOCKERY_CONFIG_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return filepath.Abs(envPath)
		}
	}

	// 2-5. Check known config file names
	configNames := []string{".mockery.yaml", ".mockery.yml", "mockery.yaml", "mockery.yml"}
	for _, name := range configNames {
		path := filepath.Join(workDir, name)
		if _, err := os.Stat(path); err == nil {
			return filepath.Abs(path)
		}
	}

	return "", fmt.Errorf("no mockery config found in %s and MOCKERY_CONFIG_PATH not set", workDir)
}

// ----------------------------------------------------- CONFIG PARSING ---------------------------------------------- //

// parseMockeryConfig parses a mockery configuration file and extracts package paths.
func parseMockeryConfig(configPath string) (*MockeryConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read mockery config: %w", err)
	}

	var config MockeryConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse mockery config: %w", err)
	}

	return &config, nil
}

// ----------------------------------------------------- PACKAGE RESOLUTION ------------------------------------------ //

// resolvePackageToFiles resolves a Go package import path to source file paths.
// Returns all .go files in the package directory (excluding _test.go).
// Returns error for external packages (not under the module path).
func resolvePackageToFiles(pkgPath string, workDir string) ([]string, error) {
	// Find go.mod
	goModPath, err := findGoMod(workDir)
	if err != nil {
		return nil, fmt.Errorf("go.mod not found: %w", err)
	}

	// Parse go.mod
	goModData, err := parseGoMod(goModPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod: %w", err)
	}

	modulePath := goModData.Module.Mod.Path
	moduleDir := filepath.Dir(goModPath)

	// Check if local package (under module path)
	if !strings.HasPrefix(pkgPath, modulePath) {
		// EXTERNAL PACKAGE - NOT SUPPORTED IN V1
		return nil, fmt.Errorf("package %s is external (not under module %s), not tracked in v1", pkgPath, modulePath)
	}

	// Compute relative path
	relPath := strings.TrimPrefix(pkgPath, modulePath)
	relPath = strings.TrimPrefix(relPath, "/")
	pkgDir := filepath.Join(moduleDir, relPath)

	// List .go files
	return listGoFiles(pkgDir)
}

// listGoFiles returns all .go files in a directory (excluding _test.go files).
func listGoFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			absPath, err := filepath.Abs(filepath.Join(dir, name))
			if err != nil {
				return nil, fmt.Errorf("failed to get absolute path for %s: %w", name, err)
			}
			files = append(files, absPath)
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no .go files found in directory %s", dir)
	}

	return files, nil
}

// ----------------------------------------------------- CORE DETECTION ---------------------------------------------- //

// DetectMockDependencies detects all dependencies for mockery mock generation.
// It finds the mockery config, parses it, and resolves all local packages to files.
// External packages are skipped with a warning (not supported in v1).
func DetectMockDependencies(input mcptypes.DetectMockDependenciesInput) (mcptypes.DetectDependenciesOutput, error) {
	var deps []mcptypes.Dependency

	// 1. Find mockery config
	configPath, err := findMockeryConfig(input.WorkDir)
	if err != nil {
		return mcptypes.DetectDependenciesOutput{}, err
	}

	// 2. Add config file as dependency
	configDep, err := createFileDependency(configPath)
	if err != nil {
		return mcptypes.DetectDependenciesOutput{}, fmt.Errorf("failed to stat config file: %w", err)
	}
	deps = append(deps, configDep)

	// 3. Find and add go.mod
	goModPath, err := findGoMod(input.WorkDir)
	if err == nil {
		goModDep, err := createFileDependency(goModPath)
		if err == nil {
			deps = append(deps, goModDep)
		}
	}

	// 4. Parse config
	config, err := parseMockeryConfig(configPath)
	if err != nil {
		return mcptypes.DetectDependenciesOutput{}, err
	}

	// 5. Resolve each package to files
	for pkgPath := range config.Packages {
		files, err := resolvePackageToFiles(pkgPath, input.WorkDir)
		if err != nil {
			// Log warning but continue - external packages or missing packages don't fail the build
			log.Printf("Warning: skipping package %s: %v", pkgPath, err)
			continue
		}
		for _, file := range files {
			fileDep, err := createFileDependency(file)
			if err != nil {
				log.Printf("Warning: failed to stat file %s: %v", file, err)
				continue
			}
			deps = append(deps, fileDep)
		}
	}

	return mcptypes.DetectDependenciesOutput{Dependencies: deps}, nil
}

// createFileDependency creates a file dependency with timestamp.
func createFileDependency(path string) (mcptypes.Dependency, error) {
	info, err := os.Stat(path)
	if err != nil {
		return mcptypes.Dependency{}, err
	}
	return mcptypes.Dependency{
		Type:      "file",
		FilePath:  path,
		Timestamp: info.ModTime().UTC().Format(time.RFC3339),
	}, nil
}

// ----------------------------------------------------- GO.MOD HELPERS ---------------------------------------------- //
// NOTE: These functions are COPIED from /cmd/go-dependency-detector/detect.go
// as per the plan (avoid refactoring existing code in v1).

// findGoMod walks up the directory tree to find go.mod.
func findGoMod(startPath string) (string, error) {
	dir := startPath
	if !filepath.IsAbs(dir) {
		var err error
		dir, err = filepath.Abs(dir)
		if err != nil {
			return "", err
		}
	}

	// If startPath is a file, start from its directory
	info, err := os.Stat(dir)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		dir = filepath.Dir(dir)
	}

	// Walk up the directory tree
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return goModPath, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", fmt.Errorf("go.mod not found in any parent directory of %s", startPath)
		}
		dir = parent
	}
}

// parseGoMod parses a go.mod file.
func parseGoMod(goModPath string) (*modfile.File, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read go.mod: %w", err)
	}

	modFile, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod: %w", err)
	}

	return modFile, nil
}
