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
	"time"

	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// runMCPServer starts the generic-test-runner MCP server with stdio transport.
func runMCPServer() error {
	server := mcpserver.New("generic-test-runner", Version)

	// Configure test runner with engineframework
	config := engineframework.TestRunnerConfig{
		Name:        "generic-test-runner",
		Version:     Version,
		RunTestFunc: runTests,
	}

	// Register test runner tools (registers 'run')
	if err := engineframework.RegisterTestRunnerTools(server, config); err != nil {
		return err
	}

	// Run the MCP server
	return server.RunDefault()
}

// runTests is the core business logic for executing a test command.
// It implements engineframework.TestRunnerFunc.
func runTests(ctx context.Context, input mcptypes.RunInput) (*forge.TestReport, error) {
	log.Printf("Running tests: stage=%s name=%s command=%s", input.Stage, input.Name, input.Command)

	// Validate required fields
	if input.Command == "" {
		return nil, fmt.Errorf("command is required")
	}

	// Execute command
	execInput := ExecuteInput{
		Command: input.Command,
		Args:    input.Args,
		Env:     input.Env,
		EnvFile: input.EnvFile,
		WorkDir: input.WorkDir,
	}

	output := executeCommand(execInput)

	// Create test report based on exit code
	// CRITICAL: Return report even if tests failed (Status="failed")
	status := "passed"
	passed := 1
	failed := 0
	errorMessage := ""

	if output.ExitCode != 0 {
		status = "failed"
		passed = 0
		failed = 1
		errorMessage = fmt.Sprintf("Command exited with code %d", output.ExitCode)
		if output.Error != "" {
			errorMessage += fmt.Sprintf(": %s", output.Error)
		}
	}

	// Log output
	if output.Stdout != "" {
		log.Printf("Stdout: %s", output.Stdout)
	}
	if output.Stderr != "" {
		log.Printf("Stderr: %s", output.Stderr)
	}

	report := &forge.TestReport{
		Stage:        input.Stage,
		Status:       status,
		ErrorMessage: errorMessage,
		StartTime:    time.Now().UTC(),
		Duration:     0, // Duration not tracked for generic test runner
		TestStats: forge.TestStats{
			Total:  1,
			Passed: passed,
			Failed: failed,
		},
		Coverage: forge.Coverage{
			Percentage: 0.0, // Coverage not tracked for generic test runner
		},
	}

	return report, nil
}
