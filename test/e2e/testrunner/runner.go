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
	"fmt"
	"log"
)

// RunTestCase executes a single test case: setup, steps, then teardown.
//
//  1. Initializes the Steps map if nil.
//  2. Runs setup steps (harness commands or CLI/MCP steps).
//  3. Runs test steps with assertions.
//  4. Runs teardown steps (always, logging errors).
//  5. Captures results in data.Steps[step.ID] if step has Capture.
func RunTestCase(data *TemplateData, tc TestCase) error {
	if data.Steps == nil {
		data.Steps = make(map[string]map[string]interface{})
	}

	// Run setup steps.
	for i, step := range tc.Setup {
		result, err := executeStep(data, step)
		if err != nil {
			return fmt.Errorf("setup step %d (%s): %w", i, stepName(step), err)
		}
		captureResult(data, step, result)
	}

	// Run test steps.
	for i, step := range tc.Steps {
		result, err := executeStep(data, step)
		if err != nil {
			return fmt.Errorf("step %d (%s): %w", i, stepName(step), err)
		}

		// Run assertions.
		if step.Expected != nil && result != nil {
			renderedExpected, err := RenderMapValues(step.Expected, data)
			if err != nil {
				return fmt.Errorf("step %d (%s): rendering expected: %w", i, stepName(step), err)
			}
			if err := AssertResult(result, renderedExpected); err != nil {
				return fmt.Errorf("step %d (%s): %w", i, stepName(step), err)
			}
		}

		captureResult(data, step, result)
	}

	// Run teardown steps (always run, log errors).
	for i, step := range tc.Teardown {
		result, err := executeStep(data, step)
		if err != nil {
			log.Printf("teardown step %d (%s): %v", i, stepName(step), err)
		}
		captureResult(data, step, result)
	}

	return nil
}

// executeStep dispatches a step to the appropriate executor based on its mode.
func executeStep(data *TemplateData, step Step) (map[string]interface{}, error) {
	// Harness commands have no mode field; they are identified by command.
	if step.Mode == "" {
		return executeHarnessStep(data, step)
	}

	switch step.Mode {
	case "cli":
		return ExecuteCLIStep(data.Binary, step, data)
	case "mcp":
		return nil, fmt.Errorf("mcp mode not yet implemented")
	default:
		return nil, fmt.Errorf("unknown step mode %q", step.Mode)
	}
}

// executeHarnessStep runs a harness command (write-forge-yaml, write-file, etc.).
func executeHarnessStep(data *TemplateData, step Step) (map[string]interface{}, error) {
	// Render input values.
	var renderedInput map[string]interface{}
	if step.Input != nil {
		var err error
		renderedInput, err = RenderMapValues(step.Input, data)
		if err != nil {
			return nil, fmt.Errorf("harness step %q: rendering input: %w", step.Command, err)
		}
	}

	switch step.Command {
	case "write-forge-yaml":
		if err := writeForgeYAML(data, renderedInput); err != nil {
			return nil, err
		}
		return nil, nil
	case "write-file":
		// write-file renders its own templates from the raw input.
		if err := writeFile(data, step.Input); err != nil {
			return nil, err
		}
		return nil, nil
	case "check-file-exists":
		return checkFileExists(data, renderedInput)
	case "check-file-absent":
		if err := checkFileAbsent(data, renderedInput); err != nil {
			return nil, err
		}
		return nil, nil
	case "set-env":
		if err := setEnv(data, renderedInput); err != nil {
			return nil, err
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown harness command %q", step.Command)
	}
}

// captureResult stores step results in data.Steps[step.ID] for use in
// subsequent template rendering.
func captureResult(data *TemplateData, step Step, result map[string]interface{}) {
	if step.ID == "" || step.Capture == nil || result == nil {
		return
	}

	captured := make(map[string]interface{}, len(step.Capture))
	for varName, fieldName := range step.Capture {
		if val, ok := result[fieldName]; ok {
			captured[varName] = val
		}
	}
	data.Steps[step.ID] = captured
}

// stepName returns a human-readable name for a step.
func stepName(step Step) string {
	if step.ID != "" {
		return step.ID
	}
	if step.Tool != "" {
		return step.Tool
	}
	if step.Command != "" {
		return step.Command
	}
	return "unnamed"
}
