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

package cmdutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// ExecuteCommand executes a shell command with the given parameters.
//
// Environment variables are merged with the following precedence (highest to lowest):
//  1. Inline env vars (input.Env)
//  2. Env file vars (input.EnvFile)
//  3. System environment
//
// Returns ExecuteOutput with exit code, stdout, stderr, and any error message.
func ExecuteCommand(input ExecuteInput) ExecuteOutput {
	// Create command
	cmd := exec.Command(input.Command, input.Args...)

	// Set working directory if specified
	if input.WorkDir != "" {
		cmd.Dir = input.WorkDir
	}

	// Merge environment variables
	// Start with system environment
	env := os.Environ()

	// Load and merge env file if specified
	if input.EnvFile != "" {
		envFileVars, err := LoadEnvFile(input.EnvFile)
		if err != nil {
			return ExecuteOutput{
				ExitCode: -1,
				Error:    fmt.Sprintf("failed to load env file: %v", err),
			}
		}
		// Merge envFile vars
		for key, value := range envFileVars {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Merge inline env vars (highest precedence)
	for key, value := range input.Env {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	cmd.Env = env

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()

	output := ExecuteOutput{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		// Get exit code from error
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
