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

package enginecli

import (
	"fmt"
	"log"
	"os"

	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"github.com/alexandremahdhaoui/forge/pkg/engineversion"
)

// Config holds the configuration for CLI bootstrap.
type Config struct {
	// Name is the command name (e.g., "go-build", "test-integration")
	Name string

	// Version information (typically set via ldflags)
	Version        string
	CommitSHA      string
	BuildTimestamp string

	// RunCLI is the function to execute in normal CLI mode
	RunCLI func() error

	// RunMCP is the function to execute in MCP server mode (optional)
	// If nil, --mcp flag will result in an error
	RunMCP func() error

	// SuccessHandler is called when RunCLI completes successfully (optional)
	// Defaults to no-op if not provided
	SuccessHandler func()

	// FailureHandler is called when RunCLI returns an error (optional)
	// Receives the error and should print it appropriately
	// Defaults to no-op if not provided
	FailureHandler func(error)

	// DocsConfig is the configuration for the docs subcommand (optional)
	// If set and "docs" is the first argument, the docs command is handled internally
	DocsConfig *enginedocs.Config
}

// Bootstrap provides a unified entry point for forge CLI commands.
// It handles version flags, MCP mode, and CLI execution with standardized error handling.
//
// This function will call os.Exit and never return.
func Bootstrap(cfg Config) {
	// Initialize version information
	versionInfo := engineversion.New(cfg.Name)
	versionInfo.Version = cfg.Version
	versionInfo.CommitSHA = cfg.CommitSHA
	versionInfo.BuildTimestamp = cfg.BuildTimestamp

	// Check for version flag
	for _, arg := range os.Args[1:] {
		if arg == "version" || arg == "--version" || arg == "-v" {
			versionInfo.Print()
			os.Exit(0)
		}
	}

	// Check for docs subcommand
	if cfg.DocsConfig != nil && len(os.Args) > 1 && os.Args[1] == "docs" {
		exitCode := handleDocsCommand(cfg.DocsConfig, os.Args[2:])
		os.Exit(exitCode)
	}

	// Check for --mcp flag to run as MCP server
	for _, arg := range os.Args[1:] {
		if arg == "--mcp" {
			if cfg.RunMCP == nil {
				log.Printf("Error: MCP mode not supported for %s", cfg.Name)
				os.Exit(1)
			}
			if err := cfg.RunMCP(); err != nil {
				log.Printf("MCP server error: %v", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}

	// Normal CLI mode
	if err := cfg.RunCLI(); err != nil {
		if cfg.FailureHandler != nil {
			cfg.FailureHandler(err)
		}
		os.Exit(1)
	}

	if cfg.SuccessHandler != nil {
		cfg.SuccessHandler()
	}
	os.Exit(0)
}

// BootstrapSimple is a convenience wrapper for commands that don't support MCP mode.
func BootstrapSimple(name, version, commitSHA, buildTimestamp string, runCLI func() error) {
	Bootstrap(Config{
		Name:           name,
		Version:        version,
		CommitSHA:      commitSHA,
		BuildTimestamp: buildTimestamp,
		RunCLI:         runCLI,
		RunMCP:         nil,
	})
}

// handleDocsCommand processes the docs subcommand and returns the exit code.
// It supports list, get <name>, and validate subcommands.
func handleDocsCommand(cfg *enginedocs.Config, args []string) int {
	switch {
	case len(args) == 0 || args[0] == "list":
		docs, err := enginedocs.DocsList(*cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing docs: %v\n", err)
			return 1
		}
		for _, doc := range docs {
			fmt.Printf("%s\t%s\t%s\n", doc.Name, doc.Title, doc.Description)
		}
		return 0

	case args[0] == "get" && len(args) > 1:
		content, err := enginedocs.DocsGet(*cfg, args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting doc: %v\n", err)
			return 1
		}
		fmt.Print(content)
		return 0

	case args[0] == "validate":
		errs := enginedocs.Validate(*cfg)
		if len(errs) == 0 {
			fmt.Println("Validation passed: all checks passed")
			return 0
		}
		fmt.Fprintf(os.Stderr, "Validation failed with %d error(s):\n", len(errs))
		for _, err := range errs {
			fmt.Fprintf(os.Stderr, "  - %v\n", err)
		}
		return 1

	default:
		fmt.Fprintln(os.Stderr, "Usage: <command> docs [list|get <name>|validate]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Subcommands:")
		fmt.Fprintln(os.Stderr, "  list              List all available documentation")
		fmt.Fprintln(os.Stderr, "  get <name>        Get the content of a specific document")
		fmt.Fprintln(os.Stderr, "  validate          Validate documentation completeness")
		return 1
	}
}
