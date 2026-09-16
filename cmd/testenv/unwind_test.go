//go:build unit

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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

type engineCall struct {
	engine string
	tool   string
}

type fakeEngineRegistry struct {
	calls    []engineCall
	failures map[engineCall]error
}

func (f *fakeEngineRegistry) call(engineURI string, toolName string, params map[string]any) (interface{}, error) {
	call := engineCall{engine: engineURI, tool: toolName}
	f.calls = append(f.calls, call)

	if err, failed := f.failures[call]; failed {
		return nil, err
	}

	return map[string]interface{}{}, nil
}

func installFakeEngineRegistry(t *testing.T, failures map[engineCall]error) (*fakeEngineRegistry, *testenvCommands) {
	t.Helper()

	registry := &fakeEngineRegistry{failures: failures}

	return registry, &testenvCommands{callEngine: registry.call}
}

func testenvSpecWithSubengines(artifactStorePath string, engines ...string) forge.Spec {
	subengines := make([]forge.TestenvEngineSpec, 0, len(engines))
	for _, engine := range engines {
		subengines = append(subengines, forge.TestenvEngineSpec{Engine: engine})
	}

	return forge.Spec{
		Name:              "test-project",
		ArtifactStorePath: artifactStorePath,
		Engines: []forge.EngineConfig{{
			Alias:   "setup",
			Type:    "testenv",
			Testenv: subengines,
		}},
		Test: []forge.TestSpec{{
			Name:    "integration",
			Runner:  "forge://go-test",
			Testenv: "alias://setup",
		}},
	}
}

func newTestEnvironment(tmpDir string) *forge.TestEnvironment {
	return &forge.TestEnvironment{
		ID:               "test-integration-20260916-abcdef01",
		Name:             "integration",
		Status:           forge.TestStatusCreated,
		TmpDir:           tmpDir,
		Files:            make(map[string]string),
		ManagedResources: []string{tmpDir},
		Metadata:         make(map[string]string),
	}
}

func TestAFailingSubengineUnwindsOnlyTheSubenginesThatAlreadySucceeded(t *testing.T) {
	tests := []struct {
		name          string
		subengines    []string
		failingEngine string
		expectedCalls []engineCall
	}{
		{
			name:          "the second of three subengines fails",
			subengines:    []string{"forge://one", "forge://two", "forge://three"},
			failingEngine: "forge://two",
			expectedCalls: []engineCall{
				{engine: "forge://one", tool: "create"},
				{engine: "forge://two", tool: "create"},
				{engine: "forge://one", tool: "delete"},
			},
		},
		{
			name:          "the first subengine fails so nothing is unwound",
			subengines:    []string{"forge://one", "forge://two"},
			failingEngine: "forge://one",
			expectedCalls: []engineCall{
				{engine: "forge://one", tool: "create"},
			},
		},
		{
			name:          "the last of three subengines fails and the first two unwind in reverse",
			subengines:    []string{"forge://one", "forge://two", "forge://three"},
			failingEngine: "forge://three",
			expectedCalls: []engineCall{
				{engine: "forge://one", tool: "create"},
				{engine: "forge://two", tool: "create"},
				{engine: "forge://three", tool: "create"},
				{engine: "forge://two", tool: "delete"},
				{engine: "forge://one", tool: "delete"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry, commands := installFakeEngineRegistry(t, map[engineCall]error{
				{engine: tt.failingEngine, tool: "create"}: errors.New("engine refused"),
			})

			config := testenvSpecWithSubengines("", tt.subengines...)
			err := commands.orchestrateCreate(config, "setup", newTestEnvironment(t.TempDir()))
			if err == nil {
				t.Fatal("expected orchestrateCreate to return an error")
			}

			assertCallsEqual(t, registry.calls, tt.expectedCalls)
		})
	}
}

func TestAFailedUnwindReportsEverySubengineByNameAndKeepsTheRecord(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	artifactStorePath := filepath.Join(workDir, "artifact-store.yaml")
	writeForgeSpec(t, workDir, artifactStorePath, "alias://setup")

	registry, commands := installFakeEngineRegistry(t, map[engineCall]error{
		{engine: "forge://two", tool: "create"}: errors.New("two refused to create"),
		{engine: "forge://one", tool: "delete"}: errors.New("one refused to delete"),
	})

	_, err := commands.cmdCreate("integration")
	if err == nil {
		t.Fatal("expected cmdCreate to return an error")
	}

	for _, name := range []string{"forge://one", "forge://two"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("expected error to name %s, got %q", name, err.Error())
		}
	}

	assertCallsEqual(t, registry.calls, []engineCall{
		{engine: "forge://one", tool: "create"},
		{engine: "forge://two", tool: "create"},
		{engine: "forge://one", tool: "delete"},
	})

	env := singleRecordedEnvironment(t, artifactStorePath)
	if env.Name != "integration" {
		t.Errorf("expected recorded stage integration, got %s", env.Name)
	}
}

func TestAFailedCreateLeavesTheEnvironmentAndItsTmpDirForALaterDelete(t *testing.T) {
	tests := []struct {
		name     string
		testenv  string
		failures map[engineCall]error
	}{
		{
			name:    "an alias chain whose second subengine fails",
			testenv: "alias://setup",
			failures: map[engineCall]error{
				{engine: "forge://two", tool: "create"}: errors.New("two refused to create"),
			},
		},
		{
			name:    "a directly named engine that fails",
			testenv: "forge://one",
			failures: map[engineCall]error{
				{engine: "forge://one", tool: "create"}: errors.New("one refused to create"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			t.Chdir(workDir)

			artifactStorePath := filepath.Join(workDir, "artifact-store.yaml")
			writeForgeSpec(t, workDir, artifactStorePath, tt.testenv)

			_, commands := installFakeEngineRegistry(t, tt.failures)

			if _, err := commands.cmdCreate("integration"); err == nil {
				t.Fatal("expected cmdCreate to return an error")
			}

			env := singleRecordedEnvironment(t, artifactStorePath)
			if info, err := os.Stat(env.TmpDir); err != nil || !info.IsDir() {
				t.Errorf("expected tmpDir %s to survive a failed create, got %v", env.TmpDir, err)
			}

			if env.Status != forge.TestStatusFailed {
				t.Errorf("expected the recorded status to be %s, got %s", forge.TestStatusFailed, env.Status)
			}

			if err := commands.cmdDelete(env.ID); err != nil {
				t.Fatalf("expected delete-env to reach the recorded environment, got %v", err)
			}
		})
	}
}

func TestTheEnvironmentIsInTheArtifactStoreBeforeTheFirstSubengineRuns(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	artifactStorePath := filepath.Join(workDir, "artifact-store.yaml")
	writeForgeSpec(t, workDir, artifactStorePath, "alias://setup")

	recordedBeforeFirstSubengine := false
	commands := &testenvCommands{
		callEngine: func(engineURI string, toolName string, params map[string]any) (interface{}, error) {
			if engineURI == "forge://one" && toolName == "create" {
				store, err := forge.ReadArtifactStore(artifactStorePath)
				recordedBeforeFirstSubengine = err == nil && len(store.TestEnvironments) == 1
				for _, recorded := range store.TestEnvironments {
					recordedBeforeFirstSubengine = recordedBeforeFirstSubengine &&
						recorded.Status == forge.TestStatusCreated
				}
			}
			return nil, errors.New("one refused to create")
		},
	}

	if _, err := commands.cmdCreate("integration"); err == nil {
		t.Fatal("expected cmdCreate to return an error")
	}

	if !recordedBeforeFirstSubengine {
		t.Error("expected the environment to be recorded before the first subengine ran")
	}
}

func writeForgeSpec(t *testing.T, dir string, artifactStorePath string, testenv string) {
	t.Helper()

	forgeYAML := `name: test-project
artifactStorePath: ` + artifactStorePath + `
engines:
  - alias: setup
    type: testenv
    testenv:
      - engine: forge://one
      - engine: forge://two
test:
  - name: integration
    runner: "forge://go-test"
    testenv: "` + testenv + `"
`

	if err := os.WriteFile(filepath.Join(dir, "forge.yaml"), []byte(forgeYAML), 0o644); err != nil {
		t.Fatalf("writing forge.yaml: %v", err)
	}
}

func singleRecordedEnvironment(t *testing.T, artifactStorePath string) *forge.TestEnvironment {
	t.Helper()

	store, err := forge.ReadArtifactStore(artifactStorePath)
	if err != nil {
		t.Fatalf("reading artifact store: %v", err)
	}

	if len(store.TestEnvironments) != 1 {
		t.Fatalf("expected 1 recorded test environment, got %d", len(store.TestEnvironments))
	}

	for _, env := range store.TestEnvironments {
		return env
	}

	return nil
}

func assertCallsEqual(t *testing.T, got []engineCall, want []engineCall) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected calls %v, got %v", want, got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected calls %v, got %v", want, got)
		}
	}
}
