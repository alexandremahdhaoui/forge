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
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// ExecuteInput contains the parameters for command execution
type ExecuteInput struct {
	Command string            // Command to execute
	Args    []string          // Command arguments
	Env     map[string]string // Environment variables
	EnvFile string            // Path to environment file (optional)
	WorkDir string            // Working directory (optional)
}

// ExecuteOutput contains the result of command execution
type ExecuteOutput struct {
	ExitCode int    // Command exit code
	Stdout   string // Standard output
	Stderr   string // Standard error
	Error    string // Error message if execution failed
}

// Run is the core business logic for executing a test command.
// It implements the TestRunnerFunc signature defined in zz_generated.mcp.go.
func Run(ctx context.Context, input mcptypes.RunInput, spec *Spec) (*forge.TestReport, error) {
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
		ID:           input.ID,
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

// loadEnvFile loads environment variables from a file
func loadEnvFile(path string) (map[string]string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return make(map[string]string), nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read env file: %w", err)
	}

	envVars := make(map[string]string)
	lines := strings.Split(string(content), "\n")

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimPrefix(line, "export ")
		line = strings.TrimSpace(line)

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format in env file at line %d: %s", lineNum+1, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if len(value) >= 2 {
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}

		envVars[key] = value
	}

	return envVars, nil
}

// executeCommand executes a shell command with the given parameters
func executeCommand(input ExecuteInput) ExecuteOutput {
	cmd := exec.Command(input.Command, input.Args...)

	if input.WorkDir != "" {
		cmd.Dir = input.WorkDir
	}

	env := os.Environ()

	if input.EnvFile != "" {
		envFileVars, err := loadEnvFile(input.EnvFile)
		if err != nil {
			return ExecuteOutput{
				ExitCode: -1,
				Error:    fmt.Sprintf("failed to load env file: %v", err),
			}
		}
		for key, value := range envFileVars {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	for key, value := range input.Env {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := ExecuteOutput{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			output.ExitCode = exitErr.ExitCode()
		} else {
			output.ExitCode = -1
			output.Error = err.Error()
		}
	} else {
		output.ExitCode = 0
	}

	return output
}
