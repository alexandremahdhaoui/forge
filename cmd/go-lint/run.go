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

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// Run implements the TestRunnerFunc for running Go linter.
// It implements the TestRunnerFunc signature defined in zz_generated.mcp.go.
func Run(ctx context.Context, input mcptypes.RunInput, spec *Spec) (*forge.TestReport, error) {
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
		ID:           input.ID,
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
