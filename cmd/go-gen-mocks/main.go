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

	// Return artifact using CreateArtifact (generated code has no version)
	return engineframework.CreateArtifact(
		"mocks",
		"generated",
		getMocksDir(mocksDir),
	), nil
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
