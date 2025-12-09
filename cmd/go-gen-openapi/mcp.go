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
	"os"
	"path/filepath"

	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// runMCPServer starts the go-gen-openapi MCP server with stdio transport.
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

	// Extract spec paths from config for dependency detection
	var specPaths []string
	for _, spec := range config.Specs {
		sourcePath := spec.Source
		if input.RootDir != "" && !filepath.IsAbs(sourcePath) {
			sourcePath = filepath.Join(input.RootDir, sourcePath)
		}
		specPaths = append(specPaths, sourcePath)
	}

	// Detect dependencies for lazy rebuild
	deps, err := detectOpenAPIDependencies(ctx, specPaths, input.RootDir)
	if err != nil {
		// Log warning but don't fail - lazy build is optional optimization
		log.Printf("WARNING: dependency detection failed: %v", err)
		// Return artifact without dependencies (will always rebuild)
		return engineframework.CreateArtifact(
			input.Name,
			"generated",
			config.Specs[0].DestinationDir,
		), nil
	}

	// Return artifact WITH dependencies for lazy rebuild
	artifact := engineframework.CreateArtifact(
		input.Name,
		"generated",
		config.Specs[0].DestinationDir,
	)
	artifact.Dependencies = deps
	artifact.DependencyDetectorEngine = "go://go-gen-openapi-dep-detector"
	return artifact, nil
}

// detectOpenAPIDependencies calls the go-gen-openapi-dep-detector MCP server
// to discover which files the OpenAPI generation depends on.
func detectOpenAPIDependencies(ctx context.Context, specPaths []string, rootDir string) ([]forge.ArtifactDependency, error) {
	// Resolve detector URI to command and args
	cmd, args, err := engineframework.ResolveDetector("go://go-gen-openapi-dep-detector", Version)
	if err != nil {
		return nil, err
	}

	input := map[string]any{
		"specSources": specPaths,
		"rootDir":     rootDir,
		"resolveRefs": false, // v1: no $ref resolution
	}

	return engineframework.CallDetector(ctx, cmd, args, "detectDependencies", input)
}
