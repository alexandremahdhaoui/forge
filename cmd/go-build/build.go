package main

import (
	"context"
	"encoding/json"
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

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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

	// Step 2: Check if go-dependency-detector is available
	detectorPath, err := findDependencyDetector()
	if err != nil {
		// Detector not found - graceful degradation
		log.Printf("⚠ Dependency detector not found: %v", err)
		log.Printf("   Dependencies will not be tracked for %s (rebuild on every build)", artifact.Name)
		return nil
	}

	log.Printf("Found dependency detector at: %s", detectorPath)

	// Step 3: Call detector with retry logic
	dependencies, err := callDependencyDetector(detectorPath, mainFile)
	if err != nil {
		// First retry
		log.Printf("⚠ Dependency detection failed (attempt 1/2): %v", err)
		log.Printf("   Retrying after 100ms...")
		time.Sleep(100 * time.Millisecond)

		dependencies, err = callDependencyDetector(detectorPath, mainFile)
		if err != nil {
			// Second failure - fail the build
			return fmt.Errorf("dependency detection failed after retry: %w", err)
		}
	}

	// Step 4: Convert mcptypes.Dependency to forge.ArtifactDependency
	artifactDeps := make([]forge.ArtifactDependency, len(dependencies))
	for i, dep := range dependencies {
		artifactDeps[i] = forge.ArtifactDependency{
			Type:            dep.Type,
			FilePath:        dep.FilePath,
			ExternalPackage: dep.ExternalPackage,
			Timestamp:       dep.Timestamp,
			Semver:          dep.Semver,
		}
	}

	// Step 5: Update artifact with dependencies
	artifact.Dependencies = artifactDeps
	artifact.DependencyDetectorEngine = "go://go-dependency-detector"
	artifact.DependencyDetectorSpec = make(map[string]interface{})

	log.Printf("✅ Detected %d dependencies for %s", len(artifactDeps), artifact.Name)

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

// findDependencyDetector locates the go-dependency-detector binary.
// Returns the absolute path to the binary or an error if not found.
func findDependencyDetector() (string, error) {
	// Try to find in PATH
	path, err := exec.LookPath("go-dependency-detector")
	if err == nil {
		return path, nil
	}

	// Try in build directory (common for forge self-build)
	buildPath := "./build/bin/go-dependency-detector"
	if _, err := os.Stat(buildPath); err == nil {
		absPath, err := filepath.Abs(buildPath)
		if err != nil {
			return "", fmt.Errorf("found detector at %s but failed to resolve absolute path: %w", buildPath, err)
		}
		return absPath, nil
	}

	return "", fmt.Errorf("go-dependency-detector not found in PATH or ./build/bin")
}

// callDependencyDetector calls the dependency detector MCP server to detect dependencies.
func callDependencyDetector(detectorPath, mainFilePath string) ([]mcptypes.Dependency, error) {
	// Create command to spawn MCP server
	cmd := exec.Command(detectorPath, "--mcp")
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr // Forward logs

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "go-build-client",
		Version: "v1.0.0",
	}, nil)

	// Create command transport
	transport := &mcp.CommandTransport{
		Command: cmd,
	}

	// Connect to the MCP server
	ctx := context.Background()
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to dependency detector: %w", err)
	}
	defer func() { _ = session.Close() }()

	// Prepare input
	input := map[string]any{
		"filePath": mainFilePath,
		"funcName": "main",
		"spec":     map[string]any{},
	}

	// Call the detectDependencies tool
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "detectDependencies",
		Arguments: input,
	})
	if err != nil {
		return nil, fmt.Errorf("MCP tool call failed: %w", err)
	}

	// Check if result indicates an error
	if result.IsError {
		errMsg := "unknown error"
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
				errMsg = textContent.Text
			}
		}
		return nil, fmt.Errorf("dependency detection failed: %s", errMsg)
	}

	// Parse structured content
	if result.StructuredContent == nil {
		return nil, fmt.Errorf("no structured content returned from detector")
	}

	// Convert structured content to DetectDependenciesOutput
	var output mcptypes.DetectDependenciesOutput
	jsonBytes, err := json.Marshal(result.StructuredContent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal detector output: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, &output); err != nil {
		return nil, fmt.Errorf("failed to unmarshal detector output: %w", err)
	}

	return output.Dependencies, nil
}
