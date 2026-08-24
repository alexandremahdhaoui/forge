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
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

const minimalSpec = `openapi: 3.0.3
info:
  title: fixture Spec Schema
  version: 0.1.0
components:
  schemas:
    Spec:
      type: object
      properties:
        greeting:
          type: string
          description: The greeting.
      required:
        - greeting
`

func writeKindFixture(t *testing.T, dir, forgeDevYaml string) {
	t.Helper()

	createRequiredDocs(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "forge-dev.yaml"), []byte(forgeDevYaml), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "spec.openapi.yaml"), []byte(minimalSpec), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTheBinaryKindEmitsOnlyTheManifestAndDocs(t *testing.T) {
	dir := t.TempDir()
	writeKindFixture(t, dir, `name: plain-tool
kind: binary
version: 0.1.0
runtime:
  env:
    - PLAIN_TOKEN
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
  docsBaseURL: https://example.com/raw
`)

	_, err := generate(context.Background(), mcptypes.BuildInput{Name: "plain-tool", Src: dir, Engine: "forge://forge-dev"})
	if err != nil {
		t.Fatalf("generate() for the binary kind: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, GeneratedRunnableFile))
	if err != nil {
		t.Fatalf("the runnable manifest must exist: %v", err)
	}

	for _, want := range []string{"- PLAIN_TOKEN", "- greeting"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("manifest missing %q:\n%s", want, raw)
		}
	}

	for _, doc := range []string{"docs/schema.md", "docs/list.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, doc)); err != nil {
			t.Errorf("the binary kind must emit %s: %v", doc, err)
		}
	}

	for _, code := range []string{GeneratedSpecFile, GeneratedMCPFile, GeneratedMainFile, GeneratedCLIFile} {
		if _, err := os.Stat(filepath.Join(dir, code)); err == nil {
			t.Errorf("the binary kind must not emit %s", code)
		}
	}
}

func cliFixtureYaml() string {
	return `name: fixture-cli
kind: cli
version: 0.1.0
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
  docsBaseURL: https://example.com/raw
surface:
  commands:
    - name: greet
      description: Print the greeting.
    - name: fail
      description: Exit with the given code.
`
}

func TestTheCLIKindGeneratesADispatcherThatRuns(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go toolchain on this host")
	}

	dir := t.TempDir()
	writeKindFixture(t, dir, cliFixtureYaml())

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/fixture-cli\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := generate(context.Background(), mcptypes.BuildInput{Name: "fixture-cli", Src: dir, Engine: "forge://forge-dev"})
	if err != nil {
		t.Fatalf("generate() for the cli kind: %v", err)
	}

	handlers := `package main

import (
	"fmt"
	"strconv"
)

func NewCLIHandlers() CLIHandlers {
	return CLIHandlers{
		Greet: func(args []string) int {
			fmt.Println("hello", args)

			return 0
		},
		Fail: func(args []string) int {
			code, _ := strconv.Atoi(args[0])

			return code
		},
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "handlers.go"), []byte(handlers), 0o644); err != nil {
		t.Fatal(err)
	}

	binary := filepath.Join(dir, "fixture-cli-bin")
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Dir = dir
	build.Env = append(os.Environ(), "GOWORK=off")

	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building the fixture cli: %v: %s", err, out)
	}

	run := func(args ...string) (string, int) {
		cmd := exec.Command(binary, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()

		code := 0
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		}

		return string(out), code
	}

	out, code := run("greet", "world")
	if code != 0 || !strings.Contains(out, "hello [world]") {
		t.Errorf("greet answered %q with code %d", out, code)
	}

	if _, code := run("fail", "4"); code != 4 {
		t.Errorf("fail must propagate the exit code, got %d", code)
	}

	out, code = run("nope")
	if code != 2 || !strings.Contains(out, `unknown command "nope"`) {
		t.Errorf("an unknown command must fail loud, got %q with code %d", out, code)
	}
}

func TestTheCLIKindSkipsRegenerationOnAMatchingChecksum(t *testing.T) {
	dir := t.TempDir()
	writeKindFixture(t, dir, cliFixtureYaml())

	input := mcptypes.BuildInput{Name: "fixture-cli", Src: dir, Engine: "forge://forge-dev"}

	if _, err := generate(context.Background(), input); err != nil {
		t.Fatal(err)
	}

	before, err := os.Stat(filepath.Join(dir, GeneratedCLIFile))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := generate(context.Background(), input); err != nil {
		t.Fatal(err)
	}

	after, err := os.Stat(filepath.Join(dir, GeneratedCLIFile))
	if err != nil {
		t.Fatal(err)
	}

	if !after.ModTime().Equal(before.ModTime()) {
		t.Error("matching checksums must skip regeneration")
	}
}

type stubGeneratorCaller struct {
	model  GeneratorModel
	answer interface{}
	err    error
}

func (s *stubGeneratorCaller) ResolveEngine(string) (string, []string, error) {
	return "stub", nil, nil
}

func (s *stubGeneratorCaller) CallMCP(_ string, _ []string, _ string, params interface{}) (interface{}, error) {
	if m, ok := params.(GeneratorModel); ok {
		s.model = m
	}

	return s.answer, s.err
}

func withStubGenerator(t *testing.T, stub *stubGeneratorCaller) {
	t.Helper()

	previous := newGeneratorCaller
	newGeneratorCaller = func() generatorCaller { return stub }
	t.Cleanup(func() { newGeneratorCaller = previous })
}

func customKindYaml() string {
	return `name: fixture-gui
kind: gui
generator: forge://example.com/org/gui-gen/engines/gui-gen
version: 0.1.0
language: rust
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
  docsBaseURL: https://example.com/raw
surface:
  windows:
    - name: main-window
`
}

func TestAnExternalGeneratorOwnsACustomKind(t *testing.T) {
	dir := t.TempDir()
	writeKindFixture(t, dir, customKindYaml())

	stub := &stubGeneratorCaller{answer: map[string]interface{}{
		"files": []interface{}{
			map[string]interface{}{"path": "zz_generated_gui.rs", "content": "// gui\n"},
		},
	}}
	withStubGenerator(t, stub)

	_, err := generate(context.Background(), mcptypes.BuildInput{Name: "fixture-gui", Src: dir, Engine: "forge://forge-dev"})
	if err != nil {
		t.Fatalf("generate() through the external generator: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "zz_generated_gui.rs")); err != nil {
		t.Fatalf("the generator's file must be written: %v", err)
	}

	if stub.model.Kind != "gui" || stub.model.Language != "rust" {
		t.Errorf("the model must carry the cell, got kind %q language %q", stub.model.Kind, stub.model.Language)
	}

	if stub.model.Surface["windows"] == nil {
		t.Errorf("a custom surface must pass through opaquely: %v", stub.model.Surface)
	}

	if !strings.Contains(stub.model.OpenAPISpec, "greeting") {
		t.Error("the model must carry the raw OpenAPI spec")
	}

	if _, err := os.Stat(filepath.Join(dir, GeneratedRunnableFile)); err != nil {
		t.Errorf("core still owns the runnable manifest: %v", err)
	}
}

func TestAGeneratorPathOutsideTheEngineDirFailsLoud(t *testing.T) {
	dir := t.TempDir()
	writeKindFixture(t, dir, customKindYaml())

	stub := &stubGeneratorCaller{answer: map[string]interface{}{
		"files": []interface{}{
			map[string]interface{}{"path": "../escape.rs", "content": "// no\n"},
		},
	}}
	withStubGenerator(t, stub)

	_, err := generate(context.Background(), mcptypes.BuildInput{Name: "fixture-gui", Src: dir, Engine: "forge://forge-dev"})
	if err == nil || !strings.Contains(err.Error(), "outside the engine directory") {
		t.Fatalf("an escaping path must fail naming the offense, got %v", err)
	}
}

func TestKindValidationRules(t *testing.T) {
	base := func(kind string) *Config {
		c := &Config{Name: "x", Kind: kind, Version: "0.1.0"}
		c.OpenAPI.SpecPath = "./spec.openapi.yaml"
		c.Generate.PackageName = "main"

		return c
	}

	cases := []struct {
		name    string
		config  *Config
		field   string
		message string
	}{
		{
			"a custom kind needs a generator",
			base("gui"),
			"kind", `"gui" is not a builtin kind, so a generator: URI must own it`,
		},
		{
			"rest-api has no builtin cell yet",
			base(KindRestAPI),
			"kind", "rest-api has no builtin generator yet; name a generator: URI",
		},
		{
			"a profile outside mcp-server fails",
			func() *Config { c := base(KindBinary); c.Profile = "builder"; return c }(),
			"profile", "only the mcp-server kind has profiles",
		},
		{
			"the binary kind has no surface",
			func() *Config {
				c := base(KindBinary)
				c.Surface = &SurfaceConfig{Commands: []CommandConfig{{Name: "x", Description: "y"}}}

				return c
			}(),
			"surface", "the binary kind has no surface",
		},
		{
			"the cli kind needs commands",
			base(KindCLI),
			"surface.commands", "at least one command is required for the cli kind",
		},
		{
			"a generator must be a forge URI",
			func() *Config { c := base("gui"); c.Generator = "https://example.com"; return c }(),
			"generator", "must be a forge:// engine URI",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !hasError(ValidateConfig(tc.config), tc.field, tc.message) {
				t.Errorf("want %s: %s, got %v", tc.field, tc.message, ValidateConfig(tc.config))
			}
		})
	}
}

func TestTheCLIKindIsGoOnlyWithoutAGenerator(t *testing.T) {
	c := &Config{Name: "x", Kind: KindCLI, Version: "0.1.0", Language: "rust"}
	c.OpenAPI.SpecPath = "./spec.openapi.yaml"
	c.Generate.PackageName = "main"
	c.Surface = &SurfaceConfig{Commands: []CommandConfig{{Name: "run", Description: "Run."}}}

	if !hasError(ValidateConfig(c), "language", "the builtin cli cell generates go only; name a generator: for another language") {
		t.Errorf("a rust cli without a generator was accepted: %v", ValidateConfig(c))
	}

	c.Generator = "forge://example.com/org/gen/engines/gen"
	if hasError(ValidateConfig(c), "language", "") {
		t.Error("a generator-owned cell must accept any language")
	}
}

func TestAConfigGeneratorFillsTheConfigSurfaceOfACLICell(t *testing.T) {
	dir := t.TempDir()
	writeKindFixture(t, dir, cliFixtureYaml()+
		"configGenerator: forge://example.com/org/configgen/cmd/configgen-generator\n")

	stub := &stubGeneratorCaller{answer: map[string]interface{}{
		"files": []interface{}{
			map[string]interface{}{"path": "zz_generated.config.go", "content": "package main\n"},
		},
	}}
	withStubGenerator(t, stub)

	_, err := generate(context.Background(), mcptypes.BuildInput{Name: "fixture-cli", Src: dir, Engine: "forge://forge-dev"})
	if err != nil {
		t.Fatalf("generate() with a config generator: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "zz_generated.config.go")); err != nil {
		t.Fatalf("the config generator's file must be written: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, GeneratedCLIFile)); err != nil {
		t.Fatalf("the builtin cell keeps the dispatcher: %v", err)
	}

	if stub.model.Kind != KindCLI {
		t.Errorf("the model must carry the cell, got kind %q", stub.model.Kind)
	}

	if !strings.Contains(stub.model.OpenAPISpec, "greeting") {
		t.Error("the model must carry the raw OpenAPI spec: the Spec schema decides the keys")
	}
}

func TestConfigGeneratorValidationRules(t *testing.T) {
	base := func(kind string) *Config {
		c := &Config{Name: "x", Kind: kind, Version: "0.1.0"}
		c.OpenAPI.SpecPath = "./spec.openapi.yaml"
		c.Generate.PackageName = "main"
		c.ConfigGenerator = "forge://example.com/org/configgen/cmd/configgen-generator"

		return c
	}

	cases := []struct {
		name    string
		config  *Config
		message string
	}{
		{
			"only cli and binary take one",
			func() *Config { c := base(KindMCPServer); return c }(),
			"only the cli and binary kinds take a config generator; mcp-server derives config-validate from the Spec schema already",
		},
		{
			"a full generator owns config too",
			func() *Config {
				c := base("gui")
				c.Generator = "forge://example.com/org/gui-gen/engines/gui-gen"

				return c
			}(),
			"a generator: owns the whole cell, config included; drop one of the two",
		},
		{
			"a config generator must be a forge URI",
			func() *Config { c := base(KindBinary); c.ConfigGenerator = "https://example.com"; return c }(),
			"must be a forge:// engine URI",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !hasError(ValidateConfig(tc.config), "configGenerator", tc.message) {
				t.Errorf("want configGenerator: %s, got %v", tc.message, ValidateConfig(tc.config))
			}
		})
	}
}
