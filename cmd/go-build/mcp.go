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
	"os/exec"
	"path/filepath"

	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// runMCPServer starts the go-build MCP server with stdio transport.
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
		Description: "Validate go-build configuration",
	}, handleConfigValidate)

	return server.RunDefault()
}

// build implements the BuilderFunc for building Go binaries
func build(ctx context.Context, input mcptypes.BuildInput) (*forge.Artifact, error) {
	log.Printf("Building binary: %s from %s", input.Name, input.Src)

	// Extract build options from input
	opts := extractBuildOptionsFromInput(input)

	// Determine destination directory
	dest := input.Dest
	if dest == "" {
		dest = "./build/bin"
	}

	// Create destination directory
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	outputPath := filepath.Join(dest, input.Name)

	// Set CGO_ENABLED=0 for static binaries (can be overridden by custom env)
	if err := os.Setenv("CGO_ENABLED", "0"); err != nil {
		return nil, fmt.Errorf("failed to set CGO_ENABLED: %w", err)
	}

	// Apply custom environment variables if provided
	if opts != nil && len(opts.CustomEnv) > 0 {
		for key, value := range opts.CustomEnv {
			if err := os.Setenv(key, value); err != nil {
				return nil, fmt.Errorf("failed to set environment variable %s: %w", key, err)
			}
		}
	}

	// Build command arguments
	args := []string{
		"build",
		"-o", outputPath,
	}

	// Add ldflags from environment if provided
	if ldflags := os.Getenv("GO_BUILD_LDFLAGS"); ldflags != "" {
		args = append(args, "-ldflags", ldflags)
	}

	// Add custom args if provided
	if opts != nil && len(opts.CustomArgs) > 0 {
		args = append(args, opts.CustomArgs...)
	}

	// Add source path
	args = append(args, input.Src)

	// Execute build
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stderr // MCP mode: redirect to stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go build failed: %w", err)
	}

	// Create versioned artifact
	artifact, err := engineframework.CreateVersionedArtifact(
		input.Name,
		"binary",
		outputPath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create artifact: %w", err)
	}

	// Detect dependencies if this is a main package
	if err := detectDependenciesForArtifact(input.Src, artifact); err != nil {
		return nil, fmt.Errorf("failed to detect dependencies: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✅ Built binary: %s (version: %s)\n", input.Name, artifact.Version)

	return artifact, nil
}

// extractBuildOptionsFromInput extracts BuildOptions from BuildInput fields.
// It first checks the Spec field (from forge.yaml BuildSpec.Spec), then falls back to direct Args/Env fields.
// Direct Args/Env fields take precedence over Spec if both are present.
func extractBuildOptionsFromInput(input mcptypes.BuildInput) *BuildOptions {
	opts := &BuildOptions{}

	// First, try to extract from Spec field (from BuildSpec.Spec in forge.yaml)
	if len(input.Spec) > 0 {
		// Extract args from spec
		if argsVal, ok := input.Spec["args"]; ok {
			if args, ok := argsVal.([]interface{}); ok {
				opts.CustomArgs = make([]string, 0, len(args))
				for _, arg := range args {
					if argStr, ok := arg.(string); ok {
						opts.CustomArgs = append(opts.CustomArgs, argStr)
					}
				}
			}
		}

		// Extract env from spec
		if envVal, ok := input.Spec["env"]; ok {
			if env, ok := envVal.(map[string]interface{}); ok {
				opts.CustomEnv = make(map[string]string, len(env))
				for key, val := range env {
					if valStr, ok := val.(string); ok {
						opts.CustomEnv[key] = valStr
					}
				}
			}
		}
	}

	// Direct Args/Env fields take precedence over Spec
	if len(input.Args) > 0 {
		opts.CustomArgs = input.Args
	}

	if len(input.Env) > 0 {
		opts.CustomEnv = input.Env
	}

	// Return nil if no options were extracted
	if len(opts.CustomArgs) == 0 && len(opts.CustomEnv) == 0 {
		return nil
	}

	return opts
}
