package main

import (
	"github.com/alexandremahdhaoui/forge/internal/cli"
	"github.com/alexandremahdhaoui/forge/internal/cmdutil"
)

// Version information (set via ldflags during build)
var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
)

// Type aliases for convenience
type (
	ExecuteInput  = cmdutil.ExecuteInput
	ExecuteOutput = cmdutil.ExecuteOutput
)

func main() {
	cli.Bootstrap(cli.Config{
		Name:           "generic-builder",
		Version:        Version,
		CommitSHA:      CommitSHA,
		BuildTimestamp: BuildTimestamp,
		RunMCP:         runMCPServer,
	})
}
