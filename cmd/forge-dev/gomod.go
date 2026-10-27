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

type SpecTypesContext struct {
	ImportPath  string
	PackageName string
	Prefix      string
	OutputDir   string
}

var modulePathRegexp = regexp.MustCompile(`(?m)^module\s+(\S+)`)

func FindGoMod(startDir string) (string, error) {
	dir := startDir

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("cannot find go.mod in parent directories of %s", startDir)
		}
		dir = parent
	}
}

func ParseModulePath(goModPath string) (string, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("reading go.mod: %w", err)
	}

	matches := modulePathRegexp.FindSubmatch(data)
	if matches == nil {
		return "", fmt.Errorf("no module line found in %s", goModPath)
	}

	return string(matches[1]), nil
}

func ResolveSpecTypesContext(srcDir string, config *SpecTypesConfig) (*SpecTypesContext, error) {
	if config == nil || !config.Enabled {
		return nil, nil
	}

	goModDir, err := FindGoMod(srcDir)
	if err != nil {
		return nil, fmt.Errorf("finding go.mod: %w", err)
	}

	goModPath := filepath.Join(goModDir, "go.mod")
	modulePath, err := ParseModulePath(goModPath)
	if err != nil {
		return nil, fmt.Errorf("parsing module path: %w", err)
	}

	importPath := modulePath + "/" + config.OutputPath

	outputDir := filepath.Join(goModDir, config.OutputPath)

	return &SpecTypesContext{
		ImportPath:  importPath,
		PackageName: config.PackageName,
		Prefix:      config.PackageName + ".",
		OutputDir:   outputDir,
	}, nil
}
