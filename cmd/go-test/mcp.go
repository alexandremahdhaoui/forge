package main

import (
	"context"
	"fmt"
	"log"

	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// runMCPServer starts the go-test MCP server with stdio transport.
func runMCPServer() error {
	server := mcpserver.New("go-test", Version)

	config := engineframework.TestRunnerConfig{
		Name:        "go-test",
		Version:     Version,
		RunTestFunc: runTestsWrapper,
	}

	if err := engineframework.RegisterTestRunnerTools(server, config); err != nil {
		return err
	}

	return server.RunDefault()
}

// runTestsWrapper implements the TestRunnerFunc for running Go tests
func runTestsWrapper(ctx context.Context, input mcptypes.RunInput) (*forge.TestReport, error) {
	log.Printf("Running tests: stage=%s name=%s", input.Stage, input.Name)

	// Run tests (pass tmpDir if provided, otherwise use current directory)
	tmpDir := input.TmpDir
	if tmpDir == "" {
		tmpDir = "." // Fallback to current directory for backward compatibility
	}

	// Pass testenv information to tests via environment variables
	testEnv := make(map[string]string)
	if input.TestenvTmpDir != "" {
		testEnv["FORGE_TESTENV_TMPDIR"] = input.TestenvTmpDir
	}
	if len(input.ArtifactFiles) > 0 {
		// Pass each artifact file as an environment variable
		for key, relPath := range input.ArtifactFiles {
			// Construct absolute path if testenvTmpDir is available
			var absPath string
			if input.TestenvTmpDir != "" {
				absPath = fmt.Sprintf("%s/%s", input.TestenvTmpDir, relPath)
			} else {
				absPath = relPath
			}
			// Convert key to env var name (e.g., "testenv-kind.kubeconfig" -> "FORGE_ARTIFACT_TESTENV_KIND_KUBECONFIG")
			envKey := fmt.Sprintf("FORGE_ARTIFACT_%s", normalizeEnvKey(key))
			testEnv[envKey] = absPath
		}
	}
	if len(input.TestenvMetadata) > 0 {
		// Pass metadata as environment variables
		for key, value := range input.TestenvMetadata {
			envKey := fmt.Sprintf("FORGE_METADATA_%s", normalizeEnvKey(key))
			testEnv[envKey] = value
		}
	}

	report, junitFile, coverageFile, err := runTests(input.Stage, input.Name, tmpDir, testEnv)
	if err != nil {
		return nil, fmt.Errorf("test run failed: %w", err)
	}

	// Store report in artifact store
	if err := storeTestReport(report, junitFile, coverageFile); err != nil {
		// Log warning but don't fail
		log.Printf("Warning: failed to store test report: %v", err)
	}

	// Convert local TestReport to forge.TestReport
	forgeReport := &forge.TestReport{
		Stage:        report.Stage,
		Status:       report.Status,
		ErrorMessage: report.ErrorMessage,
		StartTime:    report.StartTime,
		Duration:     report.Duration,
		TestStats: forge.TestStats{
			Total:   report.TestStats.Total,
			Passed:  report.TestStats.Passed,
			Failed:  report.TestStats.Failed,
			Skipped: report.TestStats.Skipped,
		},
		Coverage: forge.Coverage{
			Percentage: report.Coverage.Percentage,
		},
	}

	// CRITICAL: Return report even if tests failed (Status="failed")
	return forgeReport, nil
}

// normalizeEnvKey converts a key to an environment variable friendly format.
// Example: "testenv-kind.kubeconfig" -> "TESTENV_KIND_KUBECONFIG"
func normalizeEnvKey(key string) string {
	result := ""
	for i := 0; i < len(key); i++ {
		c := key[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			if c >= 'a' && c <= 'z' {
				result += string(c - 32) // Convert to uppercase
			} else {
				result += string(c)
			}
		} else {
			result += "_"
		}
	}
	return result
}
