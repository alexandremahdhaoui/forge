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

	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
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

	var specEnv map[string]string
	if spec != nil {
		specEnv = spec.Env
	}

	testEnv := engineframework.BuildTestEnv(input, specEnv)

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
