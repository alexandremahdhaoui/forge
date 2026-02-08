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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/google/uuid"
)

// run executes tests for the given stage and generates a structured report.
// Test output goes to stderr, JSON report goes to stdout.
// runTests executes the test suite using gotestsum and returns a structured report along with artifact file paths.
// testEnv contains environment variables to pass to the test process (e.g., artifact file paths, metadata).
func runTests(stage, name, tmpDir string, spec *Spec, testEnv map[string]string) (*TestReport, string, string, error) {
	startTime := time.Now()

	// Generate output file paths in tmpDir
	junitFile := filepath.Join(tmpDir, fmt.Sprintf("test-%s-%s.xml", stage, name))
	coverageFile := filepath.Join(tmpDir, fmt.Sprintf("test-%s-%s-coverage.out", stage, name))

	// Build gotestsum command
	args := []string{
		"run", "gotest.tools/gotestsum@v1.13.0",
		"--format", "pkgname-and-test-fails",
		"--format-hide-empty-pkg",
		"--junitfile", junitFile,
		"--",
	}

	// Tags: spec.Tags overrides default (stage name)
	tags := stage
	if spec != nil && len(spec.Tags) > 0 {
		tags = strings.Join(spec.Tags, ",")
	}
	args = append(args, "-tags", tags)

	// Race: always enabled (default behavior, no way to opt-out with current spec schema)
	args = append(args, "-race")

	args = append(args, "-count=1")

	// Timeout: spec.Timeout adds -timeout flag (no default = go test default of 10m)
	if spec != nil && spec.Timeout != "" {
		args = append(args, "-timeout", spec.Timeout)
	}

	// Cover: always enabled (default behavior)
	args = append(args, "-cover", "-coverprofile", coverageFile)

	// Additional args from spec
	if spec != nil && len(spec.Args) > 0 {
		args = append(args, spec.Args...)
	}

	// Packages: spec.Packages overrides default (./...)
	if spec != nil && len(spec.Packages) > 0 {
		args = append(args, spec.Packages...)
	} else {
		args = append(args, "./...")
	}

	cmd := exec.Command("go", args...)

	// Inherit current environment and add testenv variables
	cmd.Env = os.Environ()
	for key, value := range testEnv {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Redirect test output to stderr so JSON report can go to stdout
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	// Execute the command
	err := cmd.Run()
	duration := time.Since(startTime).Seconds()

	// Determine status based on exit code
	status := "passed"
	errorMessage := ""
	if err != nil {
		status = "failed"
		if exitErr, ok := err.(*exec.ExitError); ok {
			errorMessage = fmt.Sprintf("tests failed with exit code %d", exitErr.ExitCode())
		} else {
			errorMessage = fmt.Sprintf("failed to execute tests: %v", err)
		}
	}

	// Parse test statistics from JUnit XML (will be implemented in Task 2.3)
	testStats, statsErr := parseJUnitXML(junitFile)
	if statsErr != nil {
		// If we can't parse stats, create empty stats but don't fail
		testStats = &TestStats{}
	}

	// Parse coverage information (will be implemented in Task 2.3)
	coverage, coverageErr := parseCoverage(coverageFile)
	if coverageErr != nil {
		// If we can't parse coverage, create empty coverage but don't fail
		// Enabled is still true because go-test always attempts coverage
		coverage = &Coverage{Enabled: true, FilePath: coverageFile}
	}

	// Create test report
	report := &TestReport{
		Stage:        stage,
		Name:         name,
		Status:       status,
		StartTime:    startTime,
		Duration:     duration,
		TestStats:    *testStats,
		Coverage:     *coverage,
		OutputPath:   junitFile,
		ErrorMessage: errorMessage,
	}

	return report, junitFile, coverageFile, nil
}

// storeTestReport stores the test report in the artifact store.
func storeTestReport(report *TestReport, junitFile, coverageFile string) error {
	// Get artifact store path (environment variable takes precedence)
	artifactStorePath := os.Getenv("FORGE_ARTIFACT_STORE_PATH")
	if artifactStorePath == "" {
		// Read forge.yaml to get the artifact store path
		config, err := forge.ReadSpec()
		if err != nil {
			return fmt.Errorf("failed to read forge.yaml: %w", err)
		}
		artifactStorePath, err = forge.GetArtifactStorePath(config.ArtifactStorePath)
		if err != nil {
			return fmt.Errorf("failed to get artifact store path: %w", err)
		}
	}

	// Read or create artifact store
	store, err := forge.ReadOrCreateArtifactStore(artifactStorePath)
	if err != nil {
		return fmt.Errorf("failed to read artifact store: %w", err)
	}

	// Generate report ID (UUID)
	reportID := uuid.New().String()

	// Build list of artifact files
	var artifactFiles []string
	if junitFile != "" {
		artifactFiles = append(artifactFiles, junitFile)
	}
	if coverageFile != "" {
		artifactFiles = append(artifactFiles, coverageFile)
	}

	// Create TestReport for artifact store
	storeReport := &forge.TestReport{
		ID:            reportID,
		Stage:         report.Stage,
		Status:        report.Status,
		StartTime:     report.StartTime,
		Duration:      report.Duration,
		TestStats:     forge.TestStats(report.TestStats),
		Coverage:      forge.Coverage(report.Coverage),
		ArtifactFiles: artifactFiles,
		OutputPath:    report.OutputPath,
		ErrorMessage:  report.ErrorMessage,
	}

	// Add or update test report
	forge.AddOrUpdateTestReport(&store, storeReport)

	// Write artifact store
	if err := forge.WriteArtifactStore(artifactStorePath, store); err != nil {
		return fmt.Errorf("failed to write artifact store: %w", err)
	}

	return nil
}
