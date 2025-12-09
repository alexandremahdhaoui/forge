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
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexandremahdhaoui/forge/internal/version"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

// detectDependenciesForArtifact detects dependencies for a built artifact if it's a main package.
// It updates the artifact in-place with detected dependencies.
//
// Error handling strategy:
//   - Detector not found: returns nil with log warning (graceful degradation)
//   - Detector found but fails: returns error after 1 retry (fail build)
//   - Not a main package: returns nil silently
func detectDependenciesForArtifact(src string, artifact *forge.Artifact) error {
	log.Printf("[DEBUG] detectDependenciesForArtifact called for: %s (artifact: %s)", src, artifact.Name)

	// Step 1: Check if this is a main package with main() function
	isMain, mainFile, err := findMainPackageFile(src)
	if err != nil {
		log.Printf("[DEBUG] findMainPackageFile returned error: %v", err)
		return fmt.Errorf("failed to detect main package: %w", err)
	}

	log.Printf("[DEBUG] findMainPackageFile result: isMain=%v, mainFile=%s", isMain, mainFile)

	if !isMain {
		// Not a main package, skip dependency detection silently
		log.Printf("[DEBUG] Not a main package, skipping dependency detection for %s", artifact.Name)
		return nil
	}

	log.Printf("Detected main package in %s, attempting dependency detection", mainFile)

	// Step 2: Resolve detector URI to command and args
	// Use GetEffectiveVersion to handle both ldflags version and go run @version
	cmd, args, err := engineframework.ResolveDetector("go://go-dependency-detector", version.GetEffectiveVersion(Version))
	if err != nil {
		// Resolution failed - graceful degradation
		log.Printf("WARNING: failed to resolve detector: %v", err)
		log.Printf("   Dependencies will not be tracked for %s (rebuild on every build)", artifact.Name)
		return nil
	}

	log.Printf("Resolved dependency detector: %s %v", cmd, args)

	// Step 3: Prepare input for detector
	input := map[string]any{
		"filePath": mainFile,
		"funcName": "main",
		"spec":     map[string]any{},
	}

	// Step 4: Call detector with retry logic (using shared helper)
	ctx := context.Background()
	dependencies, err := engineframework.CallDetector(ctx, cmd, args, "detectDependencies", input)
	if err != nil {
		// First retry
		log.Printf("WARNING: dependency detection failed (attempt 1/2): %v", err)
		log.Printf("   Retrying after 100ms...")
		time.Sleep(100 * time.Millisecond)

		dependencies, err = engineframework.CallDetector(ctx, cmd, args, "detectDependencies", input)
		if err != nil {
			// Second failure - fail the build
			return fmt.Errorf("dependency detection failed after retry: %w", err)
		}
	}

	// Step 5: Update artifact with dependencies
	artifact.Dependencies = dependencies
	artifact.DependencyDetectorEngine = "go://go-dependency-detector"
	artifact.DependencyDetectorSpec = make(map[string]interface{})

	log.Printf("Detected %d dependencies for %s", len(dependencies), artifact.Name)

	return nil
}

// findMainPackageFile checks if src contains a main package with main() function.
// Returns:
//   - isMain: true if main package with main() found
//   - mainFile: absolute path to file containing main() (if found)
//   - error: non-nil if directory can't be read
func findMainPackageFile(src string) (bool, string, error) {
	// Determine if src is a file or directory
	info, err := os.Stat(src)
	if err != nil {
		return false, "", fmt.Errorf("failed to stat %s: %w", src, err)
	}

	var searchDir string
	if info.IsDir() {
		searchDir = src
	} else {
		searchDir = filepath.Dir(src)
	}

	// Parse all .go files in directory
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, searchDir, func(fi os.FileInfo) bool {
		return filepath.Ext(fi.Name()) == ".go" && !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return false, "", fmt.Errorf("failed to parse directory %s: %w", searchDir, err)
	}

	// Check for main package
	mainPkg, hasMainPkg := pkgs["main"]
	if !hasMainPkg {
		return false, "", nil
	}

	// Find file with main() function
	for filePath, file := range mainPkg.Files {
		if hasMainFunc(file) {
			absPath, err := filepath.Abs(filePath)
			if err != nil {
				return false, "", fmt.Errorf("failed to resolve absolute path for %s: %w", filePath, err)
			}
			return true, absPath, nil
		}
	}

	return false, "", nil
}

// hasMainFunc checks if an AST file contains a main() function.
func hasMainFunc(file *ast.File) bool {
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if funcDecl.Name.Name == "main" && funcDecl.Recv == nil {
			return true
		}
	}
	return false
}
