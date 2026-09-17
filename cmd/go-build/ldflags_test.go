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

func TestBuildLDFlagsStampsTheSourceDirWithTheModuleRootOfThePackage(t *testing.T) {
	moduleRoot, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	pkgDir := filepath.Join(moduleRoot, "cmd", "tool")
	require.NoError(t, os.MkdirAll(pkgDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(moduleRoot, "go.mod"),
		[]byte("module example.com/tool\n\ngo 1.26\n"), 0o600))

	flags := buildLDFlags(false, pkgDir)

	require.Contains(t, flags,
		"-X github.com/alexandremahdhaoui/forge/internal/forgepath.SourceDir="+moduleRoot)
}
