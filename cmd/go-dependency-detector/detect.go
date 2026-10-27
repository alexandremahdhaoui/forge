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
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexandremahdhaoui/forge/internal/gosource"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"golang.org/x/mod/modfile"
)

type dependencyTracker struct {
	visitedFiles    map[string]bool
	visitedPackages map[string]bool
	goModPath       string
	goModData       *modfile.File
	dependencies    []mcptypes.Dependency
	moduleDir       string
	modulePath      string
}

func DetectDependencies(input mcptypes.DetectDependenciesInput) (mcptypes.DetectDependenciesOutput, error) {
	absFilePath, err := filepath.Abs(input.FilePath)
	if err != nil {
		return mcptypes.DetectDependenciesOutput{}, fmt.Errorf("failed to resolve absolute path for %s: %w", input.FilePath, err)
	}

	goModPath, err := findGoMod(absFilePath)
	if err != nil {
		return mcptypes.DetectDependenciesOutput{}, fmt.Errorf("go.mod not found: %w", err)
	}

	goModData, err := parseGoMod(goModPath)
	if err != nil {
		return mcptypes.DetectDependenciesOutput{}, fmt.Errorf("failed to parse go.mod: %w", err)
	}

	manifest, err := forge.DependencyOf(goModPath)
	if err != nil {
		return mcptypes.DetectDependenciesOutput{}, err
	}

	seed := []mcptypes.Dependency{{Path: manifest.Path, Digest: manifest.Digest}}

	goSumPath := filepath.Join(filepath.Dir(goModPath), "go.sum")
	if _, err := os.Stat(goSumPath); err == nil {
		lock, err := forge.DependencyOf(goSumPath)
		if err != nil {
			return mcptypes.DetectDependenciesOutput{}, err
		}

		seed = append(seed, mcptypes.Dependency{Path: lock.Path, Digest: lock.Digest})
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, absFilePath, nil, parser.ParseComments)
	if err != nil {
		return mcptypes.DetectDependenciesOutput{}, fmt.Errorf("failed to parse file %s: %w", absFilePath, err)
	}

	funcFound := false
	ast.Inspect(file, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok && fn.Name.Name == input.FuncName {
			funcFound = true
		}
		return true
	})
	if !funcFound {
		return mcptypes.DetectDependenciesOutput{}, fmt.Errorf("function %s not found in %s", input.FuncName, absFilePath)
	}

	tracker := &dependencyTracker{
		visitedFiles:    make(map[string]bool),
		visitedPackages: make(map[string]bool),
		goModPath:       goModPath,
		goModData:       goModData,
		dependencies:    seed,
		moduleDir:       filepath.Dir(goModPath),
		modulePath:      goModData.Module.Mod.Path,
	}

	err = tracker.processPackage(filepath.Dir(absFilePath))
	if err != nil {
		return mcptypes.DetectDependenciesOutput{}, err
	}

	return mcptypes.DetectDependenciesOutput{
		Dependencies: tracker.dependencies,
	}, nil
}

func (t *dependencyTracker) processPackage(dir string) error {
	files, err := gosource.ListGoFiles(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := t.processFile(file); err != nil {
			return err
		}
	}

	return nil
}

func (t *dependencyTracker) processFile(filePath string) error {
	if t.visitedFiles[filePath] {
		return nil
	}
	t.visitedFiles[filePath] = true

	if filePath != t.goModPath {
		digest, err := forge.DigestFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to digest %s: %v\n", filePath, err)
		} else {
			t.dependencies = append(t.dependencies, mcptypes.Dependency{Path: filePath, Digest: digest})
		}
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", filePath, err)
		return nil
	}

	imports := extractImports(file)

	for _, importPath := range imports {
		if isStandardLibrary(importPath) {
			continue
		}

		if t.visitedPackages[importPath] {
			continue
		}

		isLocal := isLocalPackage(importPath, t.modulePath, t.goModData)

		if isLocal {
			pkgDir, err := resolveLocalPackage(t.goModData, importPath, t.moduleDir, t.modulePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to resolve local package %s: %v\n", importPath, err)
				continue
			}

			t.visitedPackages[importPath] = true

			err = t.processPackage(pkgDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to process package %s: %v\n", importPath, err)
			}
		} else {
			if _, err := getPackageVersion(t.goModData, importPath); err != nil {
				return fmt.Errorf("package %s not found in go.mod: %w", importPath, err)
			}

			t.visitedPackages[importPath] = true
		}
	}

	return nil
}

func extractImports(file *ast.File) []string {
	var imports []string
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		imports = append(imports, importPath)
	}
	return imports
}

func isStandardLibrary(pkgPath string) bool {
	if pkgPath == "C" {
		return false
	}

	firstSegment := pkgPath
	if idx := strings.Index(pkgPath, "/"); idx != -1 {
		firstSegment = pkgPath[:idx]
	}

	return !strings.Contains(firstSegment, ".")
}

func isLocalPackage(importPath, modulePath string, goModData *modfile.File) bool {
	if gosource.ModuleCarries(modulePath, importPath) {
		return true
	}

	for _, replace := range goModData.Replace {
		if replace.Old.Path == importPath {
			if strings.HasPrefix(replace.New.Path, ".") || strings.HasPrefix(replace.New.Path, "/") {
				return true
			}
			if gosource.ModuleCarries(modulePath, replace.New.Path) {
				return true
			}
		}
	}

	return false
}

func resolveLocalPackage(goModData *modfile.File, importPath, goModDir, modulePath string) (string, error) {
	for _, replace := range goModData.Replace {
		if replace.Old.Path == importPath {
			replacePath := replace.New.Path
			if strings.HasPrefix(replacePath, ".") {
				replacePath = filepath.Join(goModDir, replacePath)
			}

			return replacePath, nil
		}
	}

	if !gosource.ModuleCarries(modulePath, importPath) {
		return "", fmt.Errorf("import path %s is not under module path %s", importPath, modulePath)
	}

	relPath := strings.TrimPrefix(importPath, modulePath)
	relPath = strings.TrimPrefix(relPath, "/")

	return filepath.Join(goModDir, relPath), nil
}

func getPackageVersion(goModData *modfile.File, pkgPath string) (string, error) {
	for _, req := range goModData.Require {
		if gosource.ModuleCarries(req.Mod.Path, pkgPath) {
			return req.Mod.Version, nil
		}
	}

	for _, replace := range goModData.Replace {
		if replace.Old.Path == pkgPath {
			if replace.New.Version != "" {
				return replace.New.Version, nil
			}
		}
	}

	return "", fmt.Errorf("package %s not found in go.mod", pkgPath)
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
