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
	"log"

	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/caarlos0/env/v11"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// runMCPServer starts the container-build MCP server with stdio transport.
func runMCPServer() error {
	server := mcpserver.New(Name, Version)

	config := engineframework.BuilderConfig{
		Name:      Name,
		Version:   Version,
		BuildFunc: build,
	}

	if err := engineframework.RegisterBuilderTools(server, config); err != nil {
		return err
	}

	if err := enginedocs.RegisterDocsTools(server, *docsConfig); err != nil {
		return err
	}

	// Register config-validate tool
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "config-validate",
		Description: "Validate container-build engine configuration",
	}, handleConfigValidate)

	return server.RunDefault()
}

// build implements the BuilderFunc for building container images
func build(ctx context.Context, input mcptypes.BuildInput) (*forge.Artifact, error) {
	log.Printf("Building container: %s from %s", input.Name, input.Src)

	// Parse environment variables (CONTAINER_BUILD_ENGINE is required)
	envs := Envs{} //nolint:exhaustruct
	if err := env.Parse(&envs); err != nil {
		return nil, fmt.Errorf("environment parse failed: %w (CONTAINER_BUILD_ENGINE required)", err)
	}

	// Validate container engine
	if err := validateContainerEngine(envs.BuildEngine); err != nil {
		return nil, err
	}

	// Create BuildSpec from input (include Spec for dependsOn support)
	spec := forge.BuildSpec{
		Name:   input.Name,
		Src:    input.Src,
		Dest:   input.Dest,
		Engine: input.Engine,
		Spec:   input.Spec,
	}

	// Get git version
	version, err := engineframework.GetGitVersion()
	if err != nil {
		return nil, fmt.Errorf("could not get git version: %w", err)
	}

	// Build the container (isMCPMode=true)
	var store forge.ArtifactStore
	if err := buildContainer(envs, spec, version, "", &store, true); err != nil {
		return nil, err
	}

	// Retrieve the artifact from store to get dependencies
	location := fmt.Sprintf("%s:%s", input.Name, version)
	artifact := engineframework.CreateCustomArtifact(
		input.Name,
		"container",
		location,
		version,
	)

	// Find artifact in store to get dependencies
	for _, a := range store.Artifacts {
		if a.Name == input.Name {
			artifact.Dependencies = a.Dependencies
			artifact.DependencyDetectorEngine = a.DependencyDetectorEngine
			artifact.DependencyDetectorSpec = a.DependencyDetectorSpec
			break
		}
	}

	return artifact, nil
}
