package main

import (
	"fmt"

	"github.com/alexandremahdhaoui/forge/internal/cli"
)

// Name is the name of the tool
const Name = "go-gen-bpf"

// Version information (set via ldflags during build)
var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
)

func main() {
	cli.Bootstrap(cli.Config{
		Name:           Name,
		Version:        Version,
		CommitSHA:      CommitSHA,
		BuildTimestamp: BuildTimestamp,
		RunMCP:         runMCPServer,
		RunCLI:         runCLI,
	})
}

func runCLI() error {
	return fmt.Errorf("%s only supports MCP mode, use --mcp flag", Name)
}
