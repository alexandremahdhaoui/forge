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
	"strings"
	"time"

	"github.com/alexandremahdhaoui/forge/internal/licenseheader"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/google/uuid"
)

// Run implements the TestRunnerFunc for verifying license headers.
// NOTE: spec parameter is currently unused - using input.RootDir instead.
// This is kept for API consistency with the generated TestRunnerFunc type.
func Run(ctx context.Context, input mcptypes.RunInput, spec *Spec) (*forge.TestReport, error) {
	_ = spec // unused for now, kept for API consistency
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

// verifyLicenses performs the license verification and returns results.
// Returns (filesWithoutLicense, totalFiles, error).
//
// File discovery and header detection live in internal/licenseheader,
// shared with go-license-header (which fixes what this only reports) and
// rust-license-header.
func verifyLicenses(rootDir string) ([]string, int, error) {
	goFiles, err := licenseheader.FindFiles(rootDir, []string{".go"}, licenseheader.DefaultExcludeDirs())
	if err != nil {
		return nil, 0, fmt.Errorf("error finding Go files: %w", err)
	}

	if len(goFiles) == 0 {
		return []string{}, 0, nil
	}

	var filesWithoutLicense []string
	for _, file := range goFiles {
		hasLicense, err := licenseheader.HasHeader(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking %s: %v\n", file, err)
			continue
		}
		if !hasLicense {
			filesWithoutLicense = append(filesWithoutLicense, file)
		}
	}

	return filesWithoutLicense, len(goFiles), nil
}
