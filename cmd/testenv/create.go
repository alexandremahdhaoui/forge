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
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/portalloc"
	"github.com/alexandremahdhaoui/forge/pkg/templateutil"
	"github.com/alexandremahdhaoui/forge/pkg/testenvutil"
)

func (c *testenvCommands) cmdCreate(stageName string) (string, error) {
	if stageName == "" {
		return "", fmt.Errorf("stage name is required")
	}

	config, err := forge.ReadSpec()
	if err != nil {
		return "", fmt.Errorf("failed to read forge.yaml: %w", err)
	}

	var testSpec *forge.TestSpec
	for i := range config.Test {
		if config.Test[i].Name == stageName {
			testSpec = &config.Test[i]
			break
		}
	}

	if testSpec == nil {
		return "", fmt.Errorf("test stage not found in forge.yaml: %s", stageName)
	}

	testID := generateTestID(stageName)

	rootDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	tmpBase := filepath.Join(rootDir, ".forge", "tmp")
	if err := os.MkdirAll(tmpBase, 0o755); err != nil {
		return "", fmt.Errorf("failed to create tmp base directory: %w", err)
	}

	tmpDir := filepath.Join(tmpBase, testID)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create tmpDir: %w", err)
	}

	env := &forge.TestEnvironment{
		ID:               testID,
		Name:             stageName,
		Status:           forge.TestStatusCreated,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		TmpDir:           tmpDir,
		Files:            make(map[string]string),
		ManagedResources: []string{tmpDir},
		Metadata:         make(map[string]string),
	}

	if err := recordEnvironment(config, env); err != nil {
		return "", err
	}

	setupErr := c.runTestenvSetup(config, testSpec, env)
	if setupErr != nil {
		env.Status = forge.TestStatusFailed
	}

	if err := recordEnvironment(config, env); err != nil {
		return "", err
	}

	if setupErr != nil {
		return "", setupErr
	}

	fmt.Fprintln(os.Stderr, testID)

	return testID, nil
}

func recordEnvironment(config forge.Spec, env *forge.TestEnvironment) error {
	artifactStorePath, err := forge.GetArtifactStorePath(config.ArtifactStorePath)
	if err != nil {
		return fmt.Errorf("failed to get artifact store path: %w", err)
	}

	store, err := forge.ReadOrCreateArtifactStore(artifactStorePath)
	if err != nil {
		return fmt.Errorf("failed to read artifact store: %w", err)
	}

	env.UpdatedAt = time.Now().UTC()
	forge.AddOrUpdateTestEnvironment(&store, env)

	if err := forge.WriteArtifactStore(artifactStorePath, store); err != nil {
		return fmt.Errorf("failed to write artifact store: %w", err)
	}

	return nil
}

func (c *testenvCommands) runTestenvSetup(config forge.Spec, testSpec *forge.TestSpec, env *forge.TestEnvironment) error {
	setupSpec := testSpec.Testenv

	switch {
	case setupSpec == "":
		fmt.Fprintf(os.Stderr, "No testenv configured for stage %s\n", env.Name)
		return nil
	case strings.HasPrefix(setupSpec, "forge://"):
		if err := c.createWithDirectEngine(setupSpec, env); err != nil {
			return err
		}
		return nil
	default:
		if err := c.orchestrateCreate(config, strings.TrimPrefix(setupSpec, "alias://"), env); err != nil {
			return fmt.Errorf("failed to orchestrate testenv-subengines: %w", err)
		}
		return nil
	}
}

func (c *testenvCommands) createWithDirectEngine(setupSpec string, env *forge.TestEnvironment) error {
	fmt.Fprintf(os.Stderr, "Setting up %s...\n", setupSpec)

	params := map[string]any{
		"stage": env.Name,
	}

	if setupSpec != "forge://test-report" {
		params["testID"] = env.ID
		params["tmpDir"] = env.TmpDir
	}

	result, err := c.callEngine(setupSpec, "create", params)
	if err != nil {
		return fmt.Errorf("failed to create with %s: %w", setupSpec, err)
	}

	mergeSubengineResult(setupSpec, result, env, nil)
	fmt.Fprintf(os.Stderr, "  ✓ %s setup complete\n", setupSpec)

	return nil
}

func generateTestID(stageName string) string {
	randBytes := make([]byte, 4)
	_, _ = rand.Read(randBytes)
	suffix := hex.EncodeToString(randBytes)

	dateStr := time.Now().Format("20060102")
	return fmt.Sprintf("test-%s-%s-%s", stageName, dateStr, suffix)
}

func mergeSubengineResult(engineURI string, result interface{}, env *forge.TestEnvironment, accumulatedMetadata map[string]string) {
	env.Subengines = append(env.Subengines, engineURI)

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return
	}

	if files, ok := resultMap["files"].(map[string]interface{}); ok {
		for key, value := range files {
			if strValue, ok := value.(string); ok {
				env.Files[key] = strValue
			}
		}
	}

	if metadata, ok := resultMap["metadata"].(map[string]interface{}); ok {
		for key, value := range metadata {
			if strValue, ok := value.(string); ok {
				env.Metadata[key] = strValue
				if accumulatedMetadata != nil {
					accumulatedMetadata[key] = strValue
				}
			}
		}
	}

	if resources, ok := resultMap["managedResources"].([]interface{}); ok {
		for _, resource := range resources {
			if strResource, ok := resource.(string); ok {
				env.ManagedResources = append(env.ManagedResources, strResource)
			}
		}
	}
}

func (c *testenvCommands) orchestrateCreate(config forge.Spec, setupAlias string, env *forge.TestEnvironment) error {
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

	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	stateFilePath, err := portAllocStateFilePath()
	if err != nil {
		return fmt.Errorf("failed to resolve port allocation state path: %w", err)
	}
	allocator := portalloc.New(stateFilePath)

	accumulatedMetadata := make(map[string]string)
	envTracker := testenvutil.NewEnvSourceTracker()

	unwind := func(cause error) error {
		return errors.Join(cause, c.deleteRecordedSubengines(env))
	}

	for subengineIndex, subengine := range subengines {
		fmt.Fprintf(os.Stderr, "Setting up %s...\n", subengine.Engine)

		var specToUse map[string]interface{}
		if subengine.DeferTemplates {
			specToUse = subengine.Spec
		} else {
			specToUse = subengine.Spec
			var portEnvVars map[string]string
			if len(subengine.Spec) > 0 {
				portEnvVars = make(map[string]string)
				accumulatedEnv := envTracker.ToMap()

				if err := allocator.Open(); err != nil {
					return unwind(fmt.Errorf("failed to open port allocator: %w", err))
				}

				wrappedAllocate := func(args ...any) (string, error) {
					if len(args) < 2 {
						return "", fmt.Errorf("allocateOpenPort requires at least 2 args (addr, id), got %d", len(args))
					}
					addr, ok := args[0].(string)
					if !ok {
						return "", fmt.Errorf("allocateOpenPort: addr must be a string")
					}
					id, ok := args[1].(string)
					if !ok {
						return "", fmt.Errorf("allocateOpenPort: id must be a string")
					}

					var port string
					var err error
					if len(args) == 4 {
						minPort, err2 := toInt(args[2])
						if err2 != nil {
							return "", fmt.Errorf("allocateOpenPort: minPort: %w", err2)
						}
						maxPort, err2 := toInt(args[3])
						if err2 != nil {
							return "", fmt.Errorf("allocateOpenPort: maxPort: %w", err2)
						}
						port, err = allocator.AllocateInRange(addr, id, minPort, maxPort)
					} else {
						port, err = allocator.Allocate(addr, id)
					}
					if err != nil {
						return "", err
					}
					envKey := NormalizePortAllocEnvKey(id)
					portEnvVars[envKey] = port
					return port, nil
				}
				funcMap := template.FuncMap{
					"allocateOpenPort": wrappedAllocate,
				}

				var err error
				specToUse, err = templateutil.ExpandTemplates(subengine.Spec, accumulatedEnv, templateutil.WithFuncMap(funcMap))

				if closeErr := allocator.Close(); closeErr != nil {
					if err != nil {
						return unwind(fmt.Errorf("failed to expand templates for %s: %w (also failed to close port allocator: %v)", subengine.Engine, err, closeErr))
					}
					return unwind(fmt.Errorf("failed to close port allocator: %w", closeErr))
				}

				if err != nil {
					return unwind(fmt.Errorf("failed to expand templates for %s: %w", subengine.Engine, err))
				}
			}
			if len(portEnvVars) > 0 {
				envTracker.Merge(portEnvVars, nil, subengineIndex)
			}
		}

		var envPropagation *forge.EnvPropagation
		if envPropSpec, exists := subengine.Spec["envPropagation"]; exists {
			var err error
			envPropagation, err = extractEnvPropagation(envPropSpec)
			if err != nil {
				return unwind(fmt.Errorf("failed to parse envPropagation for %s: %w", subengine.Engine, err))
			}

			if err := envPropagation.Validate(); err != nil {
				return unwind(fmt.Errorf("invalid envPropagation for %s: %w", subengine.Engine, err))
			}
		}

		params := map[string]any{
			"testID":   env.ID,
			"stage":    env.Name,
			"tmpDir":   env.TmpDir,
			"rootDir":  rootDir,
			"metadata": accumulatedMetadata,
			"env":      envTracker.ToMap(),
		}

		if len(specToUse) > 0 {
			params["spec"] = specToUse
		}

		if envPropagation != nil {
			params["envPropagation"] = envPropagation
		}

		result, err := c.callEngine(subengine.Engine, "create", params)
		if err != nil {
			return unwind(fmt.Errorf("failed to create with %s: %w", subengine.Engine, err))
		}

		mergeSubengineResult(subengine.Engine, result, env, accumulatedMetadata)

		if resultMap, ok := result.(map[string]interface{}); ok {
			if envMap, ok := resultMap["env"].(map[string]interface{}); ok {
				newEnv := make(map[string]string)
				for key, value := range envMap {
					if strValue, ok := value.(string); ok {
						newEnv[key] = strValue
					}
				}
				envTracker.Merge(newEnv, envPropagation, subengineIndex)
			}
		}

		env.Env = envTracker.ToMap()

		fmt.Fprintf(os.Stderr, "  ✓ %s setup complete\n", subengine.Engine)
	}

	return nil
}

func portAllocStateFilePath() (string, error) {
	stateDir := os.Getenv("XDG_STATE_HOME")
	if stateDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		stateDir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(stateDir, "forge", "port-allocations.json"), nil
}

func extractEnvPropagation(envPropSpec interface{}) (*forge.EnvPropagation, error) {
	jsonData, err := json.Marshal(envPropSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal envPropagation: %w", err)
	}

	var envProp forge.EnvPropagation
	if err := json.Unmarshal(jsonData, &envProp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal envPropagation: %w", err)
	}

	return &envProp, nil
}
