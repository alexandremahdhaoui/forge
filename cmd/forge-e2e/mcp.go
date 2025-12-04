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
	"time"

	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// runMCPServer starts the forge-e2e MCP server with stdio transport.
func runMCPServer() error {
	server := mcpserver.New("forge-e2e", Version)

	config := engineframework.TestRunnerConfig{
		Name:        "forge-e2e",
		Version:     Version,
		RunTestFunc: runTestsWrapper,
	}

	if err := engineframework.RegisterTestRunnerTools(server, config); err != nil {
		return err
	}

	return server.RunDefault()
}

// runTestsWrapper adapts the test execution to the framework's TestRunnerFunc signature.
func runTestsWrapper(ctx context.Context, input mcptypes.RunInput) (*forge.TestReport, error) {
	log.Printf("Running e2e tests: stage=%s, name=%s", input.Stage, input.Name)

	// Run the actual tests
	detailedReport := runTests(input.Stage, input.Name)

	// Convert DetailedTestReport to forge.TestReport
	duration := time.Duration(detailedReport.Duration * float64(time.Second))
	forgeReport := &forge.TestReport{
		Stage:        input.Stage,
		Status:       detailedReport.Status,
		ErrorMessage: detailedReport.ErrorMessage,
		StartTime:    time.Now().Add(-duration),
		Duration:     duration.Seconds(),
		TestStats: forge.TestStats{
			Total:   detailedReport.Total,
			Passed:  detailedReport.Passed,
			Failed:  detailedReport.Failed,
			Skipped: detailedReport.Skipped,
		},
		Coverage: forge.Coverage{
			Percentage: 0.0, // forge-e2e doesn't track coverage
		},
	}

	// CRITICAL: Return (report, nil) even if tests failed
	// Status field indicates pass/fail, error is only for execution failures
	return forgeReport, nil
}

// run is the CLI entry point for direct execution (non-MCP mode).
func run(stage, name string) error {
	report := runTests(stage, name)

	if report.Status == "failed" {
		return fmt.Errorf("tests failed: %s", report.ErrorMessage)
	}

	return nil
}
