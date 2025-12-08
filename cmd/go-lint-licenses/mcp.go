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

	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/google/uuid"
)

// runMCPServer starts the go-lint-licenses MCP server with stdio transport.
func runMCPServer() error {
	server := mcpserver.New(Name, Version)

	config := engineframework.TestRunnerConfig{
		Name:        Name,
		Version:     Version,
		RunTestFunc: runTestsWrapper,
	}

	if err := engineframework.RegisterTestRunnerTools(server, config); err != nil {
		return err
	}

	if err := enginedocs.RegisterDocsTools(server, *docsConfig); err != nil {
		return err
	}

	return server.RunDefault()
}

// runTestsWrapper implements the TestRunnerFunc for verifying license headers
func runTestsWrapper(ctx context.Context, input mcptypes.RunInput) (*forge.TestReport, error) {
	log.Printf("Verifying license headers: stage=%s", input.Stage)

	startTime := time.Now()

	// Default root directory
	rootDir := "."
	if input.RootDir != "" {
		rootDir = input.RootDir
	}

	// Run verification
	filesWithoutLicense, totalFiles, err := verifyLicenses(rootDir)
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
			Passed:  totalFiles - len(filesWithoutLicense),
			Failed:  len(filesWithoutLicense),
			Skipped: 0,
		},
		Coverage: forge.Coverage{
			Percentage: 0, // No coverage for verify-license
		},
	}

	if err != nil {
		report.Status = "failed"
		report.ErrorMessage = fmt.Sprintf("Verification failed: %v", err)
		report.TestStats = forge.TestStats{Total: 0, Passed: 0, Failed: 0, Skipped: 0}

		// CRITICAL: Return report with error message, but nil error
		return report, nil
	}

	if len(filesWithoutLicense) > 0 {
		report.Status = "failed"

		// Build detailed error message
		var details strings.Builder
		details.WriteString(fmt.Sprintf("Found %d file(s) without license headers out of %d total files", len(filesWithoutLicense), totalFiles))
		details.WriteString("\n\nFiles missing license headers:\n")
		for _, file := range filesWithoutLicense {
			details.WriteString(fmt.Sprintf("  - %s\n", file))
		}
		details.WriteString("\nGo files must have one of these license header patterns:\n")
		details.WriteString("  // Copyright ...\n")
		details.WriteString("  // SPDX-License-Identifier: ...\n")
		details.WriteString("  // Licensed under ...\n")

		report.ErrorMessage = details.String()

		// CRITICAL: Return report with error message, but nil error
		return report, nil
	}

	report.Status = "passed"

	return report, nil
}
