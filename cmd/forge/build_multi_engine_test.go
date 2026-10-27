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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexandremahdhaoui/forge/internal/forgepath"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

func isolateEngineResolution(t *testing.T, baseDir string) {
	t.Helper()

	t.Setenv("FORGE_RUN_LOCAL_ENABLED", "")
	t.Setenv("FORGE_RUN_LOCAL_BASEDIR", baseDir)
	t.Setenv("GOWORK", "off")

	previousSourceDir := forgepath.SourceDir
	forgepath.SourceDir = ""
	t.Cleanup(func() { forgepath.SourceDir = previousSourceDir })

	previousVersion := versionInfo.Version
	versionInfo.Version = "v0.50.10-1-gabcdef0"
	t.Cleanup(func() { versionInfo.Version = previousVersion })
}

func writeForgeCheckoutFixture(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	for path, content := range map[string]string{
		"go.mod":               "module github.com/alexandremahdhaoui/forge\n\ngo 1.26.5\n",
		"cmd/forge/main.go":    "package main\n\nfunc main() {}\n",
		"cmd/go-build/main.go": "package main\n\nfunc main() {}\n",
	} {
		full := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("creating %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", full, err)
		}
	}

	return dir
}

func TestResolveEngine_MultiEngineAlias(t *testing.T) {
	spec := &forge.Spec{
		Engines: []forge.EngineConfig{
			{
				Alias: "multi-builder",
				Type:  forge.BuilderEngineConfigType,
				Builder: []forge.BuilderEngineSpec{
					{Engine: "forge://generic-builder", Spec: forge.EngineSpec{Command: "echo", Args: []string{"step1"}}},
					{Engine: "forge://generic-builder", Spec: forge.EngineSpec{Command: "echo", Args: []string{"step2"}}},
					{Engine: "forge://generic-builder", Spec: forge.EngineSpec{Command: "echo", Args: []string{"step3"}}},
				},
			},
			{
				Alias: "single-builder",
				Type:  forge.BuilderEngineConfigType,
				Builder: []forge.BuilderEngineSpec{
					{Engine: "forge://go-build"},
				},
			},
		},
	}

	t.Run("a multi engine builder alias is refused as not resolvable to one engine", func(t *testing.T) {
		isolateEngineResolution(t, "")

		_, _, err := resolveEngine("alias://multi-builder", spec)
		if err == nil || !strings.Contains(err.Error(), "cannot be resolved") {
			t.Fatalf("expected the multi engine alias to be refused as not resolvable, got %v", err)
		}
	})

	for _, uri := range []string{"alias://single-builder", "forge://go-build"} {
		t.Run(uri+" is refused by name when no source dir is stamped and FORGE_RUN_LOCAL_BASEDIR is unset", func(t *testing.T) {
			isolateEngineResolution(t, "")

			_, _, err := resolveEngine(uri, spec)
			if err == nil {
				t.Fatal("expected a refusal naming the missing source, got a resolved engine")
			}
			if !strings.Contains(err.Error(), "this binary carries no stamped source directory and FORGE_RUN_LOCAL_BASEDIR is unset") {
				t.Fatalf("expected the refusal to name the stamped source directory and FORGE_RUN_LOCAL_BASEDIR, got %v", err)
			}
		})

		t.Run(uri+" runs the engine built from the checkout FORGE_RUN_LOCAL_BASEDIR names", func(t *testing.T) {
			checkout := writeForgeCheckoutFixture(t)
			isolateEngineResolution(t, checkout)

			command, args, err := resolveEngine(uri, spec)
			if err != nil {
				t.Fatalf("expected the engine to build from %s, got %v", checkout, err)
			}

			want := filepath.Join(checkout, "build", "local-engines", "go-build")
			if command != want || args != nil {
				t.Fatalf("expected command %s with no args, got command %s with args %v", want, command, args)
			}
			if _, err := os.Stat(want); err != nil {
				t.Fatalf("expected the built engine at %s: %v", want, err)
			}
		})
	}
}

func TestBuildLogic_MultiEngineDetection(t *testing.T) {
	spec := &forge.Spec{
		Name: "test-project",
		Engines: []forge.EngineConfig{
			{
				Alias: "generate-all",
				Type:  forge.BuilderEngineConfigType,
				Builder: []forge.BuilderEngineSpec{
					{Engine: "forge://generic-builder", Spec: forge.EngineSpec{Command: "echo", Args: []string{"step1"}}},
					{Engine: "forge://generic-builder", Spec: forge.EngineSpec{Command: "echo", Args: []string{"step2"}}},
				},
			},
		},
	}

	engineURI := "alias://generate-all"

	if strings.HasPrefix(engineURI, "alias://") {
		aliasName := strings.TrimPrefix(engineURI, "alias://")
		engineConfig := getEngineConfig(aliasName, spec)

		if engineConfig == nil {
			t.Fatal("Engine alias not found")
		}

		if engineConfig.Type == forge.BuilderEngineConfigType && len(engineConfig.Builder) > 1 {
			t.Logf("✅ Multi-engine builder detected (%d engines)", len(engineConfig.Builder))
			t.Log("✅ Would route to orchestrator (NOT call resolveEngine)")
			return
		}

		_, _, err := resolveEngine(engineURI, spec)
		if err != nil {
			t.Fatalf("Single-engine alias resolution failed: %v", err)
		}
	}
}
