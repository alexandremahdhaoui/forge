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

// Package e2erun proves the run system end to end through real binaries:
// real git repos, a real factory, a real register carrying versions with
// provenance, and a real state repo holding the proving revision. No
// network beyond the module cache. The forge-factory sibling checkout is
// required: the run system is half forge and half forge-factory, and this
// suite is what proves the seam.
package e2erun

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	buildOnce sync.Once
	binDir    string
	forgeRepo string
	buildErr  error
)

func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the test directory")
		}

		dir = parent
	}
}

// binaries builds forge and forge-factory once per test run. forge comes
// from this repo. forge-factory comes from the sibling checkout, because
// the run system is a contract between the two.
func binaries(t *testing.T) (string, string) {
	t.Helper()

	buildOnce.Do(func() {
		forgeRepo = repoRoot(t)

		factoryRepo := filepath.Join(filepath.Dir(forgeRepo), "forge-factory")
		if _, err := os.Stat(filepath.Join(factoryRepo, "go.mod")); err != nil {
			buildErr = fmt.Errorf("the forge-factory sibling checkout is required at %s", factoryRepo)

			return
		}

		binDir, buildErr = os.MkdirTemp("", "e2erun-bin")
		if buildErr != nil {
			return
		}

		builds := [][3]string{
			{forgeRepo, "./cmd/forge", "forge"},
			{factoryRepo, "./cmd/forge-factory", "forge-factory"},
		}

		for _, b := range builds {
			cmd := exec.Command("go", "build", "-o", filepath.Join(binDir, b[2]), b[1])
			cmd.Dir = b[0]
			cmd.Env = append(os.Environ(), "GOWORK=off")

			if out, err := cmd.CombinedOutput(); err != nil {
				buildErr = fmt.Errorf("building %s: %w: %s", b[2], err, out)

				return
			}
		}
	})

	if buildErr != nil {
		t.Fatal(buildErr)
	}

	return binDir, forgeRepo
}

// universe is one hermetic fixture world: its own HOME, its own cache, and
// every repo a real git repo reachable by absolute path.
type universe struct {
	t    *testing.T
	root string
	home string
	bin  string
}

func newUniverse(t *testing.T) *universe {
	t.Helper()

	bin, _ := binaries(t)

	root := t.TempDir()
	home := filepath.Join(root, "home")

	if err := os.MkdirAll(home, 0o750); err != nil {
		t.Fatal(err)
	}

	return &universe{t: t, root: root, home: home, bin: bin}
}

func (u *universe) env() []string {
	_, forge := binaries(u.t)

	return append(os.Environ(),
		"HOME="+u.home,
		"PATH="+u.bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FORGE_RUN_LOCAL_ENABLED=true",
		"FORGE_RUN_LOCAL_BASEDIR="+forge,
		"GOWORK=off",
		"FORGE_RUN_MATERIALIZED=",
	)
}

type result struct {
	stdout string
	stderr string
	code   int
}

func (u *universe) run(dir string, extraEnv []string, name string, args ...string) result {
	u.t.Helper()

	cmd := exec.Command(filepath.Join(u.bin, name), args...)
	cmd.Dir = dir
	cmd.Env = append(u.env(), extraEnv...)

	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb

	err := cmd.Run()

	code := 0
	if exit, ok := err.(*exec.ExitError); ok {
		code = exit.ExitCode()
	} else if err != nil {
		u.t.Fatalf("running %s %v: %v\nstderr: %s", name, args, err, errb.String())
	}

	return result{stdout: out.String(), stderr: errb.String(), code: code}
}

func (u *universe) git(dir string, args ...string) string {
	u.t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=e2e", "GIT_AUTHOR_EMAIL=e2e@example.com",
		"GIT_COMMITTER_NAME=e2e", "GIT_COMMITTER_EMAIL=e2e@example.com",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		u.t.Fatalf("git %v in %s: %v: %s", args, dir, err, out)
	}

	return strings.TrimSpace(string(out))
}

func (u *universe) writeRepo(name string, files map[string]string) string {
	u.t.Helper()

	dir := filepath.Join(u.root, name)

	for path, content := range files {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			u.t.Fatal(err)
		}

		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			u.t.Fatal(err)
		}
	}

	u.git(dir, "init", "-q", "-b", "main")
	u.git(dir, "add", "-A")
	u.git(dir, "-c", "commit.gpgsign=false", "commit", "-q", "-m", "fixture")

	return dir
}

func (u *universe) headSha(dir string) string {
	u.t.Helper()

	return u.git(dir, "rev-parse", "HEAD")
}

const toolMain = `package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		if code, err := strconv.Atoi(args[0]); err == nil {
			fmt.Println(strings.Join(args[1:], " "))
			os.Exit(code)
		}
	}

	fmt.Println(strings.Join(args, " "))
}
`

func toolForgeYaml(factoryURL string) string {
	return `name: tool-a
artifactStorePath: .forge/artifact-store.yaml
build:
  - name: tool-a
    src: ./cmd/tool-a
    dest: ./build/bin
    engine: forge://go-build
run:
  - name: tool-a
    src: ./cmd/tool-a
    factory: ` + factoryURL + "\n"
}

// toolRepo is the compiled member: prints its argv, and a numeric first
// arg becomes the exit code.
func (u *universe) toolRepo(factoryURL string) string {
	return u.writeRepo("tool-a", map[string]string{
		"forge.yaml":         toolForgeYaml(factoryURL),
		"go.mod":             "module example.com/tool-a\n\ngo 1.25\n",
		"cmd/tool-a/main.go": toolMain,
		".envrc":             "",
		".gitignore":         "/build/\n/.forge/\n",
	})
}

func factoryYaml(members map[string]string, registerURL string, withState bool) string {
	var b strings.Builder

	b.WriteString("version: \"1\"\nname: fixture\n\nrepos:\n")

	for _, name := range sortedKeys(members) {
		fmt.Fprintf(&b, "  - name: %s\n    url: %s\n", name, members[name])
	}

	b.WriteString("\nengines:\n  - alias: gg\n    engine: forge://generic-builder\n")

	if registerURL != "" {
		fmt.Fprintf(&b, "\nregister:\n  url: %s\n", registerURL)
	}

	if withState {
		b.WriteString("\nstate:\n  engine: forge://ci-state-git\n  spec:\n    path: ./state\n")
	}

	return b.String()
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	for i := range keys {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	return keys
}

func (u *universe) factoryRepo(name string, members map[string]string, registerURL string, withState bool) string {
	return u.writeRepo(name, map[string]string{
		"workspace/forge-factory.yaml": factoryYaml(members, registerURL, withState),
	})
}

func trackJSON(module, version, provenance string) (string, string) {
	path := filepath.Join("index", "internal", strings.TrimPrefix(module, "/"), "0.json")
	content := fmt.Sprintf(
		`{"current":%q,"ecosystem":"internal","package":%q,"prefix":"0","history":[{"version":%q,"provenance":%q}]}`,
		version, module, version, provenance)

	return path, content
}

func revisionJSON(repos map[string]string) string {
	pairs := []string{}
	for _, name := range sortedKeys(repos) {
		pairs = append(pairs, fmt.Sprintf("%q:%q", name, repos[name]))
	}

	return `{"repos":{` + strings.Join(pairs, ",") + `}}`
}
