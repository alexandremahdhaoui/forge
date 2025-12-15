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

	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// runMCPServer starts the generic-test-runner MCP server with stdio transport.
func runMCPServer() error {
	server, err := SetupMCPServer(Name, Version, runTests)
	if err != nil {
		return err
	}

	if err := enginedocs.RegisterDocsTools(server, *docsConfig); err != nil {
		return err
	}

	// Run the MCP server
	return server.RunDefault()
}

// runTests is the core business logic for executing a test command.
// It implements the TestRunnerFunc signature defined in zz_generated.mcp.go.
func runTests(ctx context.Context, input mcptypes.RunInput, spec *Spec) (*forge.TestReport, error) {
	// Use spec values, falling back to input values for compatibility
	command := spec.Command
	if command == "" {
		command = input.Command
	}

	args := spec.Args
	if len(args) == 0 {
		args = input.Args
	}

	env := spec.Env
	if len(env) == 0 {
		env = input.Env
	}

	envFile := spec.EnvFile
	if envFile == "" {
		envFile = input.EnvFile
	}

	workDir := spec.WorkDir
	if workDir == "" {
		workDir = input.WorkDir
	}

	log.Printf("Running tests: stage=%s name=%s command=%s", input.Stage, input.Name, command)

	// Validate required fields
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}

	// Execute command
	execInput := ExecuteInput{
		Command: command,
		Args:    args,
		Env:     env,
		EnvFile: envFile,
		WorkDir: workDir,
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
