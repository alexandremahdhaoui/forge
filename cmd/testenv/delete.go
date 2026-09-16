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
		} else {
			setupAlias := strings.TrimPrefix(testSpec.Testenv, "alias://")

			if err := c.orchestrateDelete(config, setupAlias, env); err != nil {
				cleanupErr = fmt.Errorf("failed to orchestrate cleanup: %w", err)
			}
		}
	}

	if cleanupErr != nil {
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

func (c *testenvCommands) orchestrateDelete(config forge.Spec, setupAlias string, env *forge.TestEnvironment) error {
	var engineConfig *forge.EngineConfig
	for i := range config.Engines {
		if config.Engines[i].Alias == setupAlias {
			engineConfig = &config.Engines[i]
			break
		}
	}

	if engineConfig == nil {
		return fmt.Errorf("engine alias not found: %s", setupAlias)
	}

	if engineConfig.Type != "testenv" {
		return fmt.Errorf("engine %s is not a testenv type (got: %s)", setupAlias, engineConfig.Type)
	}

	subengines := engineConfig.Testenv
	if len(subengines) == 0 {
		return fmt.Errorf("no testenv-subengines configured for %s", setupAlias)
	}

	return c.deleteSubenginesInReverse(subengines, env)
}

func (c *testenvCommands) deleteSubenginesInReverse(subengines []forge.TestenvEngineSpec, env *forge.TestEnvironment) error {
	var cleanupErrors []error
	for i := len(subengines) - 1; i >= 0; i-- {
		subengine := subengines[i]
		fmt.Fprintf(os.Stderr, "Tearing down %s...\n", subengine.Engine)

		params := map[string]any{
			"testID":   env.ID,
			"metadata": env.Metadata,
		}

		if _, err := c.callEngine(subengine.Engine, "delete", params); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("failed to delete with %s: %w", subengine.Engine, err))
			continue
		}

		fmt.Fprintf(os.Stderr, "  ✓ %s teardown complete\n", subengine.Engine)
	}

	if len(cleanupErrors) > 0 {
		return fmt.Errorf("cleanup errors (resources may be leaked): %v", cleanupErrors)
	}

	return nil
}
