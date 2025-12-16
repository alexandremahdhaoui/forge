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
	"fmt"
	"os"

	"github.com/alexandremahdhaoui/forge/pkg/enginecli"
	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
)

// Name is the name of this engine.
const Name = "forge-dev"

// Version information (set via ldflags during build)
var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
)

// docsConfig is the configuration for the docs subcommand.
var docsConfig = &enginedocs.Config{
	EngineName:   Name,
	LocalDir:     "cmd/forge-dev/docs",
	BaseURL:      "https://raw.githubusercontent.com/alexandremahdhaoui/forge/refs/heads/main",
	RequiredDocs: []string{"usage", "schema"},
}

func main() {
	enginecli.Bootstrap(enginecli.Config{
		Name:           Name,
		Version:        Version,
		CommitSHA:      CommitSHA,
		BuildTimestamp: BuildTimestamp,
		RunCLI:         run,
		RunMCP:         runMCPServer,
		SuccessHandler: printSuccess,
		FailureHandler: printFailure,
		DocsConfig:     docsConfig,
	})
}

// run executes the main logic of the forge-dev tool in CLI mode.
// Currently, CLI mode is not supported as forge-dev is primarily an MCP server.
func run() error {
	return fmt.Errorf("CLI mode not supported for forge-dev. Use --mcp flag to run as MCP server")
}

func printSuccess() {
	_, _ = fmt.Fprintln(os.Stdout, "forge-dev: operation completed successfully")
}

func printFailure(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "forge-dev: error: %s\n", err.Error())
}
