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

// runTestAll executes the complete test-all workflow:
// 1. Builds all artifacts defined in forge.yaml
// 2. Runs all test stages sequentially in order
// 3. Auto-deletes test environments after each stage
// 4. Stops execution immediately if a stage fails (fail-fast)
// 5. Returns error immediately on first failure
//
// Usage: forge test-all
func runTestAll(args []string) error {
	// Step 1: Build all artifacts
	fmt.Println("🔨 Building all artifacts...")
	if err := runBuild([]string{}, false); err != nil {
		fmt.Printf("❌ Build failed: %v\n", err)
		return fmt.Errorf("build failed: %w", err)
	}
	fmt.Println("✅ Build completed successfully")

	// Step 2: Load configuration and discover test stages
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load forge.yaml: %w", err)
	}

	// Check if there are any test stages
	if len(config.Test) == 0 {
		fmt.Println("\n⚠️  No test stages defined in forge.yaml")
		return nil
	}

	// Print test stage summary
	fmt.Printf("\n🧪 Running %d test stage(s)...\n", len(config.Test))

	// Step 3: Execute test stages with fail-fast
	for i := range config.Test {
		testSpec := &config.Test[i]
		fmt.Printf("\n--- Running test stage: %s ---\n", testSpec.Name)

		// Execute the test stage - testRun returns the testID of the created environment
		testID, err := testRun(&config, testSpec, []string{})

		// Print stage result
		if err == nil {
			fmt.Printf("✅ Stage '%s' passed\n", testSpec.Name)
		} else {
			fmt.Printf("❌ Stage '%s' failed: %v\n", testSpec.Name, err)

			// Auto-delete test environment before returning (best-effort)
			// Use the specific testID returned by testRun to avoid mismatch
			if testID != "" {
				if cleanupErr := cleanupTestEnvironmentByID(testSpec, testID); cleanupErr != nil {
					fmt.Printf("⚠️  Warning: Failed to cleanup test environment for stage '%s': %v\n", testSpec.Name, cleanupErr)
				} else {
					fmt.Printf("🧹 Cleaned up test environment for stage '%s'\n", testSpec.Name)
				}
			}

			// Fail fast - return immediately
			return fmt.Errorf("test stage '%s' failed: %w", testSpec.Name, err)
		}

		// Auto-delete test environment if one was created (success case)
		// Use the specific testID returned by testRun to avoid mismatch
		if testID != "" {
			if cleanupErr := cleanupTestEnvironmentByID(testSpec, testID); cleanupErr != nil {
				// Log but don't fail - cleanup is best effort
				fmt.Printf("⚠️  Warning: Failed to cleanup test environment for stage '%s': %v\n", testSpec.Name, cleanupErr)
			} else {
				fmt.Printf("🧹 Cleaned up test environment for stage '%s'\n", testSpec.Name)
			}
		}
	}

	// All stages passed
	fmt.Println("\n✅ All test stages passed!")
	return nil
}

// cleanupTestEnvironmentByID deletes a specific test environment by its ID.
// This ensures we delete exactly the environment that was created by testRun,
// avoiding mismatch when multiple environments exist for the same stage.
func cleanupTestEnvironmentByID(testSpec *forge.TestSpec, testID string) error {
	if testID == "" {
		return fmt.Errorf("testID is required for cleanup")
	}

	// Delete it using testDeleteEnv with the specific testID
	if err := testDeleteEnv(testSpec, []string{testID}); err != nil {
		return fmt.Errorf("failed to delete environment %s: %w", testID, err)
	}

	return nil
}
