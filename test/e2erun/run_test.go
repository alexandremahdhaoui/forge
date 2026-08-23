//go:build e2erun

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

package e2erun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// workspaceWorld is the daily dev loop: a workspace root whose factory
// claims the members, no register needed.
func workspaceWorld(t *testing.T) (*universe, string, string) {
	t.Helper()

	u := newUniverse(t)

	ws := filepath.Join(u.root, "ws")
	if err := os.MkdirAll(ws, 0o750); err != nil {
		t.Fatal(err)
	}

	factoryURL := "git@example.com:org/fixture-factory.git"

	tool := u.writeRepo(filepath.Join("ws", "tool-a"), map[string]string{
		"forge.yaml":         toolForgeYaml(factoryURL),
		"go.mod":             "module example.com/tool-a\n\ngo 1.25\n",
		"cmd/tool-a/main.go": toolMain,
		".envrc":             "",
		".gitignore":         "/build/\n/.forge/\n",
	})

	interp := u.interpRepoAt(filepath.Join("ws", "interp"), factoryURL)

	members := map[string]string{"tool-a": tool, "interp": interp}
	if err := os.WriteFile(filepath.Join(ws, "forge-factory.yaml"),
		[]byte(factoryYaml(members, "", false)), 0o644); err != nil {
		t.Fatal(err)
	}

	return u, tool, interp
}

func (u *universe) interpRepoAt(name, factoryURL string) string {
	u.t.Helper()

	return u.writeRepo(name, map[string]string{
		"forge.yaml": `name: interp
artifactStorePath: .forge/artifact-store.yaml
build:
  - name: interp
    src: ./run.sh
    engine: forge://generic-builder
    spec:
      command: "true"
run:
  - name: interp
    src: ./run.sh
    engine: forge://generic-runner
    factory: ` + factoryURL + `
    spec:
      command: sh
      args: ["run.sh"]
`,
		"run.sh":     "#!/bin/sh\necho \"interp $*\"\n",
		".envrc":     "",
		".gitignore": "/.forge/\n",
	})
}

func TestNameFormRunsInTheClaimingWorkspace(t *testing.T) {
	u, tool, _ := workspaceWorld(t)

	got := u.run(tool, nil, "forge", "run", "tool-a", "--", "x", "y")
	if got.code != 0 {
		t.Fatalf("exit %d\nstderr: %s", got.code, got.stderr)
	}

	if !strings.Contains(got.stdout, "x y") {
		t.Errorf("stdout = %q, want the argv echoed", got.stdout)
	}

	if !strings.Contains(got.stderr, "rule 2") {
		t.Errorf("stderr must name rule 2:\n%s", got.stderr)
	}
}

func TestPathFormBehavesLikeTheNameForm(t *testing.T) {
	u, tool, _ := workspaceWorld(t)

	got := u.run(tool, nil, "forge", "run", "./cmd/tool-a", "--", "z")
	if got.code != 0 {
		t.Fatalf("exit %d\nstderr: %s", got.code, got.stderr)
	}

	if !strings.Contains(got.stdout, "z") || !strings.Contains(got.stderr, "rule 2") {
		t.Errorf("stdout=%q stderr:\n%s", got.stdout, got.stderr)
	}
}

func TestAnInterpretedTargetRunsThroughTheGenericRunner(t *testing.T) {
	u, _, interp := workspaceWorld(t)

	got := u.run(interp, nil, "forge", "run", "interp", "--", "a", "b")
	if got.code != 0 {
		t.Fatalf("exit %d\nstderr: %s", got.code, got.stderr)
	}

	if !strings.Contains(got.stdout, "interp a b") {
		t.Errorf("stdout = %q, want the interpreter output", got.stdout)
	}
}

func TestExitCodesPropagateThroughEveryLayer(t *testing.T) {
	u, tool, _ := workspaceWorld(t)

	got := u.run(tool, nil, "forge", "run", "tool-a", "--", "3", "boom")
	if got.code != 3 {
		t.Fatalf("exit = %d, want 3\nstderr: %s", got.code, got.stderr)
	}

	if !strings.Contains(got.stdout, "boom") {
		t.Errorf("stdout = %q", got.stdout)
	}
}

func TestARunnableWithoutAFactoryFailsValidation(t *testing.T) {
	u := newUniverse(t)

	repo := u.writeRepo("orphan", map[string]string{
		"forge.yaml": `name: orphan
artifactStorePath: .forge/artifact-store.yaml
build:
  - name: orphan
    src: ./cmd/orphan
    dest: ./build/bin
    engine: forge://go-build
run:
  - name: orphan
    src: ./cmd/orphan
`,
		"go.mod":             "module example.com/orphan\n\ngo 1.25\n",
		"cmd/orphan/main.go": toolMain,
		".envrc":             "",
	})

	got := u.run(repo, nil, "forge", "build")
	if got.code == 0 {
		t.Fatal("a runnable without a factory must fail validation")
	}

	if !strings.Contains(got.stderr+got.stdout, "factory") {
		t.Errorf("the error must name the missing key:\n%s%s", got.stderr, got.stdout)
	}
}

func TestTheGoSchemeIsRejectedWithTheMigration(t *testing.T) {
	u := newUniverse(t)

	repo := u.writeRepo("stale", map[string]string{
		"forge.yaml": `name: stale
artifactStorePath: .forge/artifact-store.yaml
build:
  - name: stale
    src: .
    engine: go://generic-builder
    spec:
      command: "true"
`,
		".envrc": "",
	})

	got := u.run(repo, nil, "forge", "build")
	if got.code == 0 {
		t.Fatal("a go:// URI must fail")
	}

	if !strings.Contains(got.stderr+got.stdout, "the go:// scheme is removed; use forge://") {
		t.Errorf("the error must name forge://:\n%s%s", got.stderr, got.stdout)
	}
}

// remoteWorld is the proven-tuple world: a factory with a register whose
// internal track names the version, and a state repo whose revision record
// pins the member shas.
type remoteWorld struct {
	u        *universe
	tool     string
	factory  string
	register string
	state    string
	pinned   string
}

func newRemoteWorld(t *testing.T) *remoteWorld {
	t.Helper()

	u := newUniverse(t)

	factoryDir := filepath.Join(u.root, "fixture-factory")

	tool := u.toolRepo(factoryDir)
	pinned := u.headSha(tool)

	u.git(tool, "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "moved past the proof")

	state := u.writeRepo("state", map[string]string{
		"revisions/rev1.json": revisionJSON(map[string]string{"tool-a": pinned}),
	})

	trackPath, trackContent := trackJSON(tool, "v1.0.0", "rev1")
	register := u.writeRepo("register", map[string]string{trackPath: trackContent})

	members := map[string]string{"tool-a": tool, "register": register, "state": state}
	factory := u.factoryRepo("fixture-factory", members, register, true)

	return &remoteWorld{u: u, tool: tool, factory: factory, register: register, state: state, pinned: pinned}
}

func TestARemoteRunResolvesTheProvenTuple(t *testing.T) {
	w := newRemoteWorld(t)

	got := w.u.run(w.u.root, nil, "forge", "run", w.tool, "tool-a", "--", "hello")
	if got.code != 0 {
		t.Fatalf("exit %d\nstderr: %s", got.code, got.stderr)
	}

	if !strings.Contains(got.stdout, "hello") {
		t.Errorf("stdout = %q", got.stdout)
	}

	for _, phase := range []string{"clone:", "resolve-target:", "resolve-factory:", "resolve-version:", "pin:", "checkout:", "exec:"} {
		if !strings.Contains(got.stderr, phase) {
			t.Errorf("stderr must carry phase %q:\n%s", phase, got.stderr)
		}
	}

	if !strings.Contains(got.stderr, "v1.0.0") || !strings.Contains(got.stderr, "rev1") {
		t.Errorf("the version and its provenance must be named:\n%s", got.stderr)
	}

	worktrees := filepath.Join(w.u.home, ".cache", "forge-factory", "run")
	entries, err := os.ReadDir(worktrees)
	if err != nil || len(entries) == 0 {
		t.Fatalf("no run context under the cache: %v", err)
	}

	repoDir := filepath.Join(worktrees, entries[0].Name(), "tool-a")
	if sha := w.u.git(repoDir, "rev-parse", "HEAD"); sha != w.pinned {
		t.Errorf("the worktree sits at %s, want the provenance-pinned %s", sha, w.pinned)
	}
}

func TestASecondRemoteRunIsWarm(t *testing.T) {
	w := newRemoteWorld(t)

	first := w.u.run(w.u.root, nil, "forge", "run", w.tool, "tool-a", "--", "one")
	if first.code != 0 {
		t.Fatalf("cold run: exit %d\nstderr: %s", first.code, first.stderr)
	}

	second := w.u.run(w.u.root, nil, "forge", "run", w.tool, "tool-a", "--", "two")
	if second.code != 0 {
		t.Fatalf("warm run: exit %d\nstderr: %s", second.code, second.stderr)
	}

	if !strings.Contains(second.stderr, "cache: warm") {
		t.Errorf("the second run must hit the warm cache:\n%s", second.stderr)
	}

	if strings.Contains(second.stderr, "checkout:") {
		t.Errorf("a warm run must not check out again:\n%s", second.stderr)
	}
}

func TestADevRevIsVisiblyUnproven(t *testing.T) {
	w := newRemoteWorld(t)

	head := w.u.headSha(w.tool)

	got := w.u.run(w.u.root, nil, "forge", "run", w.tool+"/cmd/tool-a@"+head, "--", "dev")
	if got.code != 0 {
		t.Fatalf("exit %d\nstderr: %s", got.code, got.stderr)
	}

	if !strings.Contains(got.stderr, "UNPROVEN") {
		t.Errorf("a dev rev must be marked unproven:\n%s", got.stderr)
	}

	worktrees := filepath.Join(w.u.home, ".cache", "forge-factory", "run")
	entries, _ := os.ReadDir(worktrees)

	found := false

	for _, e := range entries {
		repoDir := filepath.Join(worktrees, e.Name(), "tool-a")
		if _, err := os.Stat(repoDir); err != nil {
			continue
		}

		if w.u.git(repoDir, "rev-parse", "HEAD") == head {
			found = true
		}
	}

	if !found {
		t.Error("no worktree sits at the asked dev rev")
	}
}

func TestTheTrustGateRefusesANonMember(t *testing.T) {
	w := newRemoteWorld(t)

	stranger := w.u.writeRepo("stranger", map[string]string{
		"forge.yaml":           strings.ReplaceAll(toolForgeYaml(w.factory), "tool-a", "stranger"),
		"go.mod":               "module example.com/stranger\n\ngo 1.25\n",
		"cmd/stranger/main.go": toolMain,
		".envrc":               "",
	})

	got := w.u.run(w.u.root, nil, "forge", "run", stranger, "stranger")
	if got.code == 0 {
		t.Fatal("a non-member must fail the trust gate")
	}

	if !strings.Contains(got.stderr, "not a member") || !strings.Contains(got.stderr, "fixture-factory") {
		t.Errorf("the failure must name the factory:\n%s", got.stderr)
	}
}

func TestAMissingInputFailsBeforeTheBuild(t *testing.T) {
	u := newUniverse(t)

	ws := filepath.Join(u.root, "ws")
	if err := os.MkdirAll(ws, 0o750); err != nil {
		t.Fatal(err)
	}

	tool := u.writeRepo(filepath.Join("ws", "tool-a"), map[string]string{
		"forge.yaml":         toolForgeYaml("git@example.com:org/fixture-factory.git"),
		"go.mod":             "module example.com/tool-a\n\ngo 1.25\n",
		"cmd/tool-a/main.go": toolMain,
		"cmd/tool-a/zz_generated.runnable.yaml": `# Code generated by forge-dev. DO NOT EDIT.
name: tool-a
inputs:
  env:
    - TOOL_A_TOKEN
  files: []
`,
		".envrc":     "",
		".gitignore": "/build/\n/.forge/\n",
	})

	if err := os.WriteFile(filepath.Join(ws, "forge-factory.yaml"),
		[]byte(factoryYaml(map[string]string{"tool-a": tool}, "", false)), 0o644); err != nil {
		t.Fatal(err)
	}

	unset := u.run(tool, nil, "forge", "run", "tool-a", "--", "x")
	if unset.code == 0 {
		t.Fatalf("a missing input must fail:\n%s", unset.stderr)
	}

	if !strings.Contains(unset.stderr, "TOOL_A_TOKEN") {
		t.Errorf("the failure must name the input:\n%s", unset.stderr)
	}

	set := u.run(tool, []string{"TOOL_A_TOKEN=abc"}, "forge", "run", "tool-a", "--", "x")
	if set.code != 0 {
		t.Fatalf("with the input set the run must pass:\n%s", set.stderr)
	}
}

func TestForgeCloneStandsAWorkspaceUp(t *testing.T) {
	w := newRemoteWorld(t)

	dest := filepath.Join(w.u.root, "fresh")

	got := w.u.run(w.u.root, nil, "forge", "clone", w.factory, dest)
	if got.code != 0 {
		t.Fatalf("exit %d\nstderr: %s\nstdout: %s", got.code, got.stderr, got.stdout)
	}

	if _, err := os.Stat(filepath.Join(dest, "forge-factory.yaml")); err != nil {
		t.Fatalf("the factory file must be placed: %v", err)
	}

	for _, member := range []string{"tool-a", "register", "state"} {
		if _, err := os.Stat(filepath.Join(dest, member, ".git")); err != nil {
			t.Errorf("member %s must be cloned: %v", member, err)
		}
	}
}
