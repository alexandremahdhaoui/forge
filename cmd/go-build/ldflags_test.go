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

	"github.com/stretchr/testify/require"
)

const sourceDirSymbol = "github.com/alexandremahdhaoui/forge/internal/forgepath.SourceDir"

func packageInAModule(t *testing.T) (string, string) {
	t.Helper()

	moduleRoot, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	pkgDir := filepath.Join(moduleRoot, "cmd", "tool")
	require.NoError(t, os.MkdirAll(pkgDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(moduleRoot, "go.mod"),
		[]byte("module example.com/tool\n\ngo 1.26\n"), 0o600))

	return moduleRoot, pkgDir
}

func TestABuildWhoseVersionIsNotAReleaseTagStampsTheSourceDirWithTheModuleRootOfThePackage(t *testing.T) {
	for _, version := range []string{"v0.50.10-16-gda6c582", "v0.50.11-0.20260917090238-38080aa3993a"} {
		t.Setenv("FORGE_CI_VERSION", version)
		moduleRoot, pkgDir := packageInAModule(t)

		flags := buildLDFlags(false, pkgDir)

		require.Contains(t, flags, "-X "+sourceDirSymbol+"="+moduleRoot, version)
	}
}

func TestABuildWhoseVersionIsAReleaseTagCarriesNoSourceDirSoItStaysByteReproducible(t *testing.T) {
	t.Setenv("FORGE_CI_VERSION", "v0.50.0")
	_, pkgDir := packageInAModule(t)

	flags := buildLDFlags(false, pkgDir)

	require.Contains(t, flags, "main.Version=v0.50.0")
	require.NotContains(t, flags, sourceDirSymbol)
}
