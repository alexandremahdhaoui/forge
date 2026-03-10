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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexandremahdhaoui/forge/internal/orchestrate"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

// BuildAllResult contains the outcome of a build operation.
// Both CLI and MCP use this struct to communicate build results.
type BuildAllResult struct {
	Artifacts   []forge.Artifact
	Skipped     int
	TotalBuilt  int
	BuildErrors []string
}

// buildAll loads config, resolves context, checks lazy rebuild,
// dispatches to engines, and updates the artifact store.
// Both CLI and MCP call this function.
//
// artifactName filters to a single artifact if non-empty.
// forceRebuild bypasses lazy rebuild checks.
//
// This function MUST NOT write to stdout. Stdout is the JSON-RPC
// transport in MCP mode. Progress messages go to stderr.
func buildAll(artifactName string, forceRebuild bool) (*BuildAllResult, error) {
	// Load forge.yaml configuration
	config, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load forge.yaml: %w", err)
	}

	// Read artifact store
	store, err := forge.ReadOrCreateArtifactStore(config.ArtifactStorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read artifact store: %w", err)
	}

	// Group specs by engine
	engineSpecs := make(map[string][]map[string]any)
	result := &BuildAllResult{}

	// Track cleanup functions for git-cloned context directories.
	// Temp dirs must persist until all engine builds complete because specs
	// are grouped by engine and sent as a batch -- cleanup after all builds.
	var cleanups []func()
	defer func() {
		for _, fn := range cleanups {
			fn()
		}
	}()

	for _, spec := range config.Build {
		// Filter by artifact name if provided
		if artifactName != "" && spec.Name != artifactName {
			continue
		}

		// Check if rebuild is needed (lazy rebuild logic)
		needsRebuild, reason, err := shouldRebuild(spec.Name, store, forceRebuild)
		if err != nil {
			// If error checking rebuild status, log warning and rebuild (safe default)
			fmt.Fprintf(os.Stderr, "Warning: failed to check rebuild status for %s: %v (will rebuild)\n", spec.Name, err)
			needsRebuild = true
			reason = "rebuild check failed"
		}

		if !needsRebuild {
			// Skip this artifact - it's up to date
			fmt.Fprintf(os.Stderr, "⏭  Skipping %s (unchanged)\n", spec.Name)
			result.Skipped++
			continue
		}

		// Log reason for rebuild if provided
		if reason != "" {
			fmt.Fprintf(os.Stderr, "🔨 Building %s (%s)\n", spec.Name, reason)
		}

		// Normalize engine URI and warn if deprecated
		normalizedEngine, wasDeprecated := normalizeEngineURI(spec.Engine)
		if wasDeprecated {
			fmt.Fprintf(os.Stderr,
				"⚠️  DEPRECATED: %s is deprecated, use %s instead (in spec: %s)\n",
				spec.Engine, normalizedEngine, spec.Name)
		}

		// Use the normalized engine
		engine := normalizedEngine

		// Resolve build context
		contextDir, cleanup, err := resolveContextDir(spec.Context)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve context for %s: %w", spec.Name, err)
		}
		cleanups = append(cleanups, cleanup)

		// Resolve src relative to context directory.
		// Only join if spec.Src is relative AND context is explicitly set.
		resolvedSrc := spec.Src
		if spec.Context != "" && spec.Context != "." && !filepath.IsAbs(spec.Src) {
			resolvedSrc = filepath.Join(contextDir, spec.Src)
		}

		params := map[string]any{
			"name":    spec.Name,
			"src":     resolvedSrc,
			"dest":    spec.Dest,
			"context": contextDir,
			"engine":  engine,
		}

		// Pass engine-specific configuration if provided
		// Nest under "spec" key so engines can access it via BuildInput.Spec
		if len(spec.Spec) > 0 {
			params["spec"] = spec.Spec
		}

		engineSpecs[engine] = append(engineSpecs[engine], params)
	}

	if len(engineSpecs) == 0 {
		if artifactName != "" && result.Skipped == 0 {
			return nil, fmt.Errorf("no artifact found with name: %s", artifactName)
		}
		// Either all skipped or no artifacts to build
		return result, nil
	}

	// Create forge directories for build operations
	dirs, err := createForgeDirs()
	if err != nil {
		return nil, fmt.Errorf("failed to create forge directories: %w", err)
	}

	// Clean up old tmp directories (keep last 10 runs)
	if err := cleanupOldTmpDirs(10); err != nil {
		// Log warning but don't fail
		fmt.Fprintf(os.Stderr, "Warning: failed to cleanup old tmp directories: %v\n", err)
	}

	// Build each group using the appropriate engine
	for engineURI, specs := range engineSpecs {
		fmt.Fprintf(os.Stderr, "Building %d artifact(s) with %s...\n", len(specs), engineURI)

		// Check if this is a multi-engine alias
		var artifacts []forge.Artifact
		if strings.HasPrefix(engineURI, "alias://") {
			aliasName := strings.TrimPrefix(engineURI, "alias://")
			engineConfig := getEngineConfig(aliasName, &config)
			if engineConfig == nil {
				result.BuildErrors = append(result.BuildErrors, fmt.Sprintf("engine alias not found: %s", aliasName))
				continue
			}

			if engineConfig.Type == forge.BuilderEngineConfigType && len(engineConfig.Builder) > 1 {
				// Multi-engine builder - use orchestrator
				fmt.Fprintf(os.Stderr, "  Multi-engine builder detected (%d engines)\n", len(engineConfig.Builder))

				orchestrator := orchestrate.NewBuilderOrchestrator(
					callMCPEngine,
					func(uri string) (string, []string, error) {
						return resolveEngine(uri, &config)
					},
				)

				// Prepare directories map
				dirsMap := map[string]any{
					"tmpDir":   dirs.TmpDir,
					"buildDir": dirs.BuildDir,
					"rootDir":  dirs.RootDir,
				}

				// Execute orchestration
				artifacts, err = orchestrator.Orchestrate(engineConfig.Builder, specs, dirsMap)
				if err != nil {
					result.BuildErrors = append(result.BuildErrors, fmt.Sprintf("multi-engine build failed: %v", err))
					continue
				}
			} else {
				// Single-engine alias - resolve to actual engine
				command, args, err := resolveEngine(engineURI, &config)
				if err != nil {
					result.BuildErrors = append(result.BuildErrors, fmt.Sprintf("failed to resolve engine %s: %v", engineURI, err))
					continue
				}

				artifacts, err = buildWithSingleEngine(command, args, specs, dirs, engineConfig, forceRebuild)
				if err != nil {
					result.BuildErrors = append(result.BuildErrors, fmt.Sprintf("build failed for %s: %v", engineURI, err))
					continue
				}
			}
		} else {
			// Direct go:// URI - single engine
			command, args, err := resolveEngine(engineURI, &config)
			if err != nil {
				result.BuildErrors = append(result.BuildErrors, fmt.Sprintf("failed to resolve engine %s: %v", engineURI, err))
				continue
			}

			artifacts, err = buildWithSingleEngine(command, args, specs, dirs, nil, forceRebuild)
			if err != nil {
				result.BuildErrors = append(result.BuildErrors, fmt.Sprintf("build failed for %s: %v", engineURI, err))
				continue
			}
		}

		// Update artifact store
		for _, artifact := range artifacts {
			forge.AddOrUpdateArtifact(&store, artifact)
			result.Artifacts = append(result.Artifacts, artifact)
			result.TotalBuilt++
		}
	}

	// Write updated artifact store
	if err := forge.WriteArtifactStore(config.ArtifactStorePath, store); err != nil {
		return nil, fmt.Errorf("failed to write artifact store: %w", err)
	}

	// If there were build errors, return them as an error
	if len(result.BuildErrors) > 0 {
		return result, fmt.Errorf("build completed with errors: %v", result.BuildErrors)
	}

	return result, nil
}

// normalizeEngineURI maps deprecated engine URIs to their current equivalents.
// Returns the normalized URI and whether a deprecated URI was used.
func normalizeEngineURI(uri string) (string, bool) {
	deprecated := map[string]string{
		"go://build-container": "go://container-build",
	}

	if newURI, ok := deprecated[uri]; ok {
		return newURI, true // deprecated
	}

	return uri, false // not deprecated
}

// shouldRebuild determines if an artifact needs to be rebuilt based on its dependencies.
// Returns (needsRebuild bool, reason string, error).
// If forceRebuild is true, always returns (true, "force flag set", nil).
// Otherwise, checks if dependencies have changed since last build.
func shouldRebuild(artifactName string, store forge.ArtifactStore, forceRebuild bool) (bool, string, error) {
	// Step 1: If forceRebuild is true, always rebuild
	if forceRebuild {
		return true, "force flag set", nil
	}

	// Step 2: Look up latest artifact for artifactName in store
	artifact, err := forge.GetLatestArtifact(store, artifactName)
	if err != nil {
		// Step 3: If no artifact found, rebuild
		return true, "no previous build", nil
	}

	// Step 4: Check if artifact location still exists on filesystem
	if _, err := os.Stat(artifact.Location); os.IsNotExist(err) {
		return true, "artifact file missing", nil
	} else if err != nil {
		// If stat fails for other reason, assume rebuild needed
		return true, fmt.Sprintf("cannot access artifact file: %v", err), nil
	}

	// Step 5: If artifact has no Dependencies field (nil or empty)
	if len(artifact.Dependencies) == 0 {
		return true, "dependencies not tracked", nil
	}

	// Step 7: If artifact has no DependencyDetectorEngine, rebuild
	if artifact.DependencyDetectorEngine == "" {
		return true, "dependency detector not configured", nil
	}

	// Step 6: Compare using STORED dependencies ONLY (DO NOT re-detect)
	goModTracked := false
	for _, dep := range artifact.Dependencies {
		if dep.Type == forge.DependencyTypeFile {
			// Check if go.mod is tracked
			if strings.HasSuffix(dep.FilePath, "go.mod") {
				goModTracked = true
			}

			// Check if file still exists
			fileInfo, err := os.Stat(dep.FilePath)
			if os.IsNotExist(err) {
				return true, fmt.Sprintf("dependency file %s missing", dep.FilePath), nil
			} else if err != nil {
				// If stat fails for other reason, assume changed (safe default)
				return true, fmt.Sprintf("cannot access dependency file %s: %v", dep.FilePath, err), nil
			}

			// Get current timestamp and format as RFC3339 UTC
			currentTimestamp := fileInfo.ModTime().UTC().Format(time.RFC3339)

			// Parse stored timestamp
			storedTime, err := time.Parse(time.RFC3339, dep.Timestamp)
			if err != nil {
				// Parse error - assume changed (safe default)
				return true, fmt.Sprintf("dependency %s timestamp parse error", dep.FilePath), nil
			}

			// Parse current timestamp
			currentTime, err := time.Parse(time.RFC3339, currentTimestamp)
			if err != nil {
				// Parse error - assume changed (safe default)
				return true, fmt.Sprintf("dependency %s current timestamp parse error", dep.FilePath), nil
			}

			// Compare timestamps using .Equal()
			if !currentTime.Equal(storedTime) {
				return true, fmt.Sprintf("dependency %s modified", dep.FilePath), nil
			}
		}
		// External package dependencies: DO NOT re-parse go.mod
		// External packages are considered unchanged (semver only changes if go.mod changes)
	}

	// If go.mod is NOT in file dependencies and we have external package dependencies
	hasExternalDeps := false
	for _, dep := range artifact.Dependencies {
		if dep.Type == forge.DependencyTypeExternalPackage {
			hasExternalDeps = true
			break
		}
	}
	if hasExternalDeps && !goModTracked {
		// Log warning once (don't fail build)
		fmt.Fprintf(os.Stderr, "Warning: go.mod not tracked as dependency for %s, external package changes may not trigger rebuild\n", artifactName)
	}

	// If all dependencies unchanged, no rebuild needed
	return false, "", nil
}

// buildWithSingleEngine handles building with a single engine (either direct go:// URI or single-engine alias).
func buildWithSingleEngine(
	command string,
	args []string,
	specs []map[string]any,
	dirs *ForgeDirs,
	engineConfig *forge.EngineConfig,
	forceRebuild bool,
) ([]forge.Artifact, error) {
	// Prepare specs with injected directories and config
	specsWithConfig := make([]map[string]any, len(specs))
	for i, spec := range specs {
		// Clone the spec
		clonedSpec := make(map[string]any)
		for k, v := range spec {
			clonedSpec[k] = v
		}

		// Inject directories
		clonedSpec["tmpDir"] = dirs.TmpDir
		clonedSpec["buildDir"] = dirs.BuildDir
		clonedSpec["rootDir"] = dirs.RootDir

		// Inject force rebuild flag
		clonedSpec["force"] = forceRebuild

		// Inject engine-specific config if provided (from alias)
		// For generic engines, promote spec fields to top level for backward compatibility
		if engineConfig != nil && engineConfig.Type == forge.BuilderEngineConfigType && len(engineConfig.Builder) > 0 {
			builderSpec := engineConfig.Builder[0].Spec
			if builderSpec.Command != "" {
				clonedSpec["command"] = builderSpec.Command
			}
			if len(builderSpec.Args) > 0 {
				clonedSpec["args"] = builderSpec.Args
			}
			if len(builderSpec.Env) > 0 {
				clonedSpec["env"] = builderSpec.Env
			}
			if builderSpec.EnvFile != "" {
				clonedSpec["envFile"] = builderSpec.EnvFile
			}
			if builderSpec.Context != "" {
				clonedSpec["context"] = builderSpec.Context
			}
		} else if nestedSpec, ok := spec["spec"].(map[string]interface{}); ok {
			// Keep the spec nested - engines expect it in input.Spec
			// Convert to map[string]any for consistency
			specMap := make(map[string]any)
			for k, v := range nestedSpec {
				specMap[k] = v
			}
			clonedSpec["spec"] = specMap
		}

		specsWithConfig[i] = clonedSpec
	}

	// Call MCP engine (use build for single spec, buildBatch for multiple)
	var mcpResult interface{}
	var err error
	if len(specsWithConfig) == 1 {
		mcpResult, err = callMCPEngine(command, args, "build", specsWithConfig[0])
	} else {
		params := map[string]any{
			"specs": specsWithConfig,
		}
		mcpResult, err = callMCPEngine(command, args, "buildBatch", params)
	}

	if err != nil {
		return nil, err
	}

	// Parse and return artifacts
	artifacts, err := parseArtifacts(mcpResult)
	if err != nil {
		return nil, err
	}

	return artifacts, nil
}

// parseArtifacts converts MCP result to forge.Artifact slice.
func parseArtifacts(result interface{}) ([]forge.Artifact, error) {
	// Result could be:
	// 1. A single artifact object
	// 2. An array of artifacts
	// 3. A BatchResult object (from buildBatch) containing an artifacts array

	// Try to convert to JSON and back to parse it
	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	// Try parsing as BatchResult first (from buildBatch operations)
	type BatchResult struct {
		Artifacts []forge.Artifact `json:"artifacts"`
	}
	var batchResult BatchResult
	if err := json.Unmarshal(data, &batchResult); err == nil && len(batchResult.Artifacts) > 0 {
		return batchResult.Artifacts, nil
	}

	// Try parsing as single artifact
	var singleArtifact forge.Artifact
	if err := json.Unmarshal(data, &singleArtifact); err == nil && singleArtifact.Name != "" {
		return []forge.Artifact{singleArtifact}, nil
	}

	// Try parsing as array of artifacts
	var multipleArtifacts []forge.Artifact
	if err := json.Unmarshal(data, &multipleArtifacts); err == nil {
		return multipleArtifacts, nil
	}

	return nil, fmt.Errorf("could not parse artifacts from result")
}
