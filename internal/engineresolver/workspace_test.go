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

package engineresolver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const forgeModulePath = "github.com/alexandremahdhaoui/forge"

func chdirIntoWorkspaceWith(t *testing.T, modules ...string) {
	t.Helper()

	root := t.TempDir()
	goWork := "go 1.26\n\nuse (\n"
	for i, module := range modules {
		dir := filepath.Join(root, "m"+string(rune('a'+i)))
		require.NoError(t, os.MkdirAll(dir, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"),
			[]byte("module "+module+"\n\ngo 1.26\n"), 0o600))
		goWork += "\t./" + filepath.Base(dir) + "\n"
	}
	goWork += ")\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.work"), []byte(goWork), 0o600))

	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(filepath.Join(root, "ma")))
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	t.Setenv("GOWORK", "")
	t.Setenv("FORGE_RUN_LOCAL_ENABLED", "")
}

func TestABuiltinInAWorkspaceCarryingForgeRunsUnversionedFromTheCallersDirectory(t *testing.T) {
	chdirIntoWorkspaceWith(t, "example.com/caller", forgeModulePath)

	inv, err := ResolveForgeURI("forge://go-build", "v1.2.3")
	require.NoError(t, err)

	require.Equal(t, Invocation{Command: "go", Args: []string{"run", forgeModulePath + "/cmd/go-build"}}, inv)
}

func TestABuiltinOutsideAWorkspaceCarryingForgeKeepsThePinnedVersion(t *testing.T) {
	chdirIntoWorkspaceWith(t, "example.com/caller")

	inv, err := ResolveForgeURI("forge://go-build", "v1.2.3")
	require.NoError(t, err)

	require.Equal(t, "go", inv.Command)
	require.Equal(t, []string{"run", forgeModulePath + "/cmd/go-build@v1.2.3"}, inv.Args)
}

func TestParseEngineURIAndResolveForgeURIAgreeOnABuiltin(t *testing.T) {
	chdirIntoWorkspaceWith(t, "example.com/caller", forgeModulePath)

	engineType, command, args, err := ParseEngineURI("forge://testenv", "v1.2.3")
	require.NoError(t, err)
	require.Equal(t, EngineTypeMCP, engineType)

	inv, err := ResolveForgeURI("forge://testenv", "v1.2.3")
	require.NoError(t, err)
	require.Equal(t, inv.Command, command)
	require.Equal(t, inv.Args, args)
}
