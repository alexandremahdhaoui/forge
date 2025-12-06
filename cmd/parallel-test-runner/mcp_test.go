//go:build unit

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
	"testing"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -----------------------------------------------------------------------------
// Tests for parseTestReport
// -----------------------------------------------------------------------------

func TestParseTestReport_NilResponse(t *testing.T) {
	report, err := parseTestReport(nil)
	require.Error(t, err)
	assert.Nil(t, report)
	assert.Contains(t, err.Error(), "nil response")
}

func TestParseTestReport_ValidTestReportMap(t *testing.T) {
	startTime := time.Now().UTC()
	resp := map[string]any{
		"id":        "test-123",
		"stage":     "unit",
		"status":    "passed",
		"startTime": startTime.Format(time.RFC3339Nano),
		"duration":  1.5,
		"testStats": map[string]any{
			"total":   10,
			"passed":  8,
			"failed":  1,
			"skipped": 1,
		},
		"coverage": map[string]any{
			"enabled":    true,
			"percentage": 85.5,
			"filePath":   "/path/to/coverage.out",
		},
	}

	report, err := parseTestReport(resp)
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Equal(t, "test-123", report.ID)
	assert.Equal(t, "unit", report.Stage)
	assert.Equal(t, "passed", report.Status)
	assert.Equal(t, 1.5, report.Duration)
	assert.Equal(t, 10, report.TestStats.Total)
	assert.Equal(t, 8, report.TestStats.Passed)
	assert.Equal(t, 1, report.TestStats.Failed)
	assert.Equal(t, 1, report.TestStats.Skipped)
	assert.True(t, report.Coverage.Enabled)
	assert.Equal(t, 85.5, report.Coverage.Percentage)
	assert.Equal(t, "/path/to/coverage.out", report.Coverage.FilePath)
}

func TestParseTestReport_PartialFields(t *testing.T) {
	// Test with only required fields
	resp := map[string]any{
		"id":     "test-456",
		"stage":  "integration",
		"status": "failed",
	}

	report, err := parseTestReport(resp)
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Equal(t, "test-456", report.ID)
	assert.Equal(t, "integration", report.Stage)
	assert.Equal(t, "failed", report.Status)
	// Verify defaults
	assert.Equal(t, float64(0), report.Duration)
	assert.Equal(t, 0, report.TestStats.Total)
	assert.False(t, report.Coverage.Enabled)
	assert.Equal(t, float64(0), report.Coverage.Percentage)
}

func TestParseTestReport_WithErrorMessage(t *testing.T) {
	resp := map[string]any{
		"id":           "test-error",
		"stage":        "e2e",
		"status":       "failed",
		"errorMessage": "test execution failed: timeout",
	}

	report, err := parseTestReport(resp)
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Equal(t, "failed", report.Status)
	assert.Equal(t, "test execution failed: timeout", report.ErrorMessage)
}

func TestParseTestReport_EmptyMap(t *testing.T) {
	resp := map[string]any{}

	report, err := parseTestReport(resp)
	require.NoError(t, err)
	require.NotNil(t, report)

	// Empty map should result in zero values
	assert.Equal(t, "", report.ID)
	assert.Equal(t, "", report.Stage)
	assert.Equal(t, "", report.Status)
}

// -----------------------------------------------------------------------------
// Tests for mapToStruct
// -----------------------------------------------------------------------------

func TestMapToStruct_NilMap(t *testing.T) {
	var spec ParallelTestRunnerSpec
	err := mapToStruct(nil, &spec)
	require.NoError(t, err)
	// Struct should remain at zero values
	assert.Equal(t, "", spec.PrimaryCoverageRunner)
	assert.Nil(t, spec.Runners)
}

func TestMapToStruct_ValidMap(t *testing.T) {
	m := map[string]any{
		"primaryCoverageRunner": "go-test",
		"runners": []any{
			map[string]any{
				"name":   "go-test",
				"engine": "go://go-test",
				"spec": map[string]any{
					"packages": "./...",
				},
			},
			map[string]any{
				"name":   "go-lint",
				"engine": "go://go-lint",
				"spec":   map[string]any{},
			},
		},
	}

	var spec ParallelTestRunnerSpec
	err := mapToStruct(m, &spec)
	require.NoError(t, err)

	assert.Equal(t, "go-test", spec.PrimaryCoverageRunner)
	require.Len(t, spec.Runners, 2)
	assert.Equal(t, "go-test", spec.Runners[0].Name)
	assert.Equal(t, "go://go-test", spec.Runners[0].Engine)
	assert.Equal(t, "./...", spec.Runners[0].Spec["packages"])
	assert.Equal(t, "go-lint", spec.Runners[1].Name)
	assert.Equal(t, "go://go-lint", spec.Runners[1].Engine)
}

func TestMapToStruct_InvalidStruct(t *testing.T) {
	m := map[string]any{
		"runners": "not an array", // Invalid type for runners field
	}

	var spec ParallelTestRunnerSpec
	err := mapToStruct(m, &spec)
	// Should error because "not an array" cannot be unmarshaled into []RunnerConfig
	require.Error(t, err)
}

func TestMapToStruct_EmptyMap(t *testing.T) {
	m := map[string]any{}

	var spec ParallelTestRunnerSpec
	err := mapToStruct(m, &spec)
	require.NoError(t, err)

	// Should result in zero values
	assert.Equal(t, "", spec.PrimaryCoverageRunner)
	assert.Nil(t, spec.Runners)
}

func TestMapToStruct_ExtraFields(t *testing.T) {
	// Map with extra fields that don't exist in the struct
	m := map[string]any{
		"primaryCoverageRunner": "go-test",
		"unknownField":          "should be ignored",
		"runners":               []any{},
	}

	var spec ParallelTestRunnerSpec
	err := mapToStruct(m, &spec)
	require.NoError(t, err)

	assert.Equal(t, "go-test", spec.PrimaryCoverageRunner)
	assert.Empty(t, spec.Runners)
}

// -----------------------------------------------------------------------------
// Tests for coverage aggregation logic
// These test the aggregation behavior that would occur in parallelRun
// -----------------------------------------------------------------------------

// aggregateCoverage simulates the coverage selection logic from parallelRun
func aggregateCoverage(reports []*forge.TestReport, reportByName map[string]*forge.TestReport, primaryCoverageRunner string) forge.Coverage {
	var coverage forge.Coverage

	if primaryCoverageRunner != "" {
		if primary, ok := reportByName[primaryCoverageRunner]; ok {
			coverage = primary.Coverage
		}
	}
	// If no primary specified or not found, Coverage.Enabled stays false (zero value)

	return coverage
}

func TestAggregateCoverage_PrimaryRunnerSelection(t *testing.T) {
	reports := []*forge.TestReport{
		{
			ID:    "1",
			Stage: "unit",
			Coverage: forge.Coverage{
				Enabled:    true,
				Percentage: 85.5,
				FilePath:   "/path/to/coverage1.out",
			},
		},
		{
			ID:    "2",
			Stage: "lint",
			Coverage: forge.Coverage{
				Enabled:    false, // Lint tools don't produce coverage
				Percentage: 0,
			},
		},
	}

	reportByName := map[string]*forge.TestReport{
		"go-test": reports[0],
		"go-lint": reports[1],
	}

	// Select go-test as primary coverage runner
	coverage := aggregateCoverage(reports, reportByName, "go-test")
	assert.True(t, coverage.Enabled)
	assert.Equal(t, 85.5, coverage.Percentage)
	assert.Equal(t, "/path/to/coverage1.out", coverage.FilePath)
}

func TestAggregateCoverage_PrimaryRunnerNotFound(t *testing.T) {
	reports := []*forge.TestReport{
		{
			ID:    "1",
			Stage: "unit",
			Coverage: forge.Coverage{
				Enabled:    true,
				Percentage: 85.5,
			},
		},
	}

	reportByName := map[string]*forge.TestReport{
		"go-test": reports[0],
	}

	// Primary runner name doesn't match any runner
	coverage := aggregateCoverage(reports, reportByName, "non-existent-runner")
	assert.False(t, coverage.Enabled)
	assert.Equal(t, float64(0), coverage.Percentage)
}

func TestAggregateCoverage_NoPrimarySpecified(t *testing.T) {
	reports := []*forge.TestReport{
		{
			ID:    "1",
			Stage: "unit",
			Coverage: forge.Coverage{
				Enabled:    true,
				Percentage: 85.5,
			},
		},
	}

	reportByName := map[string]*forge.TestReport{
		"go-test": reports[0],
	}

	// No primary coverage runner specified
	coverage := aggregateCoverage(reports, reportByName, "")
	assert.False(t, coverage.Enabled)
	assert.Equal(t, float64(0), coverage.Percentage)
}

// -----------------------------------------------------------------------------
// Tests for TestStats summing logic
// -----------------------------------------------------------------------------

// sumTestStats simulates the TestStats aggregation logic from parallelRun
func sumTestStats(reports []*forge.TestReport) forge.TestStats {
	var stats forge.TestStats
	for _, report := range reports {
		stats.Total += report.TestStats.Total
		stats.Passed += report.TestStats.Passed
		stats.Failed += report.TestStats.Failed
		stats.Skipped += report.TestStats.Skipped
	}
	return stats
}

func TestSumTestStats_MultipleReports(t *testing.T) {
	reports := []*forge.TestReport{
		{
			TestStats: forge.TestStats{
				Total:   100,
				Passed:  90,
				Failed:  5,
				Skipped: 5,
			},
		},
		{
			TestStats: forge.TestStats{
				Total:   50,
				Passed:  45,
				Failed:  3,
				Skipped: 2,
			},
		},
		{
			TestStats: forge.TestStats{
				Total:   25,
				Passed:  25,
				Failed:  0,
				Skipped: 0,
			},
		},
	}

	stats := sumTestStats(reports)
	assert.Equal(t, 175, stats.Total)
	assert.Equal(t, 160, stats.Passed)
	assert.Equal(t, 8, stats.Failed)
	assert.Equal(t, 7, stats.Skipped)
}

func TestSumTestStats_EmptyReports(t *testing.T) {
	reports := []*forge.TestReport{}

	stats := sumTestStats(reports)
	assert.Equal(t, 0, stats.Total)
	assert.Equal(t, 0, stats.Passed)
	assert.Equal(t, 0, stats.Failed)
	assert.Equal(t, 0, stats.Skipped)
}

func TestSumTestStats_SingleReport(t *testing.T) {
	reports := []*forge.TestReport{
		{
			TestStats: forge.TestStats{
				Total:   100,
				Passed:  95,
				Failed:  3,
				Skipped: 2,
			},
		},
	}

	stats := sumTestStats(reports)
	assert.Equal(t, 100, stats.Total)
	assert.Equal(t, 95, stats.Passed)
	assert.Equal(t, 3, stats.Failed)
	assert.Equal(t, 2, stats.Skipped)
}

// -----------------------------------------------------------------------------
// Tests for status determination logic
// -----------------------------------------------------------------------------

// determineOverallStatus simulates the status determination logic from parallelRun
func determineOverallStatus(reports []*forge.TestReport, errors []error) string {
	// If any runner errors occurred, mark as failed
	if len(errors) > 0 {
		return "failed"
	}

	// Any failure = overall failure
	for _, report := range reports {
		if report.Status == "failed" {
			return "failed"
		}
	}

	return "passed"
}

func TestDetermineOverallStatus_AllPassed(t *testing.T) {
	reports := []*forge.TestReport{
		{Status: "passed"},
		{Status: "passed"},
		{Status: "passed"},
	}

	status := determineOverallStatus(reports, nil)
	assert.Equal(t, "passed", status)
}

func TestDetermineOverallStatus_AnyFailed(t *testing.T) {
	reports := []*forge.TestReport{
		{Status: "passed"},
		{Status: "failed"},
		{Status: "passed"},
	}

	status := determineOverallStatus(reports, nil)
	assert.Equal(t, "failed", status)
}

func TestDetermineOverallStatus_AllFailed(t *testing.T) {
	reports := []*forge.TestReport{
		{Status: "failed"},
		{Status: "failed"},
	}

	status := determineOverallStatus(reports, nil)
	assert.Equal(t, "failed", status)
}

func TestDetermineOverallStatus_WithErrors(t *testing.T) {
	reports := []*forge.TestReport{
		{Status: "passed"},
	}
	errors := []error{
		assert.AnError,
	}

	status := determineOverallStatus(reports, errors)
	assert.Equal(t, "failed", status)
}

func TestDetermineOverallStatus_EmptyReports(t *testing.T) {
	reports := []*forge.TestReport{}

	status := determineOverallStatus(reports, nil)
	assert.Equal(t, "passed", status)
}

// -----------------------------------------------------------------------------
// Tests for RunnerConfig and ParallelTestRunnerSpec
// -----------------------------------------------------------------------------

func TestRunnerConfig_JSONMarshaling(t *testing.T) {
	m := map[string]any{
		"name":   "test-runner",
		"engine": "go://go-test",
		"spec": map[string]any{
			"packages": "./pkg/...",
			"verbose":  true,
		},
	}

	var config RunnerConfig
	err := mapToStruct(m, &config)
	require.NoError(t, err)

	assert.Equal(t, "test-runner", config.Name)
	assert.Equal(t, "go://go-test", config.Engine)
	assert.Equal(t, "./pkg/...", config.Spec["packages"])
	assert.Equal(t, true, config.Spec["verbose"])
}

func TestParallelTestRunnerSpec_EmptyRunners(t *testing.T) {
	m := map[string]any{
		"runners": []any{},
	}

	var spec ParallelTestRunnerSpec
	err := mapToStruct(m, &spec)
	require.NoError(t, err)

	assert.Empty(t, spec.Runners)
	assert.Equal(t, "", spec.PrimaryCoverageRunner)
}

func TestParallelTestRunnerSpec_MultipleDifferentRunners(t *testing.T) {
	m := map[string]any{
		"primaryCoverageRunner": "go-test",
		"runners": []any{
			map[string]any{
				"name":   "go-test",
				"engine": "go://go-test",
				"spec": map[string]any{
					"packages": "./...",
					"coverage": true,
				},
			},
			map[string]any{
				"name":   "go-lint",
				"engine": "go://go-lint",
				"spec": map[string]any{
					"checks": []any{"all"},
				},
			},
			map[string]any{
				"name":   "go-lint-licenses",
				"engine": "go://go-lint-licenses",
				"spec":   map[string]any{},
			},
		},
	}

	var spec ParallelTestRunnerSpec
	err := mapToStruct(m, &spec)
	require.NoError(t, err)

	assert.Equal(t, "go-test", spec.PrimaryCoverageRunner)
	require.Len(t, spec.Runners, 3)

	// Verify each runner
	assert.Equal(t, "go-test", spec.Runners[0].Name)
	assert.Equal(t, "go://go-test", spec.Runners[0].Engine)
	assert.Equal(t, true, spec.Runners[0].Spec["coverage"])

	assert.Equal(t, "go-lint", spec.Runners[1].Name)
	assert.Equal(t, "go://go-lint", spec.Runners[1].Engine)

	assert.Equal(t, "go-lint-licenses", spec.Runners[2].Name)
	assert.Equal(t, "go://go-lint-licenses", spec.Runners[2].Engine)
}
