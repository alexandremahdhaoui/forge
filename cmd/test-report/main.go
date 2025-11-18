package main

import (
	"fmt"
	"os"

	"github.com/alexandremahdhaoui/forge/internal/cli"
)

// Version information (set via ldflags during build)
var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
)

func main() {
	// Check if running in direct CLI mode (test-report <command>)
	if len(os.Args) >= 2 && os.Args[1] != "--mcp" && os.Args[1] != "version" && os.Args[1] != "--version" && os.Args[1] != "-v" && os.Args[1] != "help" && os.Args[1] != "--help" && os.Args[1] != "-h" {
		command := os.Args[1]

		switch command {
		case "get":
			if len(os.Args) < 3 {
				fmt.Fprintf(os.Stderr, "Error: test report ID required\n")
				os.Exit(1)
			}
			if err := cmdGet(os.Args[2]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		case "list":
			stageFilter := ""
			// Parse --stage flag if present
			for i, arg := range os.Args {
				if arg == "--stage" && i+1 < len(os.Args) {
					stageFilter = os.Args[i+1]
					break
				}
			}
			if err := cmdList(stageFilter); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		case "delete":
			if len(os.Args) < 3 {
				fmt.Fprintf(os.Stderr, "Error: test report ID required\n")
				os.Exit(1)
			}
			if err := cmdDelete(os.Args[2]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
			printUsage()
			os.Exit(1)
		}
		return
	}

	// Otherwise, use standard cli.Bootstrap for MCP mode and version handling
	cli.Bootstrap(cli.Config{
		Name:           "test-report",
		Version:        Version,
		CommitSHA:      CommitSHA,
		BuildTimestamp: BuildTimestamp,
		RunMCP:         runMCPServer,
	})
}

func printUsage() {
	fmt.Print(`test-report - Manage test reports and artifacts

Usage:
  test-report get <REPORT-ID>          Get test report details
  test-report list [--stage=<NAME>]    List test reports
  test-report delete <REPORT-ID>       Delete a test report and its artifacts
  test-report --mcp                    Run as MCP server
  test-report version                  Show version information

Description:
  test-report manages test reports stored in the artifact store. It allows
  you to query test results, coverage data, and clean up test artifacts
  including JUnit XML files and coverage reports.

Examples:
  # List all test reports
  test-report list

  # List unit test reports only
  test-report list --stage=unit

  # Get details about a specific test report
  test-report get test-unit-unit-20251105-012345

  # Delete a test report and its artifacts
  test-report delete test-unit-unit-20251105-012345
`)
}
