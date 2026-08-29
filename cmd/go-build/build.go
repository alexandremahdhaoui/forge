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
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/engineversion"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// ----------------------------------------------------- BUILD (MCP) -------------------------------------------------- //

// Build implements the BuildFunc for building Go binaries (MCP mode)
func Build(ctx context.Context, input mcptypes.BuildInput, spec *Spec) (*forge.Artifact, error) {
	log.Printf("Building binary: %s from %s", input.Name, input.Src)

	// Use spec values for custom args and env, falling back to input values
	customArgs := spec.Args
	if len(customArgs) == 0 {
		customArgs = input.Args
	}

	customEnv := spec.Env
	if len(customEnv) == 0 {
		customEnv = input.Env
	}

	// Determine destination directory
	dest := input.Dest
	if dest == "" {
		dest = "./build/bin"
	}

	// Create destination directory
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	outputPath := filepath.Join(dest, input.Name)

	// The build environment is scoped to this command, never the process:
	// batch builds run in parallel, so setting GOOS on the process would
	// leak into a sibling build and cross-compile the wrong artifact.
	// CGO off keeps the binary static; custom env may override it.
	buildEnv := append(os.Environ(), "CGO_ENABLED=0")

	// A cross build is one that names a target: it gets the distribution
	// treatment - stripped, trimmed - because it is built to travel.
	cross := false

	for key, value := range customEnv {
		if key == "GOOS" || key == "GOARCH" {
			cross = true
		}

		buildEnv = append(buildEnv, key+"="+value)
	}

	// Build command arguments. -trimpath keeps absolute build paths out of
	// the binary, so the same source builds the same bytes anywhere.
	args := []string{
		"build",
		"-trimpath",
		"-o", outputPath,
	}

	args = append(args, "-ldflags", buildLDFlags(cross))

	// Add custom args if provided
	args = append(args, customArgs...)

	// Add source path
	args = append(args, input.Src)

	// Execute build
	cmd := exec.Command("go", args...)
	cmd.Env = buildEnv
	cmd.Stdout = os.Stderr // MCP mode: redirect to stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go build failed: %w", err)
	}

	// Create versioned artifact
	artifact, err := engineframework.CreateVersionedArtifact(
		input.Name,
		"binary",
		outputPath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create artifact: %w", err)
	}

	// Detect dependencies if this is a main package
	if err := detectDependenciesForArtifact(input.Src, artifact); err != nil {
		return nil, fmt.Errorf("failed to detect dependencies: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Built binary: %s (version: %s)\n", input.Name, artifact.Version)

	return artifact, nil
}

// ----------------------------------------------------- DEPENDENCY DETECTION ---------------------------------------- //

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
	cmd, args, err := engineframework.ResolveDetector("forge://go-dependency-detector", engineversion.GetEffectiveVersion(Version))
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
	artifact.DependencyDetectorEngine = "forge://go-dependency-detector"
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
	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return false, "", fmt.Errorf("failed to read directory %s: %w", searchDir, err)
	}

	fset := token.NewFileSet()

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}

		filePath := filepath.Join(searchDir, name)

		file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
		if err != nil {
			return false, "", fmt.Errorf("failed to parse %s: %w", filePath, err)
		}

		if file.Name.Name != "main" {
			continue
		}

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

// buildLDFlags composes the link flags every go-build produces. The version
// label a binary reports comes from git, so `<tool> version` never lies
// about which build it is; a -X against a symbol a command does not carry
// is ignored by the linker, so one line serves every command. A cross build
// is stripped, because it is built to travel rather than to be debugged.
// GO_BUILD_LDFLAGS, when set, is appended and therefore wins.
func buildLDFlags(cross bool) string {
	flags := []string{}

	if cross {
		flags = append(flags, "-s", "-w")
	}

	if label := gitLabel(); label != "" {
		flags = append(flags, "-X", "main.Version="+label, "-X", "main.version="+label)
	}

	// The revision the pipeline proved, handed to every compute target: a
	// released binary knows the distribution it shipped with, which is what
	// lets it find its companions in the store with no PATH and no network.
	if revision := os.Getenv("FORGE_CI_REVISION"); revision != "" {
		flags = append(flags, "-X",
			"github.com/alexandremahdhaoui/forge/pkg/toolresolver.CompanionRevision="+revision)
	}

	if extra := os.Getenv("GO_BUILD_LDFLAGS"); extra != "" {
		flags = append(flags, extra)
	}

	return strings.Join(flags, " ")
}

// gitLabel is the human name of this build. The pipeline's version wins when
// there is one, because that is the number the release will actually carry:
// three sources used to compete here, and a binary that reports a different
// version from the release it shipped in is a lie the operator acts on.
//
// With no pipeline, the nearest tag on the plain semver line, else the sha.
// --match is what keeps a namespaced tag out of it: a repo released by two
// factories carries "forge-v0.50.0" alongside "v0.49.0", and git describe
// with no match returns whichever is nearest, so the label read as a version
// nobody could pin. A tree that is not a repo has no label and the stamp is
// simply omitted.
func gitLabel() string {
	if version := os.Getenv("FORGE_CI_VERSION"); version != "" {
		return version
	}

	out, err := exec.Command("git", "describe", "--tags", "--always", "--match", "v[0-9]*").Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}
