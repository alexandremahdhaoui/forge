package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// detectDependenciesFromSpec detects dependencies by calling all configured dependency detectors.
//
// Error handling strategy:
//   - Detector not found: logs warning, skips that detector (graceful degradation)
//   - Detector found but fails: returns error after 1 retry (fail build)
//   - ParseDependsOn error: caller should log and skip detection
//   - Empty dependsOn: returns (nil, nil)
//
// Returns aggregated and deduplicated dependencies from all detectors.
func detectDependenciesFromSpec(dependsOn []forge.DependsOnSpec, spec forge.BuildSpec) ([]forge.ArtifactDependency, []string, error) {
	if len(dependsOn) == 0 {
		// No detectors configured - not an error
		return nil, nil, nil
	}

	var allDeps []forge.ArtifactDependency
	var detectorEngines []string

	for i, detectorSpec := range dependsOn {
		log.Printf("Running dependency detector %d/%d: %s", i+1, len(dependsOn), detectorSpec.Engine)

		// Step 1: Resolve engine URI to binary path
		detectorPath, err := findDependencyDetectorBinary(detectorSpec.Engine)
		if err != nil {
			// Detector not found - graceful degradation for this detector
			log.Printf("⚠ Dependency detector not found: %v", err)
			log.Printf("   Skipping detector %s (continuing with others)", detectorSpec.Engine)
			continue
		}

		log.Printf("Found dependency detector at: %s", detectorPath)

		// Step 2: Call detector with retry logic
		dependencies, err := callDependencyDetectorForContainer(detectorPath, spec, detectorSpec.Spec)
		if err != nil {
			// First retry
			log.Printf("⚠ Dependency detection failed (attempt 1/2): %v", err)
			log.Printf("   Retrying after 100ms...")
			time.Sleep(100 * time.Millisecond)

			dependencies, err = callDependencyDetectorForContainer(detectorPath, spec, detectorSpec.Spec)
			if err != nil {
				// Second failure - FAIL the build
				return nil, nil, fmt.Errorf("dependency detection failed after retry for %s: %w", detectorSpec.Engine, err)
			}
		}

		// Step 3: Convert mcptypes.Dependency to forge.ArtifactDependency
		for _, dep := range dependencies {
			artifactDep := forge.ArtifactDependency{
				Type:            dep.Type,
				FilePath:        dep.FilePath,
				ExternalPackage: dep.ExternalPackage,
				Timestamp:       dep.Timestamp,
				Semver:          dep.Semver,
			}
			allDeps = append(allDeps, artifactDep)
		}

		// Track this detector engine
		detectorEngines = append(detectorEngines, detectorSpec.Engine)

		log.Printf("✅ Detected %d dependencies from %s", len(dependencies), detectorSpec.Engine)
	}

	// Step 4: Deduplicate dependencies
	deduplicated := deduplicateDependencies(allDeps)

	log.Printf("✅ Total: %d unique dependencies from %d detectors", len(deduplicated), len(detectorEngines))

	return deduplicated, detectorEngines, nil
}

// deduplicateDependencies removes duplicate dependencies based on (Type, FilePath, ExternalPackage).
func deduplicateDependencies(deps []forge.ArtifactDependency) []forge.ArtifactDependency {
	seen := make(map[string]bool)
	result := make([]forge.ArtifactDependency, 0, len(deps))

	for _, dep := range deps {
		// Create unique key based on type and identifier
		var key string
		if dep.Type == "file" {
			key = fmt.Sprintf("file:%s", dep.FilePath)
		} else {
			key = fmt.Sprintf("external:%s", dep.ExternalPackage)
		}

		if !seen[key] {
			seen[key] = true
			result = append(result, dep)
		}
	}

	return result
}

// findDependencyDetectorBinary locates a dependency detector binary from engine URI.
// Supports:
//   - go://go-dependency-detector -> searches for "go-dependency-detector" binary
//   - Other go:// URIs -> extracts binary name from last path component
func findDependencyDetectorBinary(engineURI string) (string, error) {
	// Parse engine URI to extract binary name
	// For "go://go-dependency-detector" -> "go-dependency-detector"
	// For "go://github.com/user/repo/cmd/detector" -> "detector"
	binaryName := extractBinaryNameFromURI(engineURI)
	if binaryName == "" {
		return "", fmt.Errorf("could not extract binary name from engine URI: %s", engineURI)
	}

	// Try to find in PATH
	path, err := exec.LookPath(binaryName)
	if err == nil {
		return path, nil
	}

	// Try in build directory (common for forge self-build)
	buildPath := filepath.Join("./build/bin", binaryName)
	if _, err := os.Stat(buildPath); err == nil {
		absPath, err := filepath.Abs(buildPath)
		if err != nil {
			return "", fmt.Errorf("found detector at %s but failed to resolve absolute path: %w", buildPath, err)
		}
		return absPath, nil
	}

	return "", fmt.Errorf("%s not found in PATH or ./build/bin", binaryName)
}

// extractBinaryNameFromURI extracts the binary name from an engine URI.
// Examples:
//   - "go://go-dependency-detector" -> "go-dependency-detector"
//   - "go://github.com/user/repo/cmd/detector" -> "detector"
func extractBinaryNameFromURI(uri string) string {
	// Remove "go://" prefix
	if len(uri) > 5 && uri[:5] == "go://" {
		uri = uri[5:]
	}

	// If it's a simple name (no slashes), return as-is
	if filepath.Base(uri) == uri {
		return uri
	}

	// Extract last path component
	return filepath.Base(uri)
}

// callDependencyDetectorForContainer calls a dependency detector MCP server for container builds.
// For containers, we pass the Containerfile path as the file to analyze.
func callDependencyDetectorForContainer(detectorPath string, spec forge.BuildSpec, detectorSpec map[string]interface{}) ([]mcptypes.Dependency, error) {
	// Create command to spawn MCP server
	cmd := exec.Command(detectorPath, "--mcp")
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr // Forward logs

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "container-build-client",
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

	// Prepare input - for containers, analyze the Containerfile/Dockerfile
	input := map[string]any{
		"filePath": spec.Src, // Containerfile path
		"funcName": "",       // Not applicable for containers
		"spec":     detectorSpec,
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
