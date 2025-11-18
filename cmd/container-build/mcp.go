package main

import (
	"context"
	"fmt"
	"log"

	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/caarlos0/env/v11"
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

	// Create BuildSpec from input
	spec := forge.BuildSpec{
		Name:   input.Name,
		Src:    input.Src,
		Dest:   input.Dest,
		Engine: input.Engine,
	}

	// Get git version
	version, err := engineframework.GetGitVersion()
	if err != nil {
		return nil, fmt.Errorf("could not get git version: %w", err)
	}

	// Build the container (isMCPMode=true)
	var dummyStore forge.ArtifactStore
	if err := buildContainer(envs, spec, version, "", &dummyStore, true); err != nil {
		return nil, err
	}

	// Create versioned artifact using custom version (container uses image:version format)
	location := fmt.Sprintf("%s:%s", input.Name, version)
	artifact := engineframework.CreateCustomArtifact(
		input.Name,
		"container",
		location,
		version,
	)

	return artifact, nil
}
