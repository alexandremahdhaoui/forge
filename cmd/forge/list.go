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
)

// runList lists available build targets and test stages from forge.yaml.
// It accepts an optional category argument to filter the output:
//   - "build": List only build targets
//   - "test": List only test stages
//   - no argument: List both build targets and test stages
func runList(args []string) error {
	// Parse category argument
	category := ""
	if len(args) > 0 {
		category = args[0]
		if category != "build" && category != "test" {
			return fmt.Errorf("unknown category: %s (valid: build, test)", category)
		}
	}

	// Load config
	config, err := loadConfig()
	if err != nil {
		return err
	}

	// Output based on category
	if category == "" || category == "build" {
		fmt.Println("BUILD TARGETS:")
		if len(config.Build) == 0 {
			fmt.Println("  (none)")
		} else {
			for _, b := range config.Build {
				fmt.Printf("  - %s\n", b.Name)
			}
		}
		if category == "" {
			fmt.Println()
		}
	}

	if category == "" || category == "test" {
		fmt.Println("TEST STAGES:")
		if len(config.Test) == 0 {
			fmt.Println("  (none)")
		} else {
			for _, t := range config.Test {
				fmt.Printf("  - %s\n", t.Name)
			}
		}
	}

	return nil
}
