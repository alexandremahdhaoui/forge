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

	// First, merge testenv environment variables (e.g., KUBECONFIG)
	// These are propagated from testenv sub-engines (testenv-kind, testenv-lcr, etc.)
	// Apply EnvPropagation filtering if configured
	if len(input.TestenvEnv) > 0 {
		if input.EnvPropagation != nil && input.EnvPropagation.Disabled {
			// Propagation disabled - skip all testenv env vars
			log.Println("EnvPropagation disabled - skipping testenv environment variables")
		} else if input.EnvPropagation != nil && len(input.EnvPropagation.Whitelist) > 0 {
			// Whitelist mode - only propagate whitelisted vars
			for _, key := range input.EnvPropagation.Whitelist {
				if value, ok := input.TestenvEnv[key]; ok {
					testEnv[key] = value
				}
			}
		} else if input.EnvPropagation != nil && len(input.EnvPropagation.Blacklist) > 0 {
			// Blacklist mode - propagate all except blacklisted vars
			for key, value := range input.TestenvEnv {
				if !contains(input.EnvPropagation.Blacklist, key) {
					testEnv[key] = value
				}
			}
		} else {
			// No filtering - propagate all testenv vars
			for key, value := range input.TestenvEnv {
				testEnv[key] = value
			}
		}
	}

	// Legacy support: Pass testenv metadata via FORGE_* prefixed env vars
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

	// Override with runner-specific environment variables (if provided)
	if len(input.Env) > 0 {
		for key, value := range input.Env {
			testEnv[key] = value
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
			Enabled:    report.Coverage.Enabled,
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

// contains checks if a string slice contains a specific item
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
