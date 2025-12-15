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

	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// runMCPServer starts the go-build MCP server with stdio transport.
func runMCPServer() error {
	server, err := SetupMCPServer(Name, Version, build)
	if err != nil {
		return err
	}

	if err := enginedocs.RegisterDocsTools(server, *docsConfig); err != nil {
		return err
	}

	return server.RunDefault()
}

// build implements the BuildFunc for building Go binaries
func build(ctx context.Context, input mcptypes.BuildInput, spec *Spec) (*forge.Artifact, error) {
	log.Printf("Building binary: %s from %s", input.Name, input.Src)

	// Use spec values for custom args and env, falling back to input values
	customArgs := spec.Args
	if len(customArgs) == 0 {
		customArgs = input.Args
	}

	customEnv := spec.Env
	if len(customEnv) == 0 {
		customEnv = input.Env
	}

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
	for key, value := range customEnv {
		if err := os.Setenv(key, value); err != nil {
			return nil, fmt.Errorf("failed to set environment variable %s: %w", key, err)
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
	args = append(args, customArgs...)

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
