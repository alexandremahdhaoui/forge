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

	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"sigs.k8s.io/yaml"
)

func listYAMLFor(t *testing.T, engine string) []byte {
	t.Helper()

	out, err := GenerateListYAML(&Config{Name: engine}, "sha256:deadbeef")
	if err != nil {
		t.Fatalf("GenerateListYAML: %v", err)
	}

	return out
}

func TestGenerateListYAMLEmitsAURLForEveryEntry(t *testing.T) {
	out := listYAMLFor(t, "go-build")

	for _, want := range []string{
		`url: "cmd/go-build/docs/usage.md"`,
		`url: "cmd/go-build/docs/schema.md"`,
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("generated list.yaml is missing %s\ngot:\n%s", want, out)
		}
	}
}

func TestGenerateListYAMLParsesAsADocStore(t *testing.T) {
	var store enginedocs.DocStore

	if err := yaml.Unmarshal(listYAMLFor(t, "go-build"), &store); err != nil {
		t.Fatalf("generated list.yaml does not parse as a DocStore: %v", err)
	}

	if store.Engine != "go-build" {
		t.Errorf("engine = %q, want go-build", store.Engine)
	}

	if len(store.Docs) != 2 {
		t.Fatalf("got %d docs, want usage and schema", len(store.Docs))
	}

	for _, d := range store.Docs {
		if d.URL == "" {
			t.Errorf("doc %q has an empty url, which is the field enginedocs reads", d.Name)
		}
	}
}

// TestGeneratedDocsPassEngineDocsValidate is the test that would have caught
// the shipped bug. Twenty four engines failed their own docs validate because
// the template emitted no url.
func TestGeneratedDocsPassEngineDocsValidate(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "cmd", "go-build", "docs")

	if err := os.MkdirAll(docs, 0o750); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"list.yaml", "usage.md", "schema.md"} {
		body := []byte("# placeholder\n")
		if name == "list.yaml" {
			body = listYAMLFor(t, "go-build")
		}

		if err := os.WriteFile(filepath.Join(docs, name), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.Chdir(wd) })

	errs := enginedocs.Validate(enginedocs.Config{
		EngineName:   "go-build",
		LocalDir:     "cmd/go-build/docs",
		BaseURL:      DocsBaseURL,
		RequiredDocs: []string{"usage", "schema"},
	})

	for _, e := range errs {
		t.Errorf("generated docs failed enginedocs.Validate: %v", e)
	}
}

func TestGenerateListYAMLUsesTheEngineNameInEveryPath(t *testing.T) {
	out := string(listYAMLFor(t, "ci-state-git"))

	if !strings.Contains(out, `engine: "ci-state-git"`) {
		t.Errorf("engine name missing from list.yaml:\n%s", out)
	}

	if strings.Contains(out, "go-build") {
		t.Errorf("list.yaml leaked another engine name:\n%s", out)
	}

	if !strings.Contains(out, "cmd/ci-state-git/docs/usage.md") {
		t.Errorf("url does not use the engine name:\n%s", out)
	}
}

func TestGenerateListYAMLCarriesTheChecksum(t *testing.T) {
	out, err := GenerateListYAML(&Config{Name: "go-build"}, "sha256:abc123")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(out), "# SourceChecksum: sha256:abc123") {
		t.Errorf("checksum header missing:\n%s", out)
	}
}
