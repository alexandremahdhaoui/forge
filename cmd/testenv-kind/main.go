package main

import (
	"github.com/alexandremahdhaoui/forge/internal/cli"
)

// Version information (set via ldflags during build)
var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
)

func main() {
	cli.Bootstrap(cli.Config{
		Name:           "testenv-kind",
		Version:        Version,
		CommitSHA:      CommitSHA,
		BuildTimestamp: BuildTimestamp,
		RunMCP:         runMCPServer,
	})
}
