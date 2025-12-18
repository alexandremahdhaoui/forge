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
	"sync"
	"time"

	"github.com/alexandremahdhaoui/forge/internal/mcpcaller"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// ParallelBuilderSpec defines the input specification for parallel builds.
// This is kept for internal use as the generated Spec doesn't fully support nested object arrays.
type ParallelBuilderSpec struct {
	// Builders is the list of sub-builder configurations to run in parallel.
	Builders []BuilderConfig `json:"builders"`
}

// BuilderConfig defines a single sub-builder configuration.
type BuilderConfig struct {
	// Name is the optional name for this builder (used in logs/errors).
	Name string `json:"name,omitempty"`
	// Engine is the engine URI (e.g., "go://build-go", "go://generic-builder").
	Engine string `json:"engine"`
	// Spec contains the build specification passed to the sub-builder.
	Spec map[string]any `json:"spec"`
}

// Build implements the BuildFunc for executing multiple builders in parallel.
// It uses the typed Spec provided by the generated MCP server setup.
// Note: The generated Spec doesn't fully parse the builders array, so we parse
// the original input.Spec to get the complete ParallelBuilderSpec.
func Build(ctx context.Context, input mcptypes.BuildInput, _ *Spec) (*forge.Artifact, error) {
	// Parse spec - using manual ParallelBuilderSpec since generated Spec doesn't
	// support nested object arrays
	var pbSpec ParallelBuilderSpec
	if err := mapToStruct(input.Spec, &pbSpec); err != nil {
		return nil, fmt.Errorf("failed to parse spec: %w", err)
	}

	if len(pbSpec.Builders) == 0 {
		return nil, fmt.Errorf("no builders specified in spec.builders")
	}

	log.Printf("parallel-builder: starting %d parallel builds", len(pbSpec.Builders))

	// Create MCP caller
	caller := mcpcaller.NewCaller(Version)

	// Results channel
	type result struct {
		artifact *forge.Artifact
		err      error
		name     string
	}
	results := make(chan result, len(pbSpec.Builders))

	// WaitGroup for goroutines
	var wg sync.WaitGroup

	// Launch parallel builders
	for _, builder := range pbSpec.Builders {
		wg.Add(1)
		go func(b BuilderConfig) {
			defer wg.Done()

			name := b.Name
			if name == "" {
				name = b.Engine
			}

			log.Printf("parallel-builder: starting build for %s", name)

			// Resolve engine
			command, args, err := caller.ResolveEngine(b.Engine)
			if err != nil {
				results <- result{err: fmt.Errorf("[%s] engine resolution failed: %w", name, err), name: name}
				return
			}

			// Call build tool
			resp, err := caller.CallMCP(command, args, "build", b.Spec)
			if err != nil {
				results <- result{err: fmt.Errorf("[%s] build failed: %w", name, err), name: name}
				return
			}

			// Parse artifact from response
			artifact, err := parseArtifact(resp)
			if err != nil {
				results <- result{err: fmt.Errorf("[%s] artifact parsing failed: %w", name, err), name: name}
				return
			}

			log.Printf("parallel-builder: completed build for %s", name)
			results <- result{artifact: artifact, name: name}
		}(builder)
	}

	// Wait and close channel
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var artifacts []*forge.Artifact
	var errors []error
	for r := range results {
		if r.err != nil {
			log.Printf("parallel-builder: error from %s: %v", r.name, r.err)
			errors = append(errors, r.err)
		} else if r.artifact != nil {
			artifacts = append(artifacts, r.artifact)
		}
	}

	// Create combined artifact
	combinedArtifact := combineArtifacts(input.Name, artifacts)

	// Return combined result
	if len(errors) > 0 {
		// Return combined artifact even with partial failures
		errMsg := fmt.Sprintf("parallel-builder: %d/%d builders failed: ", len(errors), len(pbSpec.Builders))
		for i, err := range errors {
			if i > 0 {
				errMsg += "; "
			}
			errMsg += err.Error()
		}
		return combinedArtifact, fmt.Errorf("%s", errMsg)
	}

	log.Printf("parallel-builder: all %d builds completed successfully", len(pbSpec.Builders))
	return combinedArtifact, nil
}

// combineArtifacts creates a meta-artifact representing all sub-artifacts.
func combineArtifacts(name string, artifacts []*forge.Artifact) *forge.Artifact {
	if len(artifacts) == 0 {
		return &forge.Artifact{
			Name:      name,
			Type:      "parallel-build",
			Location:  ".",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   "no-artifacts",
		}
	}

	// Use the first artifact's location as the primary location
	location := artifacts[0].Location
	if len(artifacts) > 1 {
		location = "multiple"
	}

	return &forge.Artifact{
		Name:      name,
		Type:      "parallel-build",
		Location:  location,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   fmt.Sprintf("%d-artifacts", len(artifacts)),
	}
}

// parseArtifact parses an artifact from MCP response.
func parseArtifact(resp interface{}) (*forge.Artifact, error) {
	if resp == nil {
		// No structured response is valid - the build succeeded but returned no artifact
		return &forge.Artifact{
			Name:      "unknown",
			Type:      "unknown",
			Location:  ".",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   "unknown",
		}, nil
	}

	// Try to parse as artifact directly
	respMap, ok := resp.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", resp)
	}

	// Extract artifact fields
	artifact := &forge.Artifact{}

	if name, ok := respMap["name"].(string); ok {
		artifact.Name = name
	}
	if typ, ok := respMap["type"].(string); ok {
		artifact.Type = typ
	}
	if location, ok := respMap["location"].(string); ok {
		artifact.Location = location
	}
	if timestamp, ok := respMap["timestamp"].(string); ok {
		artifact.Timestamp = timestamp
	}
	if version, ok := respMap["version"].(string); ok {
		artifact.Version = version
	}

	// Set defaults for missing fields
	if artifact.Name == "" {
		artifact.Name = "unknown"
	}
	if artifact.Type == "" {
		artifact.Type = "unknown"
	}
	if artifact.Location == "" {
		artifact.Location = "."
	}
	if artifact.Timestamp == "" {
		artifact.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	if artifact.Version == "" {
		artifact.Version = "unknown"
	}

	return artifact, nil
}

// mapToStruct converts a map to a struct using JSON marshal/unmarshal.
func mapToStruct(m map[string]any, v interface{}) error {
	if m == nil {
		return nil
	}

	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal map: %w", err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to unmarshal to struct: %w", err)
	}

	return nil
}
