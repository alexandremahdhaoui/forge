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
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/alexandremahdhaoui/forge/internal/cli"
)

// Version information (set via ldflags during build)
var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
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

func main() {
	cli.Bootstrap(cli.Config{
		Name:           "generic-test-runner",
		Version:        Version,
		CommitSHA:      CommitSHA,
		BuildTimestamp: BuildTimestamp,
		RunMCP:         runMCPServer,
	})
}
