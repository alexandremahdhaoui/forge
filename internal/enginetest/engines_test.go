//go:build integration

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

package enginetest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandremahdhaoui/forge/internal/enginetest"
)

// getRepoRoot returns the repository root directory.
func getRepoRoot(t *testing.T) string {
	t.Helper()

	// Try to find the repo root by looking for go.mod
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find repository root (no go.mod found)")
		}
		dir = parent
	}
}

func TestAllEnginesHaveVersionSupport(t *testing.T) {
	repoRoot := getRepoRoot(t)
	engines := enginetest.AllEngines(repoRoot)

	for _, engine := range engines {
		t.Run(engine.Name, func(t *testing.T) {
			enginetest.TestBinaryExists(t, engine)
			enginetest.TestVersionCommand(t, engine)
		})
	}
}

func TestAllMCPEnginesHaveMCPSupport(t *testing.T) {
	repoRoot := getRepoRoot(t)
	engines := enginetest.AllEngines(repoRoot)

	for _, engine := range engines {
		if !engine.SupportsMCP {
			continue
		}

		t.Run(engine.Name, func(t *testing.T) {
			enginetest.TestMCPMode(t, engine)
		})
	}
}

func TestEnginesList(t *testing.T) {
	repoRoot := getRepoRoot(t)
	engines := enginetest.AllEngines(repoRoot)

	if len(engines) != 19 {
		t.Errorf("Expected 19 engines, got %d", len(engines))
	}

	expectedEngines := map[string]bool{
		"forge":                true,
		"go-build":             true,
		"container-build":      true,
		"generic-builder":      true,
		"testenv":              true,
		"testenv-kind":         true,
		"testenv-lcr":          true,
		"testenv-helm-install": true,
		"go-test":              true,
		"go-lint-licenses":     true,
		"go-lint-tags":         true,
		"generic-test-runner":  true,
		"test-report":          true,
		"go-format":            true,
		"go-lint":              true,
		"go-gen-mocks":         true,
		"go-gen-openapi":       true,
		"ci-orchestrator":      true,
		"forge-e2e":            true,
	}

	for _, engine := range engines {
		if !expectedEngines[engine.Name] {
			t.Errorf("Unexpected engine in list: %s", engine.Name)
		}
		delete(expectedEngines, engine.Name)
	}

	if len(expectedEngines) > 0 {
		for name := range expectedEngines {
			t.Errorf("Missing engine from list: %s", name)
		}
	}
}

func TestMCPEnginesConfiguration(t *testing.T) {
	repoRoot := getRepoRoot(t)
	engines := enginetest.AllEngines(repoRoot)

	// Verify which engines should support MCP
	expectedMCPEngines := map[string]bool{
		"forge":                true,
		"go-build":             true,
		"container-build":      true,
		"generic-builder":      true,
		"testenv":              true,
		"testenv-kind":         true,
		"testenv-lcr":          true,
		"testenv-helm-install": true,
		"go-test":              true,
		"go-lint-licenses":     true,
		"go-lint-tags":         true,
		"generic-test-runner":  true,
		"test-report":          true,
		"go-format":            true,
		"go-lint":              true,
		"go-gen-mocks":         true,
		"go-gen-openapi":       true,
		"ci-orchestrator":      true,
		"forge-e2e":            true,
	}

	for _, engine := range engines {
		expected := expectedMCPEngines[engine.Name]
		if engine.SupportsMCP != expected {
			t.Errorf("Engine %s: expected SupportsMCP=%v, got %v",
				engine.Name, expected, engine.SupportsMCP)
		}
	}
}

// TestAllBuildEnginesImplementBuildBatch verifies that all build engines
// implement both "build" and "buildBatch" MCP tools. This test ensures that
// the issue where go-gen-openapi was missing buildBatch never happens again.
func TestAllBuildEnginesImplementBuildBatch(t *testing.T) {
	repoRoot := getRepoRoot(t)
	engines := enginetest.AllEngines(repoRoot)

	// Build engines are those that should implement build and buildBatch tools
	buildEngines := map[string]bool{
		"go-build":        true,
		"container-build": true,
		"go-gen-openapi":  true,
		"go-gen-mocks":    true,
		"generic-builder": true,
	}

	for _, engine := range engines {
		if !buildEngines[engine.Name] {
			continue
		}

		t.Run(engine.Name, func(t *testing.T) {
			enginetest.TestBuildEngineTools(t, engine)
		})
	}
}
