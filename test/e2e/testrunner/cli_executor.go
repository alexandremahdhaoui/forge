//go:build e2e || unit

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

package testrunner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// ExecuteCLIStep runs a forge CLI subcommand and returns parsed output.
//
//  1. Renders step.Input values through templates.
//  2. Builds command: binary [globalFlags] <step.Command> [--<flag>=<value>...] [positional args]
//  3. Runs via exec.Command, captures stdout and stderr separately.
//  4. Parses stdout as JSON if valid JSON, otherwise stores as stdout string.
//  5. Returns result map with exitCode included.
//
// Special input keys:
//   - _args: positional arguments appended after the command
//   - _globalFlags: map of flags prepended before the command (e.g., --cwd)
func ExecuteCLIStep(binary string, step Step, data *TemplateData) (map[string]interface{}, error) {
	// Render input values.
	var renderedInput map[string]interface{}
	if step.Input != nil {
		var err error
		renderedInput, err = RenderMapValues(step.Input, data)
		if err != nil {
			return nil, fmt.Errorf("cli step %q: rendering input: %w", step.Command, err)
		}
	}

	// Build command arguments.
	var args []string

	// Handle global flags (prepended before the command).
	if gf, ok := renderedInput["_globalFlags"]; ok {
		if gfMap, ok := gf.(map[string]interface{}); ok {
			for key, val := range gfMap {
				args = append(args, fmt.Sprintf("--%s=%v", key, val))
			}
		}
	}

	// Add the subcommand.
	if step.Command != "" {
		args = append(args, step.Command)
	}

	// Add flags and collect positional args.
	var positionalArgs []string
	for key, val := range renderedInput {
		if strings.HasPrefix(key, "_") {
			// Handle positional args (_args).
			if key == "_args" {
				switch v := val.(type) {
				case []interface{}:
					for _, item := range v {
						positionalArgs = append(positionalArgs, fmt.Sprintf("%v", item))
					}
				default:
					positionalArgs = append(positionalArgs, fmt.Sprintf("%v", v))
				}
			}
			// Skip all underscore-prefixed keys (including _globalFlags, _args).
			continue
		}
		args = append(args, fmt.Sprintf("--%s=%v", key, val))
	}
	args = append(args, positionalArgs...)

	cmd := exec.Command(binary, args...)

	// Set working directory if CWD is specified.
	if data.CWD != "" {
		cmd.Dir = data.CWD
	}

	// Capture stdout and stderr separately.
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Build result map with exit code.
	result := make(map[string]interface{})
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("cli step %q: running command: %w", step.Command, err)
		}
	}
	result["exitCode"] = float64(exitCode)

	// Store stderr.
	stderrStr := strings.TrimSpace(stderr.String())
	if stderrStr != "" {
		result["stderr"] = stderrStr
	}

	// Parse stdout as JSON if non-empty.
	stdoutStr := strings.TrimSpace(stdout.String())
	if stdoutStr != "" {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(stdoutStr), &parsed); err != nil {
			// If stdout is not valid JSON, store it as raw output.
			result["stdout"] = stdoutStr
		} else {
			// Merge parsed JSON into result.
			for k, v := range parsed {
				result[k] = v
			}
		}
	}

	return result, nil
}
