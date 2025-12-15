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
	"strings"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/google/uuid"
)

// runMCPServer starts the go-lint-tags MCP server with stdio transport.
func runMCPServer() error {
	server, err := SetupMCPServer(Name, Version, runTests)
	if err != nil {
		return err
	}

	if err := enginedocs.RegisterDocsTools(server, *docsConfig); err != nil {
		return err
	}

	return server.RunDefault()
}

// runTests implements the TestRunnerFunc for verifying build tags
func runTests(ctx context.Context, input mcptypes.RunInput, spec *Spec) (*forge.TestReport, error) {
	log.Printf("Verifying build tags: stage=%s", input.Stage)

	startTime := time.Now()

	// Default root directory - prefer spec, then input, then current dir
	rootDir := "."
	if spec != nil && spec.RootDir != "" {
		rootDir = spec.RootDir
	} else if input.RootDir != "" {
		rootDir = input.RootDir
	}

	// Run verification
	filesWithoutTags, totalFiles, err := verifyTags(rootDir)
	duration := time.Since(startTime).Seconds()

	// Generate report ID
	reportID := uuid.New().String()

	// Build base report
	report := &forge.TestReport{
		ID:        reportID,
		Stage:     input.Stage,
		StartTime: startTime,
		Duration:  duration,
		TestStats: forge.TestStats{
			Total:   totalFiles,
			Passed:  totalFiles - len(filesWithoutTags),
			Failed:  len(filesWithoutTags),
			Skipped: 0,
		},
		Coverage: forge.Coverage{
			Percentage: 0, // No coverage for verify-tags
		},
	}

	if err != nil {
		report.Status = "failed"
		report.ErrorMessage = fmt.Sprintf("Verification failed: %v", err)
		report.TestStats = forge.TestStats{Total: 0, Passed: 0, Failed: 0, Skipped: 0}

		// CRITICAL: Return report with error message, but nil error
		return report, nil
	}

	if len(filesWithoutTags) > 0 {
		report.Status = "failed"

		// Build detailed error message
		var details strings.Builder
		details.WriteString(fmt.Sprintf("Found %d test file(s) without build tags out of %d total files", len(filesWithoutTags), totalFiles))
		details.WriteString("\n\nFiles missing build tags:\n")
		for _, file := range filesWithoutTags {
			details.WriteString(fmt.Sprintf("  - %s\n", file))
		}
		details.WriteString("\nTest files must have one of these build tags:\n")
		details.WriteString("  //go:build unit\n")
		details.WriteString("  //go:build integration\n")
		details.WriteString("  //go:build e2e\n")

		report.ErrorMessage = details.String()

		// CRITICAL: Return report with error message, but nil error
		return report, nil
	}

	report.Status = "passed"

	return report, nil
}
