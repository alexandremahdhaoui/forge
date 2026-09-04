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
	"testing"

	"github.com/alexandremahdhaoui/forge/internal/forgepath"
	"github.com/alexandremahdhaoui/forge/pkg/toolresolver"
	"github.com/stretchr/testify/require"
)

func chdirIntoWorkspaceMember(t *testing.T, modulePath string) {
	t.Helper()

	root := t.TempDir()
	member := filepath.Join(root, "member")
	require.NoError(t, os.MkdirAll(member, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.work"),
		[]byte("go 1.26\n\nuse ./member\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(member, "go.mod"),
		[]byte("module "+modulePath+"\n\ngo 1.26\n"), 0o600))

	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(member))
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	t.Setenv("GOWORK", "")
	t.Setenv("FORGE_RUN_LOCAL_ENABLED", "")
}

func TestAWorkspaceMemberRunsUnversionedFromTheCallersDirectoryWithoutTheLocalFlag(t *testing.T) {
	chdirIntoWorkspaceMember(t, "example.com/member")

	engine, err := resolveEngineURI("forge://example.com/member/cmd/testenv-thing@v1.2.3")
	require.NoError(t, err)

	require.Equal(t, engineInvocation{
		command: "go",
		args:    []string{"run", "example.com/member/cmd/testenv-thing"},
	}, engine)
}

func TestAMemberOutsideTheWorkspaceMaterializesThroughForgeFactoryFromTheCallersDirectory(t *testing.T) {
	chdirIntoWorkspaceMember(t, "example.com/member")

	factory, err := toolresolver.ForgeFactory()
	require.NoError(t, err)

	engine, err := resolveEngineURI("forge://example.com/other/cmd/tool@v2.0.0")
	require.NoError(t, err)

	require.Equal(t, engineInvocation{
		command: factory.Path,
		args:    append(append([]string{}, factory.Args...), "run", "--quiet", "example.com/other/cmd/tool@v2.0.0", "--", "--mcp"),
	}, engine)
}

func TestForgesOwnEngineRunsFromTheForgeModuleDirectory(t *testing.T) {
	forgeRepo, err := forgepath.FindForgeRepo()
	require.NoError(t, err)

	chdirIntoWorkspaceMember(t, "example.com/member")

	engine, err := resolveEngineURI("forge://testenv-kind")
	require.NoError(t, err)

	require.Equal(t, "go", engine.command)
	require.Equal(t, "run", engine.args[0])
	require.Contains(t, engine.args[1], "github.com/alexandremahdhaoui/forge/cmd/testenv-kind@")
	require.Equal(t, forgeRepo, engine.dir)
}

func TestAFullForgeModulePathIsOneOfForgesOwnEngines(t *testing.T) {
	chdirIntoWorkspaceMember(t, "example.com/member")

	engine, err := resolveEngineURI("forge://github.com/alexandremahdhaoui/forge/cmd/testenv-stub")
	require.NoError(t, err)

	require.Equal(t, "go", engine.command)
	require.Contains(t, engine.args[1], "github.com/alexandremahdhaoui/forge/cmd/testenv-stub@")
}

func TestAnEngineURIWithoutTheForgeSchemeIsRefused(t *testing.T) {
	_, err := resolveEngineURI("alias://setup")
	require.Error(t, err)

	_, err = resolveEngineURI("forge://")
	require.Error(t, err)
}
