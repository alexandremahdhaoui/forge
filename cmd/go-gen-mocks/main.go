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

	"github.com/alexandremahdhaoui/forge/internal/cli"
	"github.com/alexandremahdhaoui/forge/internal/version"
	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

const Name = "go-gen-mocks"

// Version information (set via ldflags during build)
var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
)

// docsConfig is the configuration for the docs subcommand.
var docsConfig = &enginedocs.Config{
	EngineName:   Name,
	LocalDir:     "cmd/go-gen-mocks/docs",
	BaseURL:      "https://raw.githubusercontent.com/alexandremahdhaoui/forge/refs/heads/main",
	RequiredDocs: []string{"usage", "schema"},
}

func main() {
	cli.Bootstrap(cli.Config{
		Name:           Name,
		Version:        Version,
		CommitSHA:      CommitSHA,
		BuildTimestamp: BuildTimestamp,
		RunMCP:         runMCPServer,
		DocsConfig:     docsConfig,
	})
}

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

// build implements the BuilderFunc for generating Go mocks using mockery
func build(ctx context.Context, input mcptypes.BuildInput, spec *Spec) (*forge.Artifact, error) {
	log.Printf("Generating mocks")

	// Get mocksDir from environment variable
	mocksDir := os.Getenv("MOCKS_DIR")

	if err := generateMocks(mocksDir); err != nil {
		return nil, fmt.Errorf("mock generation failed: %w", err)
	}

	// Detect dependencies for lazy rebuild
	deps, err := detectMockDependencies(ctx, input.RootDir)
	if err != nil {
		// Log warning but don't fail - lazy build is optional optimization
		log.Printf("WARNING: dependency detection failed: %v", err)
		// Return artifact without dependencies (will always rebuild)
		return engineframework.CreateArtifact(
			input.Name,
			"generated",
			getMocksDir(mocksDir),
		), nil
	}

	// Return artifact WITH dependencies for lazy rebuild
	artifact := engineframework.CreateArtifact(
		input.Name,
		"generated",
		getMocksDir(mocksDir),
	)
	artifact.Dependencies = deps
	artifact.DependencyDetectorEngine = "go://go-gen-mocks-dep-detector"
	return artifact, nil
}

// detectMockDependencies calls the go-gen-mocks-dep-detector MCP server
// to discover which files the mock generation depends on.
func detectMockDependencies(ctx context.Context, rootDir string) ([]forge.ArtifactDependency, error) {
	// Resolve detector URI to command and args
	// Use GetEffectiveVersion to handle both ldflags version and go run @version
	cmd, args, err := engineframework.ResolveDetector("go://go-gen-mocks-dep-detector", version.GetEffectiveVersion(Version))
	if err != nil {
		return nil, err
	}

	// Handle empty RootDir (use current working directory)
	workDir := rootDir
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	input := map[string]any{
		"workDir": workDir,
	}

	return engineframework.CallDetector(ctx, cmd, args, "detectDependencies", input)
}

func getMocksDir(mocksDir string) string {
	if mocksDir != "" {
		return mocksDir
	}
	if envDir := os.Getenv("MOCKS_DIR"); envDir != "" {
		return envDir
	}
	return "./internal/util/mocks"
}

func generateMocks(mocksDir string) error {
	mockeryVersion := os.Getenv("MOCKERY_VERSION")
	if mockeryVersion == "" {
		mockeryVersion = "v3.5.5"
	}

	mockery := fmt.Sprintf("github.com/vektra/mockery/v3@%s", mockeryVersion)

	// Clean mocks directory
	dir := getMocksDir(mocksDir)
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean mocks directory: %w", err)
	}

	// Generate mocks
	cmd := exec.Command("go", "run", mockery)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mockery failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✅ Generated mocks in %s\n", dir)
	return nil
}
