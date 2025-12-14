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
	"os"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// loadConfig loads the forge configuration from forge.yaml or custom path.
func loadConfig() (forge.Spec, error) {
	if configPath != "" {
		return forge.ReadSpecFromPath(configPath)
	}
	return forge.ReadSpec()
}

// runConfig handles the config command
func runConfig(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("config subcommand required (validate)")
	}

	subcommand := args[0]

	switch subcommand {
	case "validate":
		return runConfigValidate(args[1:])
	default:
		return fmt.Errorf("unknown config subcommand: %s (available: validate)", subcommand)
	}
}

// runConfigValidate validates the forge.yaml configuration
func runConfigValidate(args []string) error {
	// Determine config path (default: forge.yaml)
	cfgPath := "forge.yaml"
	if len(args) > 0 {
		cfgPath = args[0]
	}

	// Read the spec from path
	spec, err := forge.ReadSpecFromPath(cfgPath)
	if err != nil {
		return fmt.Errorf("validation failed:\n%v", err)
	}

	// Validate basic structure using existing spec.Validate()
	if err := spec.Validate(); err != nil {
		return fmt.Errorf("validation failed:\n%v", err)
	}

	// Extract all engine URIs from the spec
	engineRefs := extractEngineURIs(spec)

	// If no engines to validate, just show the basic validation result
	if len(engineRefs) == 0 {
		fmt.Printf("Configuration is valid: %s\n", cfgPath)
		fmt.Printf("Project: %s\n", spec.Name)
		fmt.Printf("Build specs: %d\n", len(spec.Build))
		fmt.Printf("Test stages: %d\n", len(spec.Test))
		fmt.Printf("Engine configs: %d\n", len(spec.Engines))
		fmt.Println("No engines to validate")
		return nil
	}

	// Validate each engine's spec
	ctx := context.Background()
	var results []validationResult

	fmt.Printf("Validating configuration: %s\n", cfgPath)
	fmt.Printf("Project: %s\n", spec.Name)
	fmt.Printf("Validating %d engine(s)...\n\n", len(engineRefs))

	for _, ref := range engineRefs {
		fmt.Printf("  Validating %s (%s: %s)...\n", ref.URI, ref.SpecType, ref.SpecName)
		output, err := validateEngineSpec(ctx, ref, &spec, cfgPath)
		if err != nil {
			// Programming error - should not happen
			return fmt.Errorf("internal error validating engine %s: %v", ref.URI, err)
		}
		results = append(results, validationResult{
			Ref:    ref,
			Output: output,
		})
	}

	// Aggregate all results
	combined := aggregateResults(results)

	// Print validation summary
	fmt.Println()

	if combined.Valid {
		fmt.Printf("Configuration is valid\n")
		fmt.Printf("Build specs: %d\n", len(spec.Build))
		fmt.Printf("Test stages: %d\n", len(spec.Test))
		fmt.Printf("Engine configs: %d\n", len(spec.Engines))

		// Print warnings if any
		if len(combined.Warnings) > 0 {
			fmt.Printf("\nWarnings (%d):\n", len(combined.Warnings))
			for _, w := range combined.Warnings {
				if w.Field != "" {
					fmt.Printf("  - [%s] %s\n", w.Field, w.Message)
				} else {
					fmt.Printf("  - %s\n", w.Message)
				}
			}
		}

		return nil
	}

	// Validation failed - print errors
	fmt.Fprintf(os.Stderr, "Configuration is invalid\n\n")
	fmt.Fprintf(os.Stderr, "Errors (%d):\n", len(combined.Errors))
	for _, e := range combined.Errors {
		// Use the String() method which formats with full path context
		fmt.Fprintf(os.Stderr, "  - %s\n", e.String())
	}

	// Print warnings if any
	if len(combined.Warnings) > 0 {
		fmt.Fprintf(os.Stderr, "\nWarnings (%d):\n", len(combined.Warnings))
		for _, w := range combined.Warnings {
			if w.Field != "" {
				fmt.Fprintf(os.Stderr, "  - [%s] %s\n", w.Field, w.Message)
			} else {
				fmt.Fprintf(os.Stderr, "  - %s\n", w.Message)
			}
		}
	}

	return fmt.Errorf("configuration validation failed with %d error(s)", len(combined.Errors))
}

// validateConfig validates the forge.yaml configuration and returns structured output.
// This is the MCP-friendly version of runConfigValidate that returns ConfigValidateOutput
// instead of printing to console and returning an error.
func validateConfig(configPathArg string) *mcptypes.ConfigValidateOutput {
	// Determine config path (default: forge.yaml)
	cfgPath := "forge.yaml"
	if configPathArg != "" {
		cfgPath = configPathArg
	}

	// Read the spec from path
	spec, err := forge.ReadSpecFromPath(cfgPath)
	if err != nil {
		return &mcptypes.ConfigValidateOutput{
			Valid: false,
			Errors: []mcptypes.ValidationError{
				{
					Field:   "configPath",
					Message: fmt.Sprintf("failed to read config file: %v", err),
				},
			},
		}
	}

	// Validate basic structure using existing spec.Validate()
	if err := spec.Validate(); err != nil {
		return &mcptypes.ConfigValidateOutput{
			Valid: false,
			Errors: []mcptypes.ValidationError{
				{
					Field:   "",
					Message: fmt.Sprintf("basic validation failed: %v", err),
				},
			},
		}
	}

	// Extract all engine URIs from the spec
	engineRefs := extractEngineURIs(spec)

	// If no engines to validate, just return success
	if len(engineRefs) == 0 {
		return &mcptypes.ConfigValidateOutput{
			Valid:    true,
			Errors:   []mcptypes.ValidationError{},
			Warnings: []mcptypes.ValidationWarning{},
		}
	}

	// Validate each engine's spec
	ctx := context.Background()
	var results []validationResult

	for _, ref := range engineRefs {
		output, err := validateEngineSpec(ctx, ref, &spec, cfgPath)
		if err != nil {
			// Programming error - should not happen
			return &mcptypes.ConfigValidateOutput{
				Valid:      false,
				InfraError: fmt.Sprintf("internal error validating engine %s: %v", ref.URI, err),
			}
		}
		results = append(results, validationResult{
			Ref:    ref,
			Output: output,
		})
	}

	// Aggregate all results
	return aggregateResults(results)
}
