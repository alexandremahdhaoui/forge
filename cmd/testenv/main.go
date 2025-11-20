package main

import (
	"fmt"
	"os"

	"github.com/alexandremahdhaoui/forge/internal/cli"
	"github.com/alexandremahdhaoui/forge/internal/version"
)

// Version information (set via ldflags during build)
var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
)

// versionInfo holds testenv's version information
var versionInfo *version.Info

func init() {
	versionInfo = version.New("testenv")
	versionInfo.Version = Version
	versionInfo.CommitSHA = CommitSHA
	versionInfo.BuildTimestamp = BuildTimestamp
}

// getVersion returns the actual testenv version, using build info if available
func getVersion() string {
	v, _, _ := versionInfo.Get()
	return v
}

func main() {
	// Check if running in direct CLI mode (testenv <command>)
	if len(os.Args) >= 2 && os.Args[1] != "--mcp" && os.Args[1] != "version" && os.Args[1] != "--version" && os.Args[1] != "-v" && os.Args[1] != "help" && os.Args[1] != "--help" && os.Args[1] != "-h" {
		command := os.Args[1]

		switch command {
		case "create":
			stageName := ""
			if len(os.Args) >= 3 {
				stageName = os.Args[2]
			}
			if _, err := cmdCreate(stageName); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		case "delete":
			if len(os.Args) < 3 {
				fmt.Fprintf(os.Stderr, "Error: test ID required\n\n")
				printUsage()
				os.Exit(1)
			}
			testID := os.Args[2]
			if err := cmdDelete(testID); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
			printUsage()
			os.Exit(1)
		}
		return
	}

	// Otherwise, use standard cli.Bootstrap for MCP mode and version handling
	cli.Bootstrap(cli.Config{
		Name:           "testenv",
		Version:        Version,
		CommitSHA:      CommitSHA,
		BuildTimestamp: BuildTimestamp,
		RunMCP:         runMCPServer,
	})
}

func printUsage() {
	fmt.Println(`testenv - Orchestrate test environments

Usage:
  testenv create <STAGE>        Create a test environment
  testenv delete <TEST-ID>      Delete a test environment
  testenv --mcp                 Run as MCP server
  testenv version               Show version information

Arguments:
  STAGE     Test stage name (e.g., "integration", "e2e")
  TEST-ID   Test environment ID

Examples:
  testenv create integration
  testenv delete test-integration-20241103-abc123
  testenv --mcp

Note:
  Use 'forge test <stage> get/list' to view test environments.
  testenv only handles create/delete operations.`)
}
