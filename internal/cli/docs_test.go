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

package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestDocsDir creates a temporary docs directory with list.yaml and doc files
func createTestDocsDir(t *testing.T, tmpDir string) string {
	t.Helper()
	docsDir := filepath.Join(tmpDir, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0o755))

	listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: usage
    title: Usage Guide
    description: How to use test-engine
    url: docs/usage.md
  - name: schema
    title: Configuration Schema
    description: JSON schema for configuration
    url: docs/schema.md
`
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "usage.md"), []byte("# Usage Guide\n\nThis is the usage guide."), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "schema.md"), []byte("# Schema\n\nThis is the schema."), 0o644))

	return docsDir
}

// captureOutput captures stdout and stderr during function execution
func captureOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()

	// Capture stdout
	oldStdout := os.Stdout
	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = stdoutW

	// Capture stderr
	oldStderr := os.Stderr
	stderrR, stderrW, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = stderrW

	// Run the function
	fn()

	// Restore and read output
	stdoutW.Close()
	stderrW.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var stdoutBuf, stderrBuf bytes.Buffer
	io.Copy(&stdoutBuf, stdoutR)
	io.Copy(&stderrBuf, stderrR)

	return stdoutBuf.String(), stderrBuf.String()
}

// setupTestWithCwd creates a temp directory and changes to it
func setupTestWithCwd(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir := t.TempDir()

	originalWd, err := os.Getwd()
	require.NoError(t, err)

	require.NoError(t, os.Chdir(tmpDir))

	return tmpDir, func() {
		_ = os.Chdir(originalWd)
	}
}

func TestHandleDocsCommand_List(t *testing.T) {
	// Cannot be parallel because changes working directory

	t.Run("list with no args returns docs", func(t *testing.T) {
		tmpDir, cleanup := setupTestWithCwd(t)
		defer cleanup()

		docsDir := createTestDocsDir(t, tmpDir)
		cfg := &enginedocs.Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		stdout, stderr := captureOutput(t, func() {
			exitCode := handleDocsCommand(cfg, []string{})
			assert.Equal(t, 0, exitCode)
		})

		assert.Contains(t, stdout, "usage")
		assert.Contains(t, stdout, "Usage Guide")
		assert.Contains(t, stdout, "schema")
		assert.Contains(t, stdout, "Configuration Schema")
		assert.Empty(t, stderr)
	})

	t.Run("list explicit subcommand returns docs", func(t *testing.T) {
		tmpDir, cleanup := setupTestWithCwd(t)
		defer cleanup()

		docsDir := createTestDocsDir(t, tmpDir)
		cfg := &enginedocs.Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		stdout, stderr := captureOutput(t, func() {
			exitCode := handleDocsCommand(cfg, []string{"list"})
			assert.Equal(t, 0, exitCode)
		})

		assert.Contains(t, stdout, "usage")
		assert.Contains(t, stdout, "schema")
		assert.Empty(t, stderr)
	})

	t.Run("list returns error when config is invalid", func(t *testing.T) {
		cfg := &enginedocs.Config{
			EngineName: "test-engine",
			LocalDir:   "/non/existent/path",
			BaseURL:    "", // No fallback
		}

		stdout, stderr := captureOutput(t, func() {
			exitCode := handleDocsCommand(cfg, []string{"list"})
			assert.Equal(t, 1, exitCode)
		})

		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Error listing docs")
	})
}

func TestHandleDocsCommand_Get(t *testing.T) {
	// Cannot be parallel because changes working directory

	t.Run("get returns document content", func(t *testing.T) {
		tmpDir, cleanup := setupTestWithCwd(t)
		defer cleanup()

		docsDir := createTestDocsDir(t, tmpDir)
		cfg := &enginedocs.Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		stdout, stderr := captureOutput(t, func() {
			exitCode := handleDocsCommand(cfg, []string{"get", "usage"})
			assert.Equal(t, 0, exitCode)
		})

		assert.Contains(t, stdout, "# Usage Guide")
		assert.Contains(t, stdout, "This is the usage guide.")
		assert.Empty(t, stderr)
	})

	t.Run("get returns error for non-existent doc", func(t *testing.T) {
		tmpDir, cleanup := setupTestWithCwd(t)
		defer cleanup()

		docsDir := createTestDocsDir(t, tmpDir)
		cfg := &enginedocs.Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		stdout, stderr := captureOutput(t, func() {
			exitCode := handleDocsCommand(cfg, []string{"get", "nonexistent"})
			assert.Equal(t, 1, exitCode)
		})

		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Error getting doc")
		assert.Contains(t, stderr, "document not found")
	})

	t.Run("get without name shows usage", func(t *testing.T) {
		tmpDir, cleanup := setupTestWithCwd(t)
		defer cleanup()

		docsDir := createTestDocsDir(t, tmpDir)
		cfg := &enginedocs.Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		stdout, stderr := captureOutput(t, func() {
			exitCode := handleDocsCommand(cfg, []string{"get"})
			assert.Equal(t, 1, exitCode)
		})

		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Usage:")
	})
}

func TestHandleDocsCommand_Validate(t *testing.T) {
	// Cannot be parallel because changes working directory

	t.Run("validate passes for valid docs", func(t *testing.T) {
		tmpDir, cleanup := setupTestWithCwd(t)
		defer cleanup()

		docsDir := createTestDocsDir(t, tmpDir)
		cfg := &enginedocs.Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		stdout, stderr := captureOutput(t, func() {
			exitCode := handleDocsCommand(cfg, []string{"validate"})
			assert.Equal(t, 0, exitCode)
		})

		assert.Contains(t, stdout, "Validation passed")
		assert.Empty(t, stderr)
	})

	t.Run("validate fails for invalid docs", func(t *testing.T) {
		tmpDir, cleanup := setupTestWithCwd(t)
		defer cleanup()

		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))

		// Create invalid list.yaml with wrong version
		listYAML := `version: "2.0"
engine: wrong-engine
baseURL: https://example.com
docs: []
`
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644))

		cfg := &enginedocs.Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		stdout, stderr := captureOutput(t, func() {
			exitCode := handleDocsCommand(cfg, []string{"validate"})
			assert.Equal(t, 1, exitCode)
		})

		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Validation failed")
		assert.Contains(t, stderr, "version")
		assert.Contains(t, stderr, "engine")
	})

	t.Run("validate reports missing required docs", func(t *testing.T) {
		tmpDir, cleanup := setupTestWithCwd(t)
		defer cleanup()

		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))

		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs: []
`
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644))

		cfg := &enginedocs.Config{
			EngineName:   "test-engine",
			LocalDir:     docsDir,
			RequiredDocs: []string{"usage", "schema"},
		}

		stdout, stderr := captureOutput(t, func() {
			exitCode := handleDocsCommand(cfg, []string{"validate"})
			assert.Equal(t, 1, exitCode)
		})

		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Validation failed")
		assert.Contains(t, stderr, "usage")
		assert.Contains(t, stderr, "schema")
	})
}

func TestHandleDocsCommand_Usage(t *testing.T) {
	t.Parallel()

	t.Run("unknown subcommand shows usage", func(t *testing.T) {
		t.Parallel()

		cfg := &enginedocs.Config{
			EngineName: "test-engine",
			LocalDir:   "/some/path",
		}

		stdout, stderr := captureOutput(t, func() {
			exitCode := handleDocsCommand(cfg, []string{"unknown"})
			assert.Equal(t, 1, exitCode)
		})

		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Usage:")
		assert.Contains(t, stderr, "list")
		assert.Contains(t, stderr, "get <name>")
		assert.Contains(t, stderr, "validate")
	})
}

func TestConfig_DocsConfig(t *testing.T) {
	t.Parallel()

	t.Run("DocsConfig field exists and can be set", func(t *testing.T) {
		t.Parallel()

		docsConfig := &enginedocs.Config{
			EngineName:   "test-engine",
			LocalDir:     "cmd/test-engine/docs",
			BaseURL:      "https://example.com",
			RequiredDocs: []string{"usage"},
		}

		cfg := Config{
			Name:       "test-engine",
			DocsConfig: docsConfig,
		}

		assert.NotNil(t, cfg.DocsConfig)
		assert.Equal(t, "test-engine", cfg.DocsConfig.EngineName)
		assert.Equal(t, "cmd/test-engine/docs", cfg.DocsConfig.LocalDir)
	})

	t.Run("DocsConfig can be nil", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			Name:       "test-engine",
			DocsConfig: nil,
		}

		assert.Nil(t, cfg.DocsConfig)
	})
}
