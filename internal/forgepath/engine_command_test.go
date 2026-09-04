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

package forgepath

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func chdirIntoWorkspaceListing(t *testing.T, memberModules ...string) string {
	t.Helper()

	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)

	goWork := "go 1.26\n\nuse (\n"
	for i, module := range memberModules {
		dir := filepath.Join(root, "member"+string(rune('a'+i)))
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd", "go-build"), 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"),
			[]byte("module "+module+"\n\ngo 1.26\n"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "cmd", "go-build", "main.go"),
			[]byte("package main\n\nfunc main() {}\n"), 0o600))
		goWork += "\t./" + filepath.Base(dir) + "\n"
	}
	goWork += ")\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.work"), []byte(goWork), 0o600))

	caller := filepath.Join(root, "membera")
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(caller))
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	t.Setenv("GOWORK", "")
	t.Setenv("FORGE_RUN_LOCAL_ENABLED", "")
	t.Setenv("FORGE_RUN_LOCAL_BASEDIR", "")

	return filepath.Join(root, "memberb")
}

func TestABuiltinRunsUnversionedFromTheWorkspaceCheckoutWhenGoWorkCarriesForge(t *testing.T) {
	chdirIntoWorkspaceListing(t, "example.com/caller", forgeModule)

	command, args, err := EngineCommand("go-build", "v1.2.3")
	require.NoError(t, err)

	require.Equal(t, "go", command)
	require.Equal(t, []string{"run", forgeModule + "/cmd/go-build"}, args)
}

func TestABuiltinKeepsThePinnedVersionWhenGoWorkDoesNotCarryForge(t *testing.T) {
	chdirIntoWorkspaceListing(t, "example.com/caller", "example.com/other")

	command, args, err := EngineCommand("go-build", "v1.2.3")
	require.NoError(t, err)

	require.Equal(t, "go", command)
	require.Equal(t, []string{"run", forgeModule + "/cmd/go-build@v1.2.3"}, args)
}

func TestTheLocalFlagForcesABuildFromTheNamedCheckoutEvenInsideAWorkspace(t *testing.T) {
	forgeDir := chdirIntoWorkspaceListing(t, "example.com/caller", forgeModule)
	t.Setenv("FORGE_RUN_LOCAL_ENABLED", "true")
	t.Setenv("FORGE_RUN_LOCAL_BASEDIR", forgeDir)

	command, args, err := EngineCommand("go-build", "v1.2.3")
	require.NoError(t, err)

	require.Equal(t, filepath.Join(forgeDir, "build", "local-engines", "go-build"), command)
	require.Nil(t, args)
	require.FileExists(t, command)
}

func TestAnEmptyBuiltinNameIsRefused(t *testing.T) {
	chdirIntoWorkspaceListing(t, "example.com/caller", forgeModule)

	_, _, err := EngineCommand("", "v1.2.3")
	require.Error(t, err)
}
