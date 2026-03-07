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

package cmdutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadUserConfig_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	content := `tools:
  cu: /usr/local/bin/forge-cu
  ws: /usr/local/bin/forge-ws
  ui: /usr/local/bin/forge-ui
`
	require.NoError(t, os.WriteFile(configFile, []byte(content), 0o644))

	cfg, err := loadUserConfigFromPath(configFile)
	require.NoError(t, err)
	assert.Equal(t, "/usr/local/bin/forge-cu", cfg.Tools.CU)
	assert.Equal(t, "/usr/local/bin/forge-ws", cfg.Tools.WS)
	assert.Equal(t, "/usr/local/bin/forge-ui", cfg.Tools.UI)
}

func TestLoadUserConfig_MissingFile(t *testing.T) {
	cfg, err := loadUserConfigFromPath("/non/existent/path/config.yaml")
	require.NoError(t, err)
	assert.Equal(t, UserConfig{}, cfg)
}

func TestLoadUserConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	content := `tools:
  cu: [invalid yaml
  ws: "unclosed
`
	require.NoError(t, os.WriteFile(configFile, []byte(content), 0o644))

	_, err := loadUserConfigFromPath(configFile)
	assert.Error(t, err)
}

func TestResolveToolBinary_UserOverride(t *testing.T) {
	binary, args, err := ResolveToolBinary("/custom/path/tool", "tool", "github.com/example/mod", "v1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "/custom/path/tool", binary)
	assert.Nil(t, args)
}

func TestResolveToolBinary_LookPath(t *testing.T) {
	// "go" binary should exist on PATH in any Go development environment.
	binary, args, err := ResolveToolBinary("", "go", "github.com/example/mod", "v1.0.0")
	require.NoError(t, err)
	assert.Contains(t, binary, "go")
	assert.Nil(t, args)
}

func TestResolveToolBinary_GoRunFallback(t *testing.T) {
	binary, args, err := ResolveToolBinary("", "nonexistent-forge-binary-xyz-12345", "github.com/example/mod", "v1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "go", binary)
	assert.NotNil(t, args)
}

func TestResolveToolBinary_NotFound(t *testing.T) {
	_, _, err := ResolveToolBinary("", "nonexistent-forge-binary-xyz-12345", "", "v1.0.0")
	assert.Error(t, err)
}
