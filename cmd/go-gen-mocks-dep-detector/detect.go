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

	"github.com/alexandremahdhaoui/forge/internal/gosource"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"golang.org/x/mod/modfile"
	"gopkg.in/yaml.v3"
)

type MockeryConfig struct {
	Packages map[string]MockeryPackageConfig `yaml:"packages"`
}

type MockeryPackageConfig struct {
	Interfaces map[string]interface{} `yaml:"interfaces"`
}

func findMockeryConfig(workDir string) (string, error) {
	if envPath := os.Getenv("MOCKERY_CONFIG_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return filepath.Abs(envPath)
		}
	}

	configNames := []string{".mockery.yaml", ".mockery.yml", "mockery.yaml", "mockery.yml"}
	for _, name := range configNames {
		path := filepath.Join(workDir, name)
		if _, err := os.Stat(path); err == nil {
			return filepath.Abs(path)
		}
	}

	return "", fmt.Errorf("no mockery config found in %s and MOCKERY_CONFIG_PATH not set", workDir)
}

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

func resolvePackageToFiles(pkgPath string, workDir string) ([]string, error) {
	goModPath, err := findGoMod(workDir)
	if err != nil {
		return nil, fmt.Errorf("go.mod not found: %w", err)
	}

	goModData, err := parseGoMod(goModPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod: %w", err)
	}

	modulePath := goModData.Module.Mod.Path
	moduleDir := filepath.Dir(goModPath)

	if !gosource.ModuleCarries(modulePath, pkgPath) {
		return nil, fmt.Errorf("package %s is external (not under module %s), not tracked in v1", pkgPath, modulePath)
	}

	relPath := strings.TrimPrefix(pkgPath, modulePath)
	relPath = strings.TrimPrefix(relPath, "/")
	pkgDir := filepath.Join(moduleDir, relPath)

	return listGoFiles(pkgDir)
}

func listGoFiles(dir string) ([]string, error) {
	return gosource.ListGoFiles(dir)
}

func DetectMockDependencies(input mcptypes.DetectMockDependenciesInput) (mcptypes.DetectDependenciesOutput, error) {
	var deps []mcptypes.Dependency

	configPath, err := findMockeryConfig(input.RootDir)
	if err != nil {
		return mcptypes.DetectDependenciesOutput{}, err
	}

	configDep, err := createFileDependency(configPath)
	if err != nil {
		return mcptypes.DetectDependenciesOutput{}, fmt.Errorf("failed to stat config file: %w", err)
	}
	deps = append(deps, configDep)

	goModPath, err := findGoMod(input.RootDir)
	if err == nil {
		goModDep, err := createFileDependency(goModPath)
		if err == nil {
			deps = append(deps, goModDep)
		}
	}

	config, err := parseMockeryConfig(configPath)
	if err != nil {
		return mcptypes.DetectDependenciesOutput{}, err
	}

	for pkgPath := range config.Packages {
		files, err := resolvePackageToFiles(pkgPath, input.RootDir)
		if err != nil {
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

func createFileDependency(path string) (mcptypes.Dependency, error) {
	digest, err := forge.DigestFile(path)
	if err != nil {
		return mcptypes.Dependency{}, err
	}

	return mcptypes.Dependency{Path: path, Digest: digest}, nil
}

func findGoMod(startPath string) (string, error) {
	dir := startPath
	if !filepath.IsAbs(dir) {
		var err error
		dir, err = filepath.Abs(dir)
		if err != nil {
			return "", err
		}
	}

	info, err := os.Stat(dir)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		dir = filepath.Dir(dir)
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return goModPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found in any parent directory of %s", startPath)
		}
		dir = parent
	}
}

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
