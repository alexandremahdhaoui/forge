package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// runMCPServer starts the go-gen-openapi MCP server with stdio transport.
func runMCPServer() error {
	server := mcpserver.New("go-gen-openapi", Version)

	config := engineframework.BuilderConfig{
		Name:      "go-gen-openapi",
		Version:   Version,
		BuildFunc: build,
	}

	if err := engineframework.RegisterBuilderTools(server, config); err != nil {
		return err
	}

	return server.RunDefault()
}

// build implements the BuilderFunc for generating OpenAPI client and server code
func build(ctx context.Context, input mcptypes.BuildInput) (*forge.Artifact, error) {
	log.Printf("Generating OpenAPI code for: %s", input.Name)

	// Extract OpenAPI config from BuildInput.Spec
	config, err := extractOpenAPIConfigFromInput(input)
	if err != nil {
		return nil, fmt.Errorf("failed to extract config: %w", err)
	}

	// Get oapi-codegen version and build executable command
	oapiCodegenVersion := os.Getenv("OAPI_CODEGEN_VERSION")
	if oapiCodegenVersion == "" {
		oapiCodegenVersion = "v2.3.0"
	}

	executable := fmt.Sprintf("go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@%s", oapiCodegenVersion)

	// Call existing generation logic, passing RootDir for relative path resolution
	if err := doGenerate(executable, *config, input.RootDir); err != nil {
		return nil, fmt.Errorf("generation failed: %w", err)
	}

	// Create artifact using CreateArtifact (generated code has no version)
	return engineframework.CreateArtifact(
		input.Name,
		"generated",
		config.Specs[0].DestinationDir,
	), nil
}
