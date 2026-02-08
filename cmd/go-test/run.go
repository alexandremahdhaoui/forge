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

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// Run implements the TestRunnerFunc for running Go tests.
func Run(ctx context.Context, input mcptypes.RunInput, spec *Spec) (*forge.TestReport, error) {
	log.Printf("Running tests: stage=%s name=%s", input.Stage, input.Name)

	tmpDir := input.TmpDir
	if tmpDir == "" {
		tmpDir = "."
	}

	testEnv := make(map[string]string)

	// Apply EnvPropagation filtering for testenv environment variables
	if len(input.TestenvEnv) > 0 {
		if input.EnvPropagation != nil && input.EnvPropagation.Disabled {
			log.Println("EnvPropagation disabled - skipping testenv environment variables")
		} else if input.EnvPropagation != nil && len(input.EnvPropagation.Whitelist) > 0 {
			for _, key := range input.EnvPropagation.Whitelist {
				if value, ok := input.TestenvEnv[key]; ok {
					testEnv[key] = value
				}
			}
		} else if input.EnvPropagation != nil && len(input.EnvPropagation.Blacklist) > 0 {
			for key, value := range input.TestenvEnv {
				if !contains(input.EnvPropagation.Blacklist, key) {
					testEnv[key] = value
				}
			}
		} else {
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
		for key, relPath := range input.ArtifactFiles {
			var absPath string
			if input.TestenvTmpDir != "" {
				absPath = fmt.Sprintf("%s/%s", input.TestenvTmpDir, relPath)
			} else {
				absPath = relPath
			}
			envKey := fmt.Sprintf("FORGE_ARTIFACT_%s", normalizeEnvKey(key))
			testEnv[envKey] = absPath
		}
	}
	if len(input.TestenvMetadata) > 0 {
		for key, value := range input.TestenvMetadata {
			envKey := fmt.Sprintf("FORGE_METADATA_%s", normalizeEnvKey(key))
			testEnv[envKey] = value
		}
	}

	// Override with runner-specific environment variables
	if len(input.Env) > 0 {
		for key, value := range input.Env {
			testEnv[key] = value
		}
	}

	// Apply spec-level environment variables
	if spec != nil && len(spec.Env) > 0 {
		for key, value := range spec.Env {
			testEnv[key] = value
		}
	}

	report, junitFile, coverageFile, err := runTests(input.Stage, input.Name, tmpDir, spec, testEnv)
	if err != nil {
		return nil, fmt.Errorf("test run failed: %w", err)
	}

	if err := storeTestReport(report, junitFile, coverageFile); err != nil {
		log.Printf("Warning: failed to store test report: %v", err)
	}

	forgeReport := &forge.TestReport{
		ID:           input.ID,
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

	return forgeReport, nil
}

func normalizeEnvKey(key string) string {
	result := ""
	for i := 0; i < len(key); i++ {
		c := key[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			if c >= 'a' && c <= 'z' {
				result += string(c - 32)
			} else {
				result += string(c)
			}
		} else {
			result += "_"
		}
	}
	return result
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
