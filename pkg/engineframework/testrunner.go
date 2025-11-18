package engineframework

import (
	"context"
	"fmt"
	"log"

	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/alexandremahdhaoui/forge/pkg/mcputil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestRunnerFunc is the signature for test execution.
//
// Implementations must:
//   - Validate input fields (required fields should be checked)
//   - Execute tests (run test commands, collect results)
//   - Return TestReport on success (with Status "passed" or "failed")
//   - Return TestReport even on test failure (Status field indicates pass/fail)
//   - Return error only for execution failures (not test failures)
//
// The framework handles:
//   - MCP tool registration
//   - Result formatting
//   - Error conversion to MCP responses
//   - Report return even on test failure
//
// IMPORTANT: Test failures are NOT errors. Return a TestReport with Status="failed".
// Only return error for execution failures (can't run tests, can't parse results, etc.).
//
// Example:
//
//	func myTestRunnerFunc(ctx context.Context, input mcptypes.RunInput) (*forge.TestReport, error) {
//	    // Execute tests
//	    output, err := runTests(input.Stage)
//	    if err != nil {
//	        // Execution error - couldn't run tests
//	        return nil, fmt.Errorf("failed to execute tests: %w", err)
//	    }
//
//	    // Parse results
//	    report := parseTestOutput(output)
//
//	    // Return report (even if tests failed)
//	    // Framework will use ErrorResultWithArtifact for failed tests
//	    return report, nil
//	}
type TestRunnerFunc func(ctx context.Context, input mcptypes.RunInput) (*forge.TestReport, error)

// TestRunnerConfig configures test runner tool registration.
//
// Fields:
//   - Name: Engine name (e.g., "go-test", "generic-test-runner")
//   - Version: Engine version string (e.g., "1.0.0" or git commit hash)
//   - RunTestFunc: The test execution implementation function
//
// Example:
//
//	config := TestRunnerConfig{
//	    Name:        "my-test-runner",
//	    Version:     "1.0.0",
//	    RunTestFunc: myTestRunnerFunc,
//	}
type TestRunnerConfig struct {
	Name        string         // Engine name (e.g., "go-test")
	Version     string         // Engine version
	RunTestFunc TestRunnerFunc // Test execution implementation
}

// RegisterTestRunnerTools registers the run tool with the MCP server.
//
// This function automatically:
//   - Registers "run" tool that calls the RunTestFunc
//   - Validates required input fields (Stage, Runner)
//   - Converts TestRunnerFunc errors to MCP error responses
//   - Returns TestReport as artifact even when tests fail
//   - Uses ErrorResultWithArtifact for failed tests (report still returned)
//   - Uses SuccessResultWithArtifact for passed tests
//
// Parameters:
//   - server: The MCP server instance
//   - config: TestRunner configuration with Name, Version, and RunTestFunc
//
// Returns:
//   - nil on success
//   - error if tool registration fails (e.g., duplicate tool names)
//
// Example:
//
//	func runMCPServer() error {
//	    server := mcpserver.New("my-test-runner", "1.0.0")
//
//	    config := TestRunnerConfig{
//	        Name:        "my-test-runner",
//	        Version:     "1.0.0",
//	        RunTestFunc: myTestRunnerFunc,
//	    }
//
//	    if err := RegisterTestRunnerTools(server, config); err != nil {
//	        return err
//	    }
//
//	    return server.RunDefault()
//	}
func RegisterTestRunnerTools(server *mcpserver.Server, config TestRunnerConfig) error {
	// Register run tool
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "run",
		Description: fmt.Sprintf("Run tests using %s", config.Name),
	}, makeRunHandler(config))

	return nil
}

// makeRunHandler creates an MCP handler function from a TestRunnerFunc.
//
// The returned handler:
//   - Validates required input fields (Stage, Runner)
//   - Calls the TestRunnerFunc with the input
//   - Converts TestRunnerFunc errors to MCP error responses
//   - Returns TestReport as artifact even when tests fail
//   - Uses ErrorResultWithArtifact for failed tests (Status="failed")
//   - Uses SuccessResultWithArtifact for passed tests (Status="passed")
//
// This is an internal helper function used by RegisterTestRunnerTools.
func makeRunHandler(config TestRunnerConfig) func(context.Context, *mcp.CallToolRequest, mcptypes.RunInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input mcptypes.RunInput) (*mcp.CallToolResult, any, error) {
		log.Printf("Running tests for stage %s using %s", input.Stage, config.Name)

		// Validate required input fields
		if result := mcputil.ValidateRequiredWithPrefix("Test run failed", map[string]string{
			"stage": input.Stage,
			"name":  input.Name,
		}); result != nil {
			return result, nil, nil
		}

		// Call the TestRunnerFunc
		report, err := config.RunTestFunc(ctx, input)
		if err != nil {
			// Execution error (couldn't run tests)
			return mcputil.ErrorResult(fmt.Sprintf("Test execution failed: %v", err)), nil, nil
		}

		// Check if report is nil (shouldn't happen, but defensive)
		if report == nil {
			return mcputil.ErrorResult("Test runner returned nil report"), nil, nil
		}

		// Return result based on test status
		// IMPORTANT: Even if tests failed, we return the report as an artifact
		if report.Status == "failed" {
			// Tests failed - use ErrorResultWithArtifact
			// Create summary message
			summary := fmt.Sprintf("%d/%d tests failed", report.TestStats.Failed, report.TestStats.Total)
			if report.ErrorMessage != "" {
				summary = report.ErrorMessage
			}

			result, returnedReport := mcputil.ErrorResultWithArtifact(
				fmt.Sprintf("Tests failed for stage %s: %s", input.Stage, summary),
				report,
			)
			return result, returnedReport, nil
		}

		// Tests passed - use SuccessResultWithArtifact
		summary := fmt.Sprintf("%d/%d tests passed", report.TestStats.Passed, report.TestStats.Total)
		result, returnedReport := mcputil.SuccessResultWithArtifact(
			fmt.Sprintf("Tests passed for stage %s: %s", input.Stage, summary),
			report,
		)
		return result, returnedReport, nil
	}
}
