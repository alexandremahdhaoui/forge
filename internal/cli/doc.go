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

// Package cli provides common CLI bootstrapping functionality for forge commands.
//
// This package eliminates duplicated main() function logic across 18+ command binaries
// by providing a unified bootstrap mechanism that handles:
//   - Version information initialization from ldflags
//   - Version flag handling (--version, -v, version)
//   - MCP server mode handling (--mcp flag)
//   - Standardized error handling and exit codes
//
// Example usage:
//
//	package main
//
//	import (
//	    "github.com/alexandremahdhaoui/forge/internal/cli"
//	)
//
//	// Version information (set via ldflags)
//	var (
//	    Version        = "dev"
//	    CommitSHA      = "unknown"
//	    BuildTimestamp = "unknown"
//	)
//
//	func main() {
//	    cli.Bootstrap(cli.Config{
//	        Name:           "my-command",
//	        Version:        Version,
//	        CommitSHA:      CommitSHA,
//	        BuildTimestamp: BuildTimestamp,
//	        RunCLI:         runCLI,
//	        RunMCP:         runMCP,
//	    })
//	}
//
//	func runCLI() error {
//	    // Command-specific CLI logic
//	    return nil
//	}
//
//	func runMCP() error {
//	    // Command-specific MCP server logic
//	    return nil
//	}
package cli
