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

package orchestrate

import (
	"encoding/json"
	"fmt"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

// BuilderOrchestrator orchestrates multiple builder engines in sequence.
// It executes each builder with the provided build specs and aggregates
// all resulting artifacts into a single slice.
type BuilderOrchestrator struct {
	callMCP    MCPCaller
	resolveURI EngineResolver
}

// NewBuilderOrchestrator creates a new builder orchestrator.
func NewBuilderOrchestrator(callMCP MCPCaller, resolveURI EngineResolver) *BuilderOrchestrator {
	return &BuilderOrchestrator{
		callMCP:    callMCP,
		resolveURI: resolveURI,
	}
}

// Orchestrate executes multiple builder engines sequentially and aggregates artifacts.
// All builders receive the same build specs (with directories and builder-specific config injected).
// Execution is sequential - if any builder fails, the entire orchestration fails (fail-fast).
// All artifacts from all builders are collected and returned as a single slice.
func (o *BuilderOrchestrator) Orchestrate(
	builderSpecs []forge.BuilderEngineSpec,
	buildSpecs []map[string]any,
	dirs map[string]any,
) ([]forge.Artifact, error) {
	if len(builderSpecs) == 0 {
		return nil, fmt.Errorf("no builder engines provided")
	}

	if len(buildSpecs) == 0 {
		return nil, fmt.Errorf("no build specs provided")
	}

	var allArtifacts []forge.Artifact

	// Execute each builder in sequence
	for i, builderSpec := range builderSpecs {
		// Resolve engine URI to command and args
		command, args, err := o.resolveURI(builderSpec.Engine)
		if err != nil {
			return nil, fmt.Errorf("builder[%d] %s: failed to resolve engine: %w",
				i, builderSpec.Engine, err)
		}

		// Prepare specs for this builder (clone and inject config)
		specsForBuilder := make([]map[string]any, len(buildSpecs))
		for j, spec := range buildSpecs {
			// Clone the spec to avoid mutating the original
			clonedSpec := make(map[string]any)
			for k, v := range spec {
				clonedSpec[k] = v
			}

			// Inject directories
			for k, v := range dirs {
				clonedSpec[k] = v
			}

			// Inject builder-specific config from EngineSpec into nested "spec" map
			// This is required because engines look for config in input.Spec, not top-level fields
			specMap := make(map[string]any)
			if builderSpec.Spec.Command != "" {
				specMap["command"] = builderSpec.Spec.Command
			}
			if len(builderSpec.Spec.Args) > 0 {
				specMap["args"] = builderSpec.Spec.Args
			}
			if len(builderSpec.Spec.Env) > 0 {
				specMap["env"] = builderSpec.Spec.Env
			}
			if builderSpec.Spec.EnvFile != "" {
				specMap["envFile"] = builderSpec.Spec.EnvFile
			}
			if builderSpec.Spec.WorkDir != "" {
				specMap["workDir"] = builderSpec.Spec.WorkDir
			}
			if len(specMap) > 0 {
				clonedSpec["spec"] = specMap
			}

			specsForBuilder[j] = clonedSpec
		}

		// Call builder engine (use build or buildBatch based on spec count)
		var result interface{}
		if len(specsForBuilder) == 1 {
			result, err = o.callMCP(command, args, "build", specsForBuilder[0])
		} else {
			params := map[string]any{
				"specs": specsForBuilder,
			}
			result, err = o.callMCP(command, args, "buildBatch", params)
		}

		if err != nil {
			return nil, fmt.Errorf("builder[%d] %s: build failed: %w",
				i, builderSpec.Engine, err)
		}

		// Parse artifacts from result
		artifacts, err := parseArtifacts(result)
		if err != nil {
			return nil, fmt.Errorf("builder[%d] %s: failed to parse artifacts: %w",
				i, builderSpec.Engine, err)
		}

		// Accumulate artifacts
		allArtifacts = append(allArtifacts, artifacts...)
	}

	return allArtifacts, nil
}

// parseArtifacts converts MCP result to forge.Artifact slice.
// Copied and adapted from cmd/forge/build.go:159-181.
func parseArtifacts(result interface{}) ([]forge.Artifact, error) {
	// Result could be:
	// 1. A single artifact object
	// 2. An array of artifacts
	// 3. A BatchResult object (from buildBatch) containing an artifacts array

	// Try to convert to JSON and back to parse it
	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
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
	if err := json.Unmarshal(data, &multipleArtifacts); err == nil && len(multipleArtifacts) > 0 {
		return multipleArtifacts, nil
	}

	return nil, fmt.Errorf("could not parse artifacts from result")
}
