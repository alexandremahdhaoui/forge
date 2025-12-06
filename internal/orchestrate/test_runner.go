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

package orchestrate

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

// TestRunnerOrchestrator orchestrates multiple test runner engines in sequence.
// It executes each test runner and merges their test reports into a single aggregate report.
type TestRunnerOrchestrator struct {
	callMCP    MCPCaller
	resolveURI EngineResolver
}

// NewTestRunnerOrchestrator creates a new test runner orchestrator.
func NewTestRunnerOrchestrator(callMCP MCPCaller, resolveURI EngineResolver) *TestRunnerOrchestrator {
	return &TestRunnerOrchestrator{
		callMCP:    callMCP,
		resolveURI: resolveURI,
	}
}

// Orchestrate executes multiple test runners sequentially and merges reports.
// All runners receive the same base params (with runner-specific config injected).
// Execution is sequential - if any runner fails, the entire orchestration fails (fail-fast).
// Test reports from all runners are merged into a single aggregate report.
func (o *TestRunnerOrchestrator) Orchestrate(
	runnerSpecs []forge.TestRunnerSpec,
	params map[string]any,
) (*forge.TestReport, error) {
	if len(runnerSpecs) == 0 {
		return nil, fmt.Errorf("no test runner engines provided")
	}

	// Merged report starts empty
	var mergedReport *forge.TestReport

	// Execute each test runner in sequence
	for i, runnerSpec := range runnerSpecs {
		// Resolve engine URI to command and args
		command, args, err := o.resolveURI(runnerSpec.Engine)
		if err != nil {
			return nil, fmt.Errorf("runner[%d] %s: failed to resolve engine: %w",
				i, runnerSpec.Engine, err)
		}

		// Prepare params for this runner (clone and inject config)
		runnerParams := make(map[string]any)
		for k, v := range params {
			runnerParams[k] = v
		}

		// Inject runner-specific config from EngineSpec
		if runnerSpec.Spec.Command != "" {
			runnerParams["command"] = runnerSpec.Spec.Command
		}
		if len(runnerSpec.Spec.Args) > 0 {
			runnerParams["args"] = runnerSpec.Spec.Args
		}
		if len(runnerSpec.Spec.Env) > 0 {
			runnerParams["env"] = runnerSpec.Spec.Env
		}
		if runnerSpec.Spec.EnvFile != "" {
			runnerParams["envFile"] = runnerSpec.Spec.EnvFile
		}
		if runnerSpec.Spec.WorkDir != "" {
			runnerParams["workDir"] = runnerSpec.Spec.WorkDir
		}

		// Call test runner
		result, err := o.callMCP(command, args, "run", runnerParams)
		if err != nil {
			return nil, fmt.Errorf("runner[%d] %s: run failed: %w",
				i, runnerSpec.Engine, err)
		}

		// Parse test report from result
		report, err := parseTestReport(result)
		if err != nil {
			return nil, fmt.Errorf("runner[%d] %s: failed to parse test report: %w",
				i, runnerSpec.Engine, err)
		}

		// Merge into aggregate report
		if mergedReport == nil {
			// First runner - use its report as base
			mergedReport = report
		} else {
			// Subsequent runners - merge into existing report
			mergedReport = mergeTestReports(mergedReport, report)
		}
	}

	return mergedReport, nil
}

// parseTestReport converts MCP result to TestReport.
// Adapted from cmd/forge/test.go:492-530 (storeTestReportFromResult).
func parseTestReport(result interface{}) (*forge.TestReport, error) {
	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("result is not a map")
	}

	// Marshal and unmarshal to convert to TestReport struct
	reportJSON, err := json.Marshal(resultMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var report forge.TestReport
	if err := json.Unmarshal(reportJSON, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal test report: %w", err)
	}

	return &report, nil
}

// mergeTestReports merges two test reports into one aggregate report.
// The merged report contains:
// - Same ID and stage as the base report
// - Summed test statistics (total, passed, failed, skipped)
// - Combined duration
// - "failed" status if any runner failed, else "passed"
// - Averaged coverage percentage
// - Merged artifact files and output paths
// - Concatenated error messages
func mergeTestReports(base *forge.TestReport, additional *forge.TestReport) *forge.TestReport {
	merged := &forge.TestReport{
		// Keep original ID and stage from base
		ID:    base.ID,
		Stage: base.Stage,

		// Keep earliest start time
		StartTime: base.StartTime,
	}

	// Merge test statistics (sum all counts)
	merged.TestStats.Total = base.TestStats.Total + additional.TestStats.Total
	merged.TestStats.Passed = base.TestStats.Passed + additional.TestStats.Passed
	merged.TestStats.Failed = base.TestStats.Failed + additional.TestStats.Failed
	merged.TestStats.Skipped = base.TestStats.Skipped + additional.TestStats.Skipped

	// Overall status: "failed" if any runner failed, else "passed"
	if base.Status == "failed" || additional.Status == "failed" {
		merged.Status = "failed"
	} else if base.Status == "passed" && additional.Status == "passed" {
		merged.Status = "passed"
	} else {
		// Default to failed if status is unknown
		merged.Status = "failed"
	}

	// Merge durations (sum)
	merged.Duration = base.Duration + additional.Duration

	// Merge coverage - only from Enabled sources
	baseEnabled := base.Coverage.Enabled
	additionalEnabled := additional.Coverage.Enabled

	if baseEnabled && additionalEnabled {
		// Both have coverage enabled - weighted average by test count
		if base.TestStats.Total > 0 && additional.TestStats.Total > 0 {
			totalTests := float64(base.TestStats.Total + additional.TestStats.Total)
			merged.Coverage.Enabled = true
			merged.Coverage.Percentage = (base.Coverage.Percentage*float64(base.TestStats.Total) +
				additional.Coverage.Percentage*float64(additional.TestStats.Total)) / totalTests
		} else if base.TestStats.Total > 0 {
			merged.Coverage = base.Coverage
		} else if additional.TestStats.Total > 0 {
			merged.Coverage = additional.Coverage
		} else {
			// Both enabled but zero tests - use base
			merged.Coverage = base.Coverage
		}
	} else if baseEnabled {
		// Only base has coverage
		merged.Coverage = base.Coverage
	} else if additionalEnabled {
		// Only additional has coverage
		merged.Coverage = additional.Coverage
	}
	// If neither enabled, merged.Coverage.Enabled stays false (zero value)

	// Merge coverage file paths (comma-separated) - only from enabled sources
	coverageFiles := []string{}
	if baseEnabled && base.Coverage.FilePath != "" {
		coverageFiles = append(coverageFiles, base.Coverage.FilePath)
	}
	if additionalEnabled && additional.Coverage.FilePath != "" {
		coverageFiles = append(coverageFiles, additional.Coverage.FilePath)
	}
	if len(coverageFiles) > 0 {
		merged.Coverage.FilePath = strings.Join(coverageFiles, ",")
	}

	// Merge artifact files (append)
	merged.ArtifactFiles = append(base.ArtifactFiles, additional.ArtifactFiles...)

	// Merge output paths (comma-separated)
	outputPaths := []string{}
	if base.OutputPath != "" {
		outputPaths = append(outputPaths, base.OutputPath)
	}
	if additional.OutputPath != "" {
		outputPaths = append(outputPaths, additional.OutputPath)
	}
	if len(outputPaths) > 0 {
		merged.OutputPath = strings.Join(outputPaths, ",")
	}

	// Merge error messages (semicolon-separated)
	errorMessages := []string{}
	if base.ErrorMessage != "" {
		errorMessages = append(errorMessages, base.ErrorMessage)
	}
	if additional.ErrorMessage != "" {
		errorMessages = append(errorMessages, additional.ErrorMessage)
	}
	if len(errorMessages) > 0 {
		merged.ErrorMessage = strings.Join(errorMessages, "; ")
	}

	// Update timestamp to now
	merged.StartTime = time.Now()

	return merged
}
