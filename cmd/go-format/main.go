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
		Name:           "go-format",
		Version:        Version,
		CommitSHA:      CommitSHA,
		BuildTimestamp: BuildTimestamp,
		RunMCP:         runMCPServer,
	})
}

func runMCPServer() error {
	server := mcpserver.New("go-format", Version)

	config := engineframework.BuilderConfig{
		Name:      "go-format",
		Version:   Version,
		BuildFunc: build,
	}

	if err := engineframework.RegisterBuilderTools(server, config); err != nil {
		return err
	}

	return server.RunDefault()
}

// build implements the BuilderFunc for formatting Go code
func build(ctx context.Context, input mcptypes.BuildInput) (*forge.Artifact, error) {
	path := input.Path
	if path == "" && input.Src != "" {
		path = input.Src
	}
	if path == "" {
		path = "."
	}

	log.Printf("Formatting Go code at: %s", path)

	if err := formatCode(path); err != nil {
		return nil, fmt.Errorf("formatting failed: %w", err)
	}

	// Return artifact using CreateArtifact (formatted code has no version)
	return engineframework.CreateArtifact(
		"formatted-code",
		"formatted",
		path,
	), nil
}

func formatCode(path string) error {
	gofumptVersion := os.Getenv("GOFUMPT_VERSION")
	if gofumptVersion == "" {
		gofumptVersion = "v0.6.0"
	}

	gofumptPkg := fmt.Sprintf("mvdan.cc/gofumpt@%s", gofumptVersion)

	cmd := exec.Command("go", "run", gofumptPkg, "-w", path)
	cmd.Stdout = os.Stderr // Send to stderr to not interfere with MCP JSON-RPC on stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gofumpt failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✅ Formatted Go code at %s\n", path)
	return nil
}
