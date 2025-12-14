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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/alexandremahdhaoui/forge/internal/version"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
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

		// Step 1: Resolve engine URI to command and args using go run pattern
		// Use GetEffectiveVersion to handle both ldflags version and go run @version
		cmd, args, err := engineframework.ResolveDetector(detectorSpec.Engine, version.GetEffectiveVersion(Version))
		if err != nil {
			// Detector resolution failed - graceful degradation for this detector
			log.Printf("⚠ Dependency detector resolution failed: %v", err)
			log.Printf("   Skipping detector %s (continuing with others)", detectorSpec.Engine)
			continue
		}

		log.Printf("Resolved dependency detector: %s %v", cmd, args)

		// Step 2: Call detector with retry logic
		dependencies, err := callDependencyDetectorForContainer(cmd, args, spec, detectorSpec.Spec)
		if err != nil {
			// First retry
			log.Printf("⚠ Dependency detection failed (attempt 1/2): %v", err)
			log.Printf("   Retrying after 100ms...")
			time.Sleep(100 * time.Millisecond)

			dependencies, err = callDependencyDetectorForContainer(cmd, args, spec, detectorSpec.Spec)
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

// callDependencyDetectorForContainer calls a dependency detector MCP server for container builds.
// For containers, we pass the Containerfile path as the file to analyze.
func callDependencyDetectorForContainer(cmdName string, cmdArgs []string, spec forge.BuildSpec, detectorSpec map[string]interface{}) ([]mcptypes.Dependency, error) {
	// Create command to spawn MCP server (append --mcp flag)
	execCmd := exec.Command(cmdName, append(cmdArgs, "--mcp")...)
	execCmd.Env = os.Environ()
	execCmd.Stderr = os.Stderr // Forward logs

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "container-build-client",
		Version: "v1.0.0",
	}, nil)

	// Create command transport
	transport := &mcp.CommandTransport{
		Command: execCmd,
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
