package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/alexandremahdhaoui/forge/internal/cli"
	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// Version information (set via ldflags during build)
var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
)

func main() {
	cli.Bootstrap(cli.Config{
		Name:           "go-gen-mocks",
		Version:        Version,
		CommitSHA:      CommitSHA,
		BuildTimestamp: BuildTimestamp,
		RunMCP:         runMCPServer,
	})
}

func runMCPServer() error {
	server := mcpserver.New("go-gen-mocks", Version)

	config := engineframework.BuilderConfig{
		Name:      "go-gen-mocks",
		Version:   Version,
		BuildFunc: build,
	}

	if err := engineframework.RegisterBuilderTools(server, config); err != nil {
		return err
	}

	return server.RunDefault()
}

// build implements the BuilderFunc for generating Go mocks using mockery
func build(ctx context.Context, input mcptypes.BuildInput) (*forge.Artifact, error) {
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
	// Find the detector binary
	detectorPath, err := engineframework.FindDetector("go-gen-mocks-dep-detector")
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

	return engineframework.CallDetector(ctx, detectorPath, "detectDependencies", input)
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
