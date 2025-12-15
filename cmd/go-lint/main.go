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
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/alexandremahdhaoui/forge/internal/cli"
	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

const Name = "go-lint"

// Version information (set via ldflags during build)
var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
)

// docsConfig is the configuration for the docs subcommand.
var docsConfig = &enginedocs.Config{
	EngineName:   Name,
	LocalDir:     "cmd/go-lint/docs",
	BaseURL:      "https://raw.githubusercontent.com/alexandremahdhaoui/forge/refs/heads/main",
	RequiredDocs: []string{"usage", "schema"},
}

func main() {
	cli.Bootstrap(cli.Config{
		Name:           Name,
		Version:        Version,
		CommitSHA:      CommitSHA,
		BuildTimestamp: BuildTimestamp,
		RunMCP:         runMCPServer,
		DocsConfig:     docsConfig,
	})
}

func runMCPServer() error {
	server, err := SetupMCPServer(Name, Version, runTestsWithSpec)
	if err != nil {
		return err
	}

	if err := enginedocs.RegisterDocsTools(server, *docsConfig); err != nil {
		return err
	}

	// Run the MCP server
	return server.RunDefault()
}

// runTestsWithSpec implements the TestRunnerFunc for running Go linter.
// It implements the TestRunnerFunc signature defined in zz_generated.mcp.go.
func runTestsWithSpec(ctx context.Context, input mcptypes.RunInput, spec *Spec) (*forge.TestReport, error) {
	log.Printf("Running linter: stage=%s, name=%s", input.Stage, input.Name)

	startTime := time.Now()

	golangciVersion := os.Getenv("GOLANGCI_LINT_VERSION")
	if golangciVersion == "" {
		golangciVersion = "v2.6.0"
	}

	golangciPkg := fmt.Sprintf("github.com/golangci/golangci-lint/v2/cmd/golangci-lint@%s", golangciVersion)

	args := []string{"run", golangciPkg, "run", "--fix"}

	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	// Execute the command
	err := cmd.Run()
	duration := time.Since(startTime)

	// CRITICAL: Return report even if linting failed (Status="failed")
	status := "passed"
	errorMessage := ""
	total := 0
	passed := 1
	failed := 0

	if err != nil {
		status = "failed"
		failed = 1
		passed = 0
		if exitErr, ok := err.(*exec.ExitError); ok {
			total = 1 // At least one issue found
			errorMessage = fmt.Sprintf("linting failed with exit code %d", exitErr.ExitCode())
		} else {
			errorMessage = fmt.Sprintf("failed to execute linter: %v", err)
		}
	}

	return &forge.TestReport{
		Stage:        input.Stage,
		Status:       status,
		ErrorMessage: errorMessage,
		StartTime:    startTime,
		Duration:     duration.Seconds(),
		TestStats: forge.TestStats{
			Total:  total,
			Passed: passed,
			Failed: failed,
		},
		Coverage: forge.Coverage{
			Percentage: 0.0, // Linting doesn't provide coverage
		},
	}, nil
}
