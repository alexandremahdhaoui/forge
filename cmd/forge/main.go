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
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexandremahdhaoui/forge/internal/cmdutil"
	"github.com/alexandremahdhaoui/forge/pkg/engineversion"
)

// Version information (set via ldflags during build)
var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
)

// versionInfo holds forge's version information
var versionInfo *engineversion.Info

// configPath is the path to the forge.yaml file
var configPath string

// cwdOverride is set by --cwd flag to change working directory before command execution
var cwdOverride string

// skipWorkspaceResolution is set by --skip-workspace-resolution flag to disable
// automatic Go workspace detection
var skipWorkspaceResolution bool

func init() {
	versionInfo = engineversion.New("forge")
	versionInfo.Version = Version
	versionInfo.CommitSHA = CommitSHA
	versionInfo.BuildTimestamp = BuildTimestamp
}

// getVersion returns the actual forge version, using build info if available
func getVersion() string {
	v, _, _ := versionInfo.Get()
	return v
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Parse global flags
	args := os.Args[1:]
	args = parseGlobalFlags(args)

	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	// Apply --cwd override before workspace resolution and config-based directory change
	if cwdOverride != "" {
		absPath, err := filepath.Abs(cwdOverride)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot resolve --cwd path %q: %v\n", cwdOverride, err)
			os.Exit(1)
		}
		if err := os.Chdir(absPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot change to --cwd directory %q: %v\n", absPath, err)
			os.Exit(1)
		}
	}

	// Resolve Go workspace if present
	if err := resolveWorkspace(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: workspace resolution failed: %v\n", err)
		// Non-fatal: continue without workspace resolution
	}

	// Change to the directory containing forge.yaml if --config specifies a path with directory components.
	// This ensures all relative paths in forge.yaml resolve correctly.
	if err := changeToProjectDir(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Source global environment file if configured
	// TODO: This calls loadConfig() which is also called in command handlers.
	// This is intentional to avoid refactoring all handlers, but is technical debt.
	// The double-call is acceptable as YAML parsing is fast (<1ms).
	config, err := loadConfig()
	if err == nil && config.EnvFile != "" {
		if err := cmdutil.SourceEnvFile(config.EnvFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error sourcing envFile: %v\n", err)
			os.Exit(1)
		}
	}
	// Note: We don't fail if loadConfig errors here - let the command handler report it

	command := args[0]
	cmdArgs := args[1:]

	switch command {
	case "--mcp":
		// Run in MCP server mode
		if err := runMCPServer(); err != nil {
			log.Printf("MCP server error: %v", err)
			os.Exit(1)
		}
	case "build":
		// Parse force flag
		forceRebuild := false
		filteredArgs := make([]string, 0, len(cmdArgs))
		for _, arg := range cmdArgs {
			if arg == "-f" || arg == "--force" {
				forceRebuild = true
			} else {
				filteredArgs = append(filteredArgs, arg)
			}
		}
		if err := runBuild(filteredArgs, forceRebuild); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "test":
		if err := runTest(cmdArgs); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "test-all":
		// Parse force flag
		forceRebuild := false
		filteredArgs := make([]string, 0, len(cmdArgs))
		for _, arg := range cmdArgs {
			if arg == "-f" || arg == "--force" {
				forceRebuild = true
			} else {
				filteredArgs = append(filteredArgs, arg)
			}
		}
		if err := runTestAll(filteredArgs, forceRebuild); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "docs":
		if err := runDocs(cmdArgs); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "config":
		if err := runConfig(cmdArgs); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "cu":
		if err := runCU(cmdArgs); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "ws", "workspace":
		if err := runWS(cmdArgs); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "ui":
		if err := runUI(cmdArgs); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "list":
		if err := runList(cmdArgs); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "version", "--version", "-v":
		versionInfo.Print()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

// changeToProjectDir changes the working directory to the directory containing
// the config file when --config specifies a path with directory components.
// This ensures all relative paths in forge.yaml resolve correctly when forge
// is invoked from a parent directory (e.g. a Go workspace root).
func changeToProjectDir() error {
	if configPath == "" {
		return nil
	}
	dir := filepath.Dir(configPath)
	if dir == "." {
		return nil
	}
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("cannot change to project directory %q: %w", dir, err)
	}
	configPath = filepath.Base(configPath)
	fmt.Fprintf(os.Stderr, "forge: changed working directory to %s\n", dir)
	return nil
}

// parseGlobalFlags parses global flags like --config and returns remaining args
func parseGlobalFlags(args []string) []string {
	result := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--config" {
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "Error: --config requires a path argument\n")
				os.Exit(1)
			}
			configPath = args[i+1]
			i++ // Skip the next argument (the path)
		} else if val, ok := strings.CutPrefix(arg, "--config="); ok {
			configPath = val
			if configPath == "" {
				fmt.Fprintf(os.Stderr, "Error: --config requires a path argument\n")
				os.Exit(1)
			}
		} else if arg == "--cwd" {
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "Error: --cwd requires a path argument\n")
				os.Exit(1)
			}
			cwdOverride = args[i+1]
			i++ // Skip the next argument (the path)
		} else if val, ok := strings.CutPrefix(arg, "--cwd="); ok {
			cwdOverride = val
			if cwdOverride == "" {
				fmt.Fprintf(os.Stderr, "Error: --cwd requires a path argument\n")
				os.Exit(1)
			}
		} else if arg == "--skip-workspace-resolution" {
			skipWorkspaceResolution = true
		} else {
			result = append(result, arg)
		}
	}

	return result
}

func printUsage() {
	fmt.Println(`forge - A build orchestration tool

Usage:
  forge [global flags] <command> [args...]

Global Flags:
  --config <path>                    Use custom forge.yaml path (default: forge.yaml)
  --cwd <path>                       Change working directory before running command
  --skip-workspace-resolution        Disable automatic Go workspace detection

Commands:
  build [artifact-name]              Build all artifacts
  test <subcommand> <stage> [args...]  Test operations (run, list, manage environments)
  test-all                           Build all artifacts and run all test stages
  list [build|test]                  List available build targets and test stages
  docs <list|get> [name]             Fetch project documentation
  config <subcommand>                Configuration management
  cu <subcommand>                    Continuous-update operations (status, commit, checkout, go-get)
  ws <subcommand>                    Workspace lifecycle (list, create, delete, suspend, resume)
  ui                                 Launch the forge TUI dashboard
  version                            Show version information

Build:
  build [-f|--force]                 Build all artifacts from forge.yaml
  build [-f|--force] <artifact-name> Build specific artifact (force rebuild all)

Test:
  test run <stage> [env-id]          Run tests for stage (optionally reuse environment)
  test list <stage>                  List test reports for stage
  test get <stage> <test-id>         Get test report details
  test delete <stage> <test-id>      Delete test report
  test list-env <stage>              List test environments for stage
  test get-env <stage> <env-id>      Get test environment details
  test create-env <stage>            Create test environment for stage
  test delete-env <stage> <env-id>   Delete test environment

Test All:
  test-all [-f|--force]              Build all artifacts and run all test stages sequentially

List:
  list                               List all build targets and test stages
  list build                         List only build targets
  list test                          List only test stages

Docs:
  docs list                          List all available documentation
  docs get <name>                    Fetch a specific document

Config:
  config validate [path]             Validate forge.yaml configuration

Continuous Update:
  cu status                          Show pending dependency changes
  cu commit --message <msg>          Commit pending changes
  cu checkout --branch <name>        Check out a branch
  cu list-branches                   List branches
  cu go-get --package <pkg>          Run go get and commit changes

Workspace:
  ws list                            List workspaces
  ws create [flags]                  Create a workspace (--image, --namespace)
  ws get --name <name>               Get workspace details
  ws delete --name <name>            Delete a workspace
  ws suspend --name <name>           Suspend a workspace
  ws resume --name <name>            Resume a workspace

UI:
  ui                                 Launch the forge TUI dashboard

Other:
  version                            Show version information
  help                               Show this help message`)
}
