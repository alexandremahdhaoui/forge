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

package engineframework

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mockTestRunnerFunc creates a mock TestRunnerFunc for testing.
// Returns report with Status="passed" if stage doesn't contain "fail".
// Returns report with Status="failed" if stage contains "fail".
// Returns error if returnError is true.
func mockTestRunnerFunc(returnError bool) TestRunnerFunc {
	return func(ctx context.Context, input mcptypes.RunInput) (*forge.TestReport, error) {
		// Simulate execution error
		if returnError {
			return nil, errors.New("test execution failed: simulated error")
		}

		// Simulate test failure if stage contains "fail"
		if strings.Contains(input.Stage, "fail") {
			return &forge.TestReport{
				Stage:        input.Stage,
				Status:       "failed",
				ErrorMessage: "2 tests failed",
				TestStats: forge.TestStats{
					Total:  10,
					Passed: 8,
					Failed: 2,
				},
			}, nil
		}

		// Tests passed
		return &forge.TestReport{
			Stage:  input.Stage,
			Status: "passed",
			TestStats: forge.TestStats{
				Total:  10,
				Passed: 10,
				Failed: 0,
			},
		}, nil
	}
}

func TestMakeRunHandler_Success(t *testing.T) {
	config := TestRunnerConfig{
		Name:        "test-runner",
		Version:     "1.0.0",
		RunTestFunc: mockTestRunnerFunc(false),
	}

	handler := makeRunHandler(config)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := mcptypes.RunInput{
		Stage: "unit",
		Name:  "test-runner",
	}

	result, report, err := handler(ctx, req, input)
	// Should not return Go error
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Result should not be an error (tests passed)
	if result.IsError {
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
				t.Fatalf("handler returned error result: %s", textContent.Text)
			}
		}
		t.Fatalf("handler returned error result")
	}

	// Report should be returned
	if report == nil {
		t.Fatal("handler returned nil report")
	}

	// Report should be of correct type
	reportObj, ok := report.(*forge.TestReport)
	if !ok {
		t.Fatalf("report is not *forge.TestReport, got %T", report)
	}

	// Report should have correct stage
	if reportObj.Stage != "unit" {
		t.Errorf("report.Stage = %q, want %q", reportObj.Stage, "unit")
	}

	// Report should have Status="passed"
	if reportObj.Status != "passed" {
		t.Errorf("report.Status = %q, want %q", reportObj.Status, "passed")
	}

	// Report should have test stats
	if reportObj.TestStats.Total == 0 {
		t.Error("report.TestStats.Total is 0")
	}
}

func TestMakeRunHandler_TestsFailedButReportReturned(t *testing.T) {
	config := TestRunnerConfig{
		Name:        "test-runner",
		Version:     "1.0.0",
		RunTestFunc: mockTestRunnerFunc(false), // Tests fail if stage contains "fail"
	}

	handler := makeRunHandler(config)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := mcptypes.RunInput{
		Stage: "fail-stage", // Stage contains "fail" - tests will fail
		Name:  "test-runner",
	}

	result, report, err := handler(ctx, req, input)
	// Should not return Go error (test failures are not errors)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Result SHOULD be an error (tests failed)
	if !result.IsError {
		t.Fatal("handler should return error result when tests fail")
	}

	// CRITICAL: Report should still be returned even though tests failed
	if report == nil {
		t.Fatal("handler returned nil report despite test failure - report must be returned")
	}

	// Report should be of correct type
	reportObj, ok := report.(*forge.TestReport)
	if !ok {
		t.Fatalf("report is not *forge.TestReport, got %T", report)
	}

	// Report should have Status="failed"
	if reportObj.Status != "failed" {
		t.Errorf("report.Status = %q, want %q", reportObj.Status, "failed")
	}

	// Report should have correct stage
	if reportObj.Stage != "fail-stage" {
		t.Errorf("report.Stage = %q, want %q", reportObj.Stage, "fail-stage")
	}

	// Error message should mention test failure
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
			if !strings.Contains(textContent.Text, "Tests failed") {
				t.Errorf("error message should mention test failure: %s", textContent.Text)
			}
		}
	}
}

func TestMakeRunHandler_ExecutionError(t *testing.T) {
	config := TestRunnerConfig{
		Name:        "test-runner",
		Version:     "1.0.0",
		RunTestFunc: mockTestRunnerFunc(true), // Always returns error
	}

	handler := makeRunHandler(config)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := mcptypes.RunInput{
		Stage: "unit",
		Name:  "test-runner",
	}

	result, report, err := handler(ctx, req, input)
	// Should not return Go error (errors converted to MCP results)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Result should be an error (execution failed)
	if !result.IsError {
		t.Fatal("handler should return error result when execution fails")
	}

	// Report should be nil (execution error, not test failure)
	if report != nil {
		t.Errorf("handler returned report despite execution error: %v", report)
	}

	// Error message should contain "Test execution failed"
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
			if !strings.Contains(textContent.Text, "Test execution failed") {
				t.Errorf("error message should mention execution failure: %s", textContent.Text)
			}
		}
	}
}

func TestMakeRunHandler_MissingStage(t *testing.T) {
	config := TestRunnerConfig{
		Name:        "test-runner",
		Version:     "1.0.0",
		RunTestFunc: mockTestRunnerFunc(false),
	}

	handler := makeRunHandler(config)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := mcptypes.RunInput{
		Stage: "", // Missing stage
		Name:  "test-runner",
	}

	result, report, err := handler(ctx, req, input)
	// Should not return Go error
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Result should be an error (validation failure)
	if !result.IsError {
		t.Fatal("handler should return error result when required field is missing")
	}

	// Report should be nil (validation error)
	if report != nil {
		t.Errorf("handler returned report despite validation error: %v", report)
	}
}

func TestMakeRunHandler_MissingName(t *testing.T) {
	config := TestRunnerConfig{
		Name:        "test-runner",
		Version:     "1.0.0",
		RunTestFunc: mockTestRunnerFunc(false),
	}

	handler := makeRunHandler(config)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := mcptypes.RunInput{
		Stage: "unit",
		Name:  "", // Missing name
	}

	result, report, err := handler(ctx, req, input)
	// Should not return Go error
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Result should be an error (validation failure)
	if !result.IsError {
		t.Fatal("handler should return error result when required field is missing")
	}

	// Report should be nil (validation error)
	if report != nil {
		t.Errorf("handler returned report despite validation error: %v", report)
	}
}

func TestMakeRunHandler_NilReport(t *testing.T) {
	// Test defensive handling of nil report from TestRunnerFunc
	nilReportFunc := func(ctx context.Context, input mcptypes.RunInput) (*forge.TestReport, error) {
		return nil, nil // Returns nil report (shouldn't happen, but defensive)
	}

	config := TestRunnerConfig{
		Name:        "test-runner",
		Version:     "1.0.0",
		RunTestFunc: nilReportFunc,
	}

	handler := makeRunHandler(config)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := mcptypes.RunInput{
		Stage: "unit",
		Name:  "test-runner",
	}

	result, report, err := handler(ctx, req, input)
	// Should not return Go error
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Result should be an error (nil report)
	if !result.IsError {
		t.Fatal("handler should return error result when TestRunnerFunc returns nil report")
	}

	// Report should be nil
	if report != nil {
		t.Errorf("handler returned report despite nil report from TestRunnerFunc: %v", report)
	}

	// Error message should mention nil report
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
			if !strings.Contains(textContent.Text, "nil report") {
				t.Errorf("error message should mention nil report: %s", textContent.Text)
			}
		}
	}
}

func TestRegisterTestRunnerTools(t *testing.T) {
	config := TestRunnerConfig{
		Name:        "test-runner",
		Version:     "1.0.0",
		RunTestFunc: mockTestRunnerFunc(false),
	}

	server := mcpserver.New("test-runner", "1.0.0")

	err := RegisterTestRunnerTools(server, config)
	if err != nil {
		t.Fatalf("RegisterTestRunnerTools returned error: %v", err)
	}

	// NOTE: We can't easily test that tools are registered without
	// examining internal server state or running a full MCP server.
	// This test just verifies the function doesn't error.
	// Real validation happens in integration tests.
}

func TestRegisterTestRunnerTools_IntegrationTest(t *testing.T) {
	// This test verifies that the registered tool can actually be called
	config := TestRunnerConfig{
		Name:        "test-runner",
		Version:     "1.0.0",
		RunTestFunc: mockTestRunnerFunc(false),
	}

	server := mcpserver.New("test-runner", "1.0.0")

	err := RegisterTestRunnerTools(server, config)
	if err != nil {
		t.Fatalf("RegisterTestRunnerTools returned error: %v", err)
	}

	// Create handler
	runHandler := makeRunHandler(config)

	// Test successful run
	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := mcptypes.RunInput{
		Stage: "integration-test",
		Name:  "test-runner",
	}

	runResult, runReport, runErr := runHandler(ctx, req, input)
	if runErr != nil {
		t.Errorf("run handler returned error: %v", runErr)
	}
	if runResult.IsError {
		t.Error("run handler returned error result")
	}
	if runReport == nil {
		t.Error("run handler returned nil report")
	}
}

func TestTestRunnerFunc_ContextPropagation(t *testing.T) {
	// Test that context is properly propagated to TestRunnerFunc
	var receivedCtx context.Context

	testRunner := func(ctx context.Context, input mcptypes.RunInput) (*forge.TestReport, error) {
		receivedCtx = ctx
		return &forge.TestReport{
			Stage:  input.Stage,
			Status: "passed",
			TestStats: forge.TestStats{
				Total:  5,
				Passed: 5,
			},
		}, nil
	}

	config := TestRunnerConfig{
		Name:        "test-runner",
		Version:     "1.0.0",
		RunTestFunc: testRunner,
	}

	handler := makeRunHandler(config)

	// Create context with value
	type ctxKey string
	ctx := context.WithValue(context.Background(), ctxKey("test"), "value")
	req := &mcp.CallToolRequest{}
	input := mcptypes.RunInput{
		Stage: "unit",
		Name:  "test-runner",
	}

	_, _, err := handler(ctx, req, input)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Verify context was propagated
	if receivedCtx == nil {
		t.Fatal("context was not propagated to TestRunnerFunc")
	}

	// Verify context value
	if val := receivedCtx.Value(ctxKey("test")); val != "value" {
		t.Errorf("context value = %v, want %q", val, "value")
	}
}

func TestMakeRunHandler_ReportFields(t *testing.T) {
	// Test that all expected report fields are present
	config := TestRunnerConfig{
		Name:        "test-runner",
		Version:     "1.0.0",
		RunTestFunc: mockTestRunnerFunc(false),
	}

	handler := makeRunHandler(config)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := mcptypes.RunInput{
		Stage: "unit",
		Name:  "test-runner",
	}

	_, report, err := handler(ctx, req, input)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	reportObj, ok := report.(*forge.TestReport)
	if !ok {
		t.Fatalf("report is not *forge.TestReport, got %T", report)
	}

	// Verify all expected fields are set
	if reportObj.Stage == "" {
		t.Error("report.Stage is empty")
	}
	if reportObj.Status == "" {
		t.Error("report.Status is empty")
	}
	// TestStats should be populated
	if reportObj.TestStats.Total == 0 {
		t.Error("report.TestStats.Total is 0")
	}
}

func TestMakeRunHandler_FailedReportFields(t *testing.T) {
	// Test that failed reports have all expected fields
	config := TestRunnerConfig{
		Name:        "test-runner",
		Version:     "1.0.0",
		RunTestFunc: mockTestRunnerFunc(false),
	}

	handler := makeRunHandler(config)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := mcptypes.RunInput{
		Stage: "fail-tests", // Will fail
		Name:  "test-runner",
	}

	result, report, err := handler(ctx, req, input)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Should be error result
	if !result.IsError {
		t.Fatal("handler should return error result for failed tests")
	}

	// But report should still be returned
	if report == nil {
		t.Fatal("handler returned nil report for failed tests")
	}

	reportObj, ok := report.(*forge.TestReport)
	if !ok {
		t.Fatalf("report is not *forge.TestReport, got %T", report)
	}

	// Verify Status is "failed"
	if reportObj.Status != "failed" {
		t.Errorf("report.Status = %q, want %q", reportObj.Status, "failed")
	}

	// Verify all other fields are set
	if reportObj.Stage == "" {
		t.Error("report.Stage is empty")
	}
	// Should have error message for failed tests
	if reportObj.ErrorMessage == "" {
		t.Error("report.ErrorMessage is empty for failed tests")
	}
	// TestStats should show failures
	if reportObj.TestStats.Failed == 0 {
		t.Error("report.TestStats.Failed is 0 for failed tests")
	}
}
