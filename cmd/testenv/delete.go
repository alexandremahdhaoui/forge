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
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

func (c *testenvCommands) cmdDelete(testID string) error {
	if testID == "" {
		return fmt.Errorf("test ID is required")
	}

	config, err := forge.ReadSpec()
	if err != nil {
		return fmt.Errorf("failed to read forge.yaml: %w", err)
	}

	artifactStorePath, err := forge.GetArtifactStorePath(config.ArtifactStorePath)
	if err != nil {
		return fmt.Errorf("failed to get artifact store path: %w", err)
	}

	store, err := forge.ReadArtifactStore(artifactStorePath)
	if err != nil {
		return fmt.Errorf("failed to read artifact store: %w", err)
	}

	env, err := forge.GetTestEnvironment(&store, testID)
	if err != nil {
		return fmt.Errorf("test environment not found: %s", testID)
	}

	var testSpec *forge.TestSpec
	for i := range config.Test {
		if config.Test[i].Name == env.Name {
			testSpec = &config.Test[i]
			break
		}
	}

	var cleanupErr error
	if testSpec != nil && testSpec.Testenv != "" {
		if strings.HasPrefix(testSpec.Testenv, "forge://") {
			fmt.Fprintf(os.Stderr, "Tearing down %s...\n", testSpec.Testenv)

			params := map[string]any{}
			if testSpec.Testenv == "forge://test-report" {
				params["reportID"] = testID
			} else {
				params["testID"] = testID
				params["metadata"] = env.Metadata
			}

			if _, err := c.callEngine(testSpec.Testenv, "delete", params); err != nil {
				cleanupErr = fmt.Errorf("failed to delete with %s: %w", testSpec.Testenv, err)
			} else {
				fmt.Fprintf(os.Stderr, "  ✓ %s teardown complete\n", testSpec.Testenv)
			}
		} else if err := c.deleteRecordedSubengines(env); err != nil {
			cleanupErr = fmt.Errorf("failed to orchestrate cleanup: %w", err)
		}
	}

	if cleanupErr != nil {
		if err := recordEnvironment(config, env); err != nil {
			return errors.Join(cleanupErr, err)
		}

		return cleanupErr
	}

	for _, resource := range env.ManagedResources {
		if err := os.RemoveAll(resource); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove resource %s: %v\n", resource, err)
		}
	}

	if err := forge.AtomicDeleteTestEnvironment(artifactStorePath, testID); err != nil {
		return fmt.Errorf("failed to delete test environment: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Deleted test environment: %s\n", testID)
	return nil
}

func (c *testenvCommands) deleteRecordedSubengines(env *forge.TestEnvironment) error {
	var cleanupErrors []error
	remaining := make([]string, 0, len(env.Subengines))

	for i := len(env.Subengines) - 1; i >= 0; i-- {
		engineURI := env.Subengines[i]
		fmt.Fprintf(os.Stderr, "Tearing down %s...\n", engineURI)

		params := map[string]any{
			"testID":   env.ID,
			"metadata": env.Metadata,
		}

		if _, err := c.callEngine(engineURI, "delete", params); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("failed to delete with %s: %w", engineURI, err))
			remaining = append(remaining, engineURI)
			continue
		}

		fmt.Fprintf(os.Stderr, "  ✓ %s teardown complete\n", engineURI)
	}

	env.Subengines = reverse(remaining)

	if len(cleanupErrors) > 0 {
		return fmt.Errorf("cleanup errors (resources may be leaked): %v", cleanupErrors)
	}

	return nil
}

func reverse(engineURIs []string) []string {
	out := make([]string, 0, len(engineURIs))
	for i := len(engineURIs) - 1; i >= 0; i-- {
		out = append(out, engineURIs[i])
	}

	return out
}
