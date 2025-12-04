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
	"fmt"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
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
	configPath := "forge.yaml"
	if len(args) > 0 {
		configPath = args[0]
	}

	// Read and validate the spec
	spec, err := forge.ReadSpecFromPath(configPath)
	if err != nil {
		return fmt.Errorf("validation failed:\n%v", err)
	}

	// If we got here, validation passed
	fmt.Printf("✅ Configuration is valid: %s\n", configPath)
	fmt.Printf("Project: %s\n", spec.Name)
	fmt.Printf("Build specs: %d\n", len(spec.Build))
	fmt.Printf("Test stages: %d\n", len(spec.Test))
	fmt.Printf("Engine configs: %d\n", len(spec.Engines))

	return nil
}
