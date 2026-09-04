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

func writeForgeDevYaml(t *testing.T, dir, body string) {
	t.Helper()

	require.NoError(t, os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(body), 0o644))
}

func TestAForgeDevYamlThatStillCarriesSurfaceFailsToReadNamingTheRename(t *testing.T) {
	dir := t.TempDir()
	writeForgeDevYaml(t, dir, `name: stale
kind: mcp-server
version: 0.1.0
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
surface:
  tools:
    - name: get
      description: Read one record.
      input: StateGetInput
`)

	_, err := ReadConfig(dir)

	require.Error(t, err)
	require.Contains(t, err.Error(), "surface was renamed to layout on 2026-09-04, rename the block")
}

func TestALayoutBlockReadsItsToolsAndItsExtraKeys(t *testing.T) {
	dir := t.TempDir()
	writeForgeDevYaml(t, dir, `name: fresh
kind: mcp-server
version: 0.1.0
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
layout:
  tools:
    - name: get
      description: Read one record.
      input: StateGetInput
  cells: [rest, grpc]
`)

	config, err := ReadConfig(dir)

	require.NoError(t, err)
	require.NotNil(t, config.Layout)
	require.Len(t, config.Layout.Tools, 1)
	require.Equal(t, "get", config.Layout.Tools[0].Name)
	require.Contains(t, config.Layout.Extra, "cells")
}

func TestTheLayoutBlockTravelsToTheGeneratorUnderTheLayoutKey(t *testing.T) {
	dir := t.TempDir()
	writeForgeDevYaml(t, dir, `name: fixture-gui
kind: gui
generator: forge://example.com/org/gui-gen/engines/gui-gen
version: 0.1.0
language: rust
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
layout:
  cells: [rest]
`)

	model := GeneratorModel{}
	config, err := ReadConfig(dir)
	require.NoError(t, err)

	model.Layout = layoutMap(config)

	require.Contains(t, model.Layout, "cells")
}
