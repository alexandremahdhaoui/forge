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
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/alexandremahdhaoui/forge/internal/mcpcaller"
	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ParallelTestRunnerSpec defines the input specification for parallel test runner.
// It is parsed from RunInput.Spec field.
type ParallelTestRunnerSpec struct {
	// PrimaryCoverageRunner is the name of the runner whose coverage is used.
	// If not specified or runner not found, Coverage.Enabled=false in result.
	PrimaryCoverageRunner string `json:"primaryCoverageRunner,omitempty"`

	// Runners is the list of test runners to execute in parallel.
	Runners []RunnerConfig `json:"runners"`
}

// RunnerConfig defines a single sub-runner configuration.
type RunnerConfig struct {
	// Name is required and used for coverage selection.
	Name string `json:"name"`

	// Engine must be a go:// URI (not alias://).
	Engine string `json:"engine"`

	// Spec is passed to the sub-runner.
	Spec map[string]any `json:"spec"`
}

// runnerResult holds the result from a single runner goroutine.
type runnerResult struct {
	report *forge.TestReport
	err    error
	name   string
}

// runMCPServer starts the parallel-test-runner MCP server.
func runMCPServer() error {
	server := mcpserver.New(Name, Version)

	config := engineframework.TestRunnerConfig{
		Name:        Name,
		Version:     Version,
		RunTestFunc: parallelRun,
	}

	if err := engineframework.RegisterTestRunnerTools(server, config); err != nil {
		return err
	}

	if err := enginedocs.RegisterDocsTools(server, *docsConfig); err != nil {
		return err
	}

	// Register config-validate tool
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "config-validate",
		Description: "Validate parallel-test-runner configuration and recursively validate sub-runners",
	}, handleConfigValidate)

	return server.RunDefault()
}

// parallelRun executes multiple test runners in parallel.
func parallelRun(ctx context.Context, input mcptypes.RunInput) (*forge.TestReport, error) {
	// Parse spec from input
	var spec ParallelTestRunnerSpec
	if err := mapToStruct(input.Spec, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse spec: %w", err)
	}

	if len(spec.Runners) == 0 {
		return nil, fmt.Errorf("no runners specified")
	}

	// Create MCP caller
	caller := mcpcaller.NewCaller(Version)
	mcpCall := caller.GetMCPCaller()
	resolveEngine := caller.GetEngineResolver()

	// Results channel
	results := make(chan runnerResult, len(spec.Runners))

	// WaitGroup for goroutines
	var wg sync.WaitGroup
	startTime := time.Now()

	// Launch parallel runners
	for _, runner := range spec.Runners {
		wg.Add(1)
		go func(r RunnerConfig) {
			defer wg.Done()

			// Resolve engine URI to command and args
			command, args, err := resolveEngine(r.Engine)
			if err != nil {
				results <- runnerResult{err: fmt.Errorf("[%s] failed to resolve engine: %w", r.Name, err), name: r.Name}
				return
			}

			// Build the run input for the sub-runner
			// Pass through key fields from parent input, but use runner's spec
			runInput := map[string]any{
				"id":    input.ID,
				"stage": input.Stage,
				"name":  r.Name,
				"spec":  r.Spec,
			}

			// Pass through directory params if present
			if input.TmpDir != "" {
				runInput["tmpDir"] = input.TmpDir
			}
			if input.BuildDir != "" {
				runInput["buildDir"] = input.BuildDir
			}
			if input.RootDir != "" {
				runInput["rootDir"] = input.RootDir
			}

			// Pass through testenv fields if present
			if len(input.ArtifactFiles) > 0 {
				runInput["artifactFiles"] = input.ArtifactFiles
			}
			if input.TestenvTmpDir != "" {
				runInput["testenvTmpDir"] = input.TestenvTmpDir
			}
			if len(input.TestenvMetadata) > 0 {
				runInput["testenvMetadata"] = input.TestenvMetadata
			}
			if len(input.TestenvEnv) > 0 {
				runInput["testenvEnv"] = input.TestenvEnv
			}
			if input.EnvPropagation != nil {
				runInput["envPropagation"] = input.EnvPropagation
			}

			// Call run tool on sub-runner
			resp, err := mcpCall(command, args, "run", runInput)
			if err != nil {
				results <- runnerResult{err: fmt.Errorf("[%s] MCP call failed: %w", r.Name, err), name: r.Name}
				return
			}

			// Parse test report from response
			report, err := parseTestReport(resp)
			if err != nil {
				results <- runnerResult{err: fmt.Errorf("[%s] failed to parse report: %w", r.Name, err), name: r.Name}
				return
			}

			results <- runnerResult{report: report, name: r.Name}
		}(runner)
	}

	// Wait for all goroutines and close channel
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var reports []*forge.TestReport
	var errors []error
	reportByName := make(map[string]*forge.TestReport)

	for r := range results {
		if r.err != nil {
			errors = append(errors, r.err)
		} else if r.report != nil {
			reports = append(reports, r.report)
			reportByName[r.name] = r.report
		}
	}

	// Aggregate test report
	aggregated := &forge.TestReport{
		ID:        input.ID,
		Stage:     input.Stage,
		StartTime: startTime,
		Duration:  time.Since(startTime).Seconds(),
	}

	// Sum test stats from all runners
	for _, report := range reports {
		aggregated.TestStats.Total += report.TestStats.Total
		aggregated.TestStats.Passed += report.TestStats.Passed
		aggregated.TestStats.Failed += report.TestStats.Failed
		aggregated.TestStats.Skipped += report.TestStats.Skipped
	}

	// Select coverage from primary runner only (NOT averaging)
	// If no primary specified or not found, Coverage.Enabled stays false
	if spec.PrimaryCoverageRunner != "" {
		if primary, ok := reportByName[spec.PrimaryCoverageRunner]; ok {
			aggregated.Coverage = primary.Coverage
		}
	}

	// Determine overall status
	// Any failure = overall failure
	aggregated.Status = "passed"
	for _, report := range reports {
		if report.Status == "failed" {
			aggregated.Status = "failed"
			break
		}
	}

	// If any runner errors occurred, mark as failed
	if len(errors) > 0 {
		aggregated.Status = "failed"
		aggregated.ErrorMessage = fmt.Sprintf("some runners failed: %v", errors)
	}

	return aggregated, nil
}

// parseTestReport parses a test report from MCP response.
func parseTestReport(resp interface{}) (*forge.TestReport, error) {
	if resp == nil {
		return nil, fmt.Errorf("nil response")
	}

	// Convert response to JSON and back to TestReport
	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var report forge.TestReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to TestReport: %w", err)
	}

	return &report, nil
}

// mapToStruct converts map to struct using JSON marshaling.
func mapToStruct(m map[string]any, v interface{}) error {
	if m == nil {
		return nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
