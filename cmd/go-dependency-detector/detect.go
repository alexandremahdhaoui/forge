package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"golang.org/x/mod/modfile"
)

// dependencyTracker tracks visited files/packages and accumulates dependencies.
type dependencyTracker struct {
	visitedFiles    map[string]bool // Absolute file paths
	visitedPackages map[string]bool // Import paths
	goModPath       string          // Absolute path to go.mod
	goModData       *modfile.File   // Parsed go.mod
	dependencies    []mcptypes.Dependency
	moduleDir       string // Directory containing go.mod
	modulePath      string // Module name from go.mod
}

// DetectDependencies detects all dependencies for a given Go function.
// It recursively analyzes all transitive dependencies (local and external packages).
func DetectDependencies(input mcptypes.DetectDependenciesInput) (mcptypes.DetectDependenciesOutput, error) {
	// Step 1: Convert input.FilePath to absolute path
	absFilePath, err := filepath.Abs(input.FilePath)
	if err != nil {
		return mcptypes.DetectDependenciesOutput{}, fmt.Errorf("failed to resolve absolute path for %s: %w", input.FilePath, err)
	}

	// Step 2: Find and parse go.mod file
	goModPath, err := findGoMod(absFilePath)
	if err != nil {
		return mcptypes.DetectDependenciesOutput{}, fmt.Errorf("go.mod not found: %w", err)
	}

	goModData, err := parseGoMod(goModPath)
	if err != nil {
		return mcptypes.DetectDependenciesOutput{}, fmt.Errorf("failed to parse go.mod: %w", err)
	}

	// Get go.mod timestamp
	goModInfo, err := os.Stat(goModPath)
	if err != nil {
		return mcptypes.DetectDependenciesOutput{}, fmt.Errorf("failed to stat go.mod: %w", err)
	}
	goModTimestamp := goModInfo.ModTime().UTC().Format(time.RFC3339)

	// Step 3: Parse the Go file at input.FilePath
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, absFilePath, nil, parser.ParseComments)
	if err != nil {
		return mcptypes.DetectDependenciesOutput{}, fmt.Errorf("failed to parse file %s: %w", absFilePath, err)
	}

	// Step 4: Find the function specified by input.FuncName
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

	// Step 5: Initialize dependencyTracker with go.mod as first dependency
	tracker := &dependencyTracker{
		visitedFiles:    make(map[string]bool),
		visitedPackages: make(map[string]bool),
		goModPath:       goModPath,
		goModData:       goModData,
		dependencies: []mcptypes.Dependency{
			{
				Type:      "file",
				FilePath:  goModPath,
				Timestamp: goModTimestamp,
			},
		},
		moduleDir:  filepath.Dir(goModPath),
		modulePath: goModData.Module.Mod.Path,
	}

	// Step 6: Recursively traverse all imports (transitive dependencies)
	err = tracker.processFile(absFilePath)
	if err != nil {
		return mcptypes.DetectDependenciesOutput{}, err
	}

	// Step 7: Return DependencyDetectorOutput with all collected dependencies
	return mcptypes.DetectDependenciesOutput{
		Dependencies: tracker.dependencies,
	}, nil
}

// processFile recursively processes a Go file and its imports.
func (t *dependencyTracker) processFile(filePath string) error {
	// Prevent cycles
	if t.visitedFiles[filePath] {
		return nil
	}
	t.visitedFiles[filePath] = true

	// Parse the file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		// Log warning but continue (as per spec)
		fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", filePath, err)
		return nil
	}

	// Extract imports
	imports := extractImports(file)

	// Process each import
	for _, importPath := range imports {
		// Skip standard library
		if isStandardLibrary(importPath) {
			continue
		}

		// Check if already visited
		if t.visitedPackages[importPath] {
			continue
		}

		// Determine if this is a local or external package
		isLocal := isLocalPackage(importPath, t.modulePath, t.goModData)

		if isLocal {
			// Local package - resolve to file path and recurse
			localFilePath, err := resolveLocalPackage(t.goModData, importPath, t.moduleDir, t.modulePath)
			if err != nil {
				// Log warning but continue
				fmt.Fprintf(os.Stderr, "Warning: failed to resolve local package %s: %v\n", importPath, err)
				continue
			}

			// Get absolute path
			absLocalPath, err := filepath.Abs(localFilePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to get absolute path for %s: %v\n", localFilePath, err)
				continue
			}

			// Skip if already visited
			if t.visitedFiles[absLocalPath] {
				continue
			}

			// Get file timestamp
			timestamp, err := getFileTimestamp(absLocalPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to get timestamp for %s: %v\n", absLocalPath, err)
				continue
			}

			// Add to dependencies
			t.dependencies = append(t.dependencies, mcptypes.Dependency{
				Type:      "file",
				FilePath:  absLocalPath,
				Timestamp: timestamp,
			})

			// Mark package as visited
			t.visitedPackages[importPath] = true

			// Recurse into the local package
			err = t.processFile(absLocalPath)
			if err != nil {
				// Log warning but continue
				fmt.Fprintf(os.Stderr, "Warning: failed to process file %s: %v\n", absLocalPath, err)
			}
		} else {
			// External package - get version from go.mod
			version, err := getPackageVersion(t.goModData, importPath)
			if err != nil {
				return fmt.Errorf("package %s not found in go.mod: %w", importPath, err)
			}

			// Add to dependencies
			t.dependencies = append(t.dependencies, mcptypes.Dependency{
				Type:            "externalPackage",
				ExternalPackage: importPath,
				Semver:          version,
			})

			// Mark package as visited
			t.visitedPackages[importPath] = true
		}
	}

	return nil
}

// extractImports extracts all import paths from a Go file.
func extractImports(file *ast.File) []string {
	var imports []string
	for _, imp := range file.Imports {
		// Remove quotes from import path
		importPath := strings.Trim(imp.Path.Value, `"`)
		imports = append(imports, importPath)
	}
	return imports
}

// isStandardLibrary checks if a package is in the Go standard library.
// Algorithm: Check if import path does NOT contain a dot in first segment.
func isStandardLibrary(pkgPath string) bool {
	// Special case: "C" is NOT stdlib (cgo)
	if pkgPath == "C" {
		return false
	}

	// Check first segment for dot
	firstSegment := pkgPath
	if idx := strings.Index(pkgPath, "/"); idx != -1 {
		firstSegment = pkgPath[:idx]
	}

	// If first segment contains a dot, it's not stdlib
	return !strings.Contains(firstSegment, ".")
}

// isLocalPackage determines if an import is a local package (vs external).
func isLocalPackage(importPath, modulePath string, goModData *modfile.File) bool {
	// Check if it's under the module path
	if strings.HasPrefix(importPath, modulePath) {
		return true
	}

	// Check if there's a replace directive pointing to a local path
	for _, replace := range goModData.Replace {
		if replace.Old.Path == importPath {
			// If new path is a relative path, it's local
			if strings.HasPrefix(replace.New.Path, ".") || strings.HasPrefix(replace.New.Path, "/") {
				return true
			}
			// If new path is under module path, it's local
			if strings.HasPrefix(replace.New.Path, modulePath) {
				return true
			}
		}
	}

	return false
}

// resolveLocalPackage resolves a local import path to a file path.
func resolveLocalPackage(goModData *modfile.File, importPath, goModDir, modulePath string) (string, error) {
	// Check for replace directive
	for _, replace := range goModData.Replace {
		if replace.Old.Path == importPath {
			// Use replacement path
			replacePath := replace.New.Path
			if strings.HasPrefix(replacePath, ".") {
				// Relative path
				replacePath = filepath.Join(goModDir, replacePath)
			}
			// Find first .go file in the package directory
			return findFirstGoFile(replacePath)
		}
	}

	// Construct path relative to module root
	if !strings.HasPrefix(importPath, modulePath) {
		return "", fmt.Errorf("import path %s is not under module path %s", importPath, modulePath)
	}

	// Remove module path prefix to get relative path
	relPath := strings.TrimPrefix(importPath, modulePath)
	relPath = strings.TrimPrefix(relPath, "/")

	// Construct full path
	pkgDir := filepath.Join(goModDir, relPath)

	// Find first .go file in the package directory
	return findFirstGoFile(pkgDir)
}

// findFirstGoFile finds the first .go file in a directory.
func findFirstGoFile(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
			return filepath.Join(dir, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("no .go files found in directory %s", dir)
}

// getPackageVersion extracts the version of an external package from go.mod.
func getPackageVersion(goModData *modfile.File, pkgPath string) (string, error) {
	// Search in require directives
	for _, req := range goModData.Require {
		if req.Mod.Path == pkgPath {
			return req.Mod.Version, nil
		}
		// Handle subpackages (e.g., github.com/foo/bar/baz should match github.com/foo/bar)
		if strings.HasPrefix(pkgPath, req.Mod.Path+"/") {
			return req.Mod.Version, nil
		}
	}

	// Check replace directives
	for _, replace := range goModData.Replace {
		if replace.Old.Path == pkgPath {
			if replace.New.Version != "" {
				return replace.New.Version, nil
			}
		}
	}

	return "", fmt.Errorf("package %s not found in go.mod", pkgPath)
}

// getFileTimestamp returns the modification timestamp of a file in RFC3339 UTC format.
func getFileTimestamp(filePath string) (string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}
	return info.ModTime().UTC().Format(time.RFC3339), nil
}

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
