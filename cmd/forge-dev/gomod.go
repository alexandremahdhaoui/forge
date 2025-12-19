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
	"os"
	"path/filepath"
	"regexp"
)

// SpecTypesContext holds resolved information for external spec types.
// This struct is populated when specTypes.enabled = true and passed to templates.
type SpecTypesContext struct {
	// ImportPath is the full Go import path for the spec types package.
	// Example: "github.com/user/project/pkg/api/v1"
	ImportPath string
	// PackageName is the Go package name (last component of path).
	// Example: "v1"
	PackageName string
	// Prefix is the package qualifier with trailing dot.
	// Example: "v1."
	Prefix string
	// OutputDir is the absolute filesystem path where spec.go will be written.
	// Example: "/home/user/project/pkg/api/v1"
	OutputDir string
}

// modulePathRegexp matches the module line in go.mod files.
// It captures the module path (non-whitespace characters after "module").
var modulePathRegexp = regexp.MustCompile(`^module\s+(\S+)`)

// FindGoMod walks up the directory tree from startDir to find go.mod.
// It returns the directory containing go.mod, or an error if not found.
func FindGoMod(startDir string) (string, error) {
	dir := startDir

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding go.mod
			return "", fmt.Errorf("cannot find go.mod in parent directories of %s", startDir)
		}
		dir = parent
	}
}

// ParseModulePath reads a go.mod file and extracts the module path.
// It returns the module path (e.g., "github.com/user/project") or an error.
func ParseModulePath(goModPath string) (string, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("reading go.mod: %w", err)
	}

	// Find the module line
	matches := modulePathRegexp.FindSubmatch(data)
	if matches == nil {
		return "", fmt.Errorf("no module line found in %s", goModPath)
	}

	return string(matches[1]), nil
}

// ResolveSpecTypesContext resolves the spec types context from the given source directory and config.
// It returns nil, nil if config is nil or not enabled (not an error).
// When enabled, it finds go.mod, parses the module path, and computes the full import path.
func ResolveSpecTypesContext(srcDir string, config *SpecTypesConfig) (*SpecTypesContext, error) {
	// Return nil if config is nil or not enabled
	if config == nil || !config.Enabled {
		return nil, nil
	}

	// Find go.mod location
	goModDir, err := FindGoMod(srcDir)
	if err != nil {
		return nil, fmt.Errorf("finding go.mod: %w", err)
	}

	// Parse module path from go.mod
	goModPath := filepath.Join(goModDir, "go.mod")
	modulePath, err := ParseModulePath(goModPath)
	if err != nil {
		return nil, fmt.Errorf("parsing module path: %w", err)
	}

	// Compute import path: module path + "/" + output path
	importPath := modulePath + "/" + config.OutputPath

	// Compute output directory: go.mod dir + output path
	outputDir := filepath.Join(goModDir, config.OutputPath)

	return &SpecTypesContext{
		ImportPath:  importPath,
		PackageName: config.PackageName,
		Prefix:      config.PackageName + ".",
		OutputDir:   outputDir,
	}, nil
}
