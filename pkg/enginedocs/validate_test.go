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

package enginedocs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestDocStore is a helper to create a valid list.yaml for testing
func createTestDocStore(t *testing.T, dir string, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "list.yaml"), []byte(content), 0o644))
}

// setupTestWithCwd creates a temp directory, changes to it, and returns cleanup function
// This allows tests to use relative paths
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

func TestValidate_Rule1_ListYAMLExists(t *testing.T) {
	t.Parallel()

	t.Run("error when list.yaml does not exist", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   "/non/existent/path",
		}

		errs := Validate(cfg)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "list.yaml: file not found")
	})

	t.Run("error when list.yaml is invalid YAML", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte("invalid: yaml: content:"), 0o644))

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		errs := Validate(cfg)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "list.yaml: invalid YAML")
	})
}

func TestValidate_Rule2_Version(t *testing.T) {
	t.Parallel()

	t.Run("error when version is not 1.0", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		listYAML := `version: "2.0"
engine: test-engine
baseURL: https://example.com
docs: []
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		errs := Validate(cfg)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "version: expected \"1.0\", got \"2.0\"")
	})

	t.Run("error when version is empty", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		listYAML := `version: ""
engine: test-engine
baseURL: https://example.com
docs: []
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		errs := Validate(cfg)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "version: expected \"1.0\"")
	})

	t.Run("success when version is 1.0", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs: []
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		errs := Validate(cfg)
		assert.Empty(t, errs)
	})
}

func TestValidate_Rule3_EngineName(t *testing.T) {
	t.Parallel()

	t.Run("error when engine does not match config", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		listYAML := `version: "1.0"
engine: wrong-engine
baseURL: https://example.com
docs: []
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		errs := Validate(cfg)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "engine: expected \"test-engine\", got \"wrong-engine\"")
	})

	t.Run("error when engine is empty", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		listYAML := `version: "1.0"
engine: ""
baseURL: https://example.com
docs: []
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		errs := Validate(cfg)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "engine: expected \"test-engine\", got \"\"")
	})
}

func TestValidate_Rule4_RequiredDocFields(t *testing.T) {
	// NOTE: These tests cannot be parallel because they change working directory

	t.Run("error when name is empty", func(t *testing.T) {
		tmpDir, cleanup := setupTestWithCwd(t)
		defer cleanup()

		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "test.md"), []byte("content"), 0o644))

		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: ""
    title: Test Title
    description: Test Description
    url: docs/test.md
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		errs := Validate(cfg)
		require.GreaterOrEqual(t, len(errs), 1)
		hasNameError := false
		for _, err := range errs {
			if assert.Contains(t, err.Error(), ".name: field is required but empty") {
				hasNameError = true
				break
			}
		}
		assert.True(t, hasNameError, "expected error about empty name field")
	})

	t.Run("error when title is empty", func(t *testing.T) {
		tmpDir, cleanup := setupTestWithCwd(t)
		defer cleanup()

		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "test.md"), []byte("content"), 0o644))

		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: test
    title: ""
    description: Test Description
    url: docs/test.md
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		errs := Validate(cfg)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), ".title: field is required but empty")
	})

	t.Run("error when description is empty", func(t *testing.T) {
		tmpDir, cleanup := setupTestWithCwd(t)
		defer cleanup()

		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "test.md"), []byte("content"), 0o644))

		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: test
    title: Test Title
    description: ""
    url: docs/test.md
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		errs := Validate(cfg)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), ".description: field is required but empty")
	})

	t.Run("error when url is empty", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: test
    title: Test Title
    description: Test Description
    url: ""
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		errs := Validate(cfg)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), ".url: field is required but empty")
	})

	t.Run("multiple empty fields reported", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: ""
    title: ""
    description: ""
    url: ""
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		errs := Validate(cfg)
		assert.Len(t, errs, 4)
	})
}

func TestValidate_Rule5_URLPointsToExistingFile(t *testing.T) {
	// Cannot be parallel because some subtests change working directory

	t.Run("error when file does not exist", func(t *testing.T) {
		// This subtest is safe to run but must wait for cwd-changing subtests

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: test
    title: Test Title
    description: Test Description
    url: non/existent/file.md
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		errs := Validate(cfg)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), ".url: file not found")
	})

	t.Run("success when file exists", func(t *testing.T) {
		// Cannot be parallel because changes working directory
		tmpDir, cleanup := setupTestWithCwd(t)
		defer cleanup()

		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "test.md"), []byte("content"), 0o644))

		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: test
    title: Test Title
    description: Test Description
    url: docs/test.md
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		errs := Validate(cfg)
		assert.Empty(t, errs)
	})
}

func TestValidate_Rule6_RequiredDocs(t *testing.T) {
	// Cannot be parallel because some subtests change working directory

	t.Run("error when required doc is missing", func(t *testing.T) {
		// This subtest is safe to run but must wait for cwd-changing subtests

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs: []
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName:   "test-engine",
			LocalDir:     docsDir,
			RequiredDocs: []string{"usage", "schema"},
		}

		errs := Validate(cfg)
		require.Len(t, errs, 2)
		assert.Contains(t, errs[0].Error(), "required doc \"usage\" is missing")
		assert.Contains(t, errs[1].Error(), "required doc \"schema\" is missing")
	})

	t.Run("success when all required docs present", func(t *testing.T) {
		// Cannot be parallel because changes working directory
		tmpDir, cleanup := setupTestWithCwd(t)
		defer cleanup()

		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "usage.md"), []byte("usage"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "schema.md"), []byte("schema"), 0o644))

		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: usage
    title: Usage
    description: Usage guide
    url: docs/usage.md
  - name: schema
    title: Schema
    description: Schema docs
    url: docs/schema.md
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName:   "test-engine",
			LocalDir:     docsDir,
			RequiredDocs: []string{"usage", "schema"},
		}

		errs := Validate(cfg)
		assert.Empty(t, errs)
	})

	t.Run("partial required docs reports only missing", func(t *testing.T) {
		// Cannot be parallel because changes working directory
		tmpDir, cleanup := setupTestWithCwd(t)
		defer cleanup()

		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "usage.md"), []byte("usage"), 0o644))

		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: usage
    title: Usage
    description: Usage guide
    url: docs/usage.md
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName:   "test-engine",
			LocalDir:     docsDir,
			RequiredDocs: []string{"usage", "schema"},
		}

		errs := Validate(cfg)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "required doc \"schema\" is missing")
	})
}

func TestValidate_Rule7_NoDuplicateNames(t *testing.T) {
	// Cannot be parallel because changes working directory

	t.Run("error when duplicate names exist", func(t *testing.T) {
		tmpDir, cleanup := setupTestWithCwd(t)
		defer cleanup()

		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "doc1.md"), []byte("doc1"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "doc2.md"), []byte("doc2"), 0o644))

		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: duplicate
    title: First Doc
    description: First doc with name
    url: docs/doc1.md
  - name: duplicate
    title: Second Doc
    description: Second doc with same name
    url: docs/doc2.md
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		errs := Validate(cfg)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "duplicate name \"duplicate\"")
	})

	t.Run("success with unique names", func(t *testing.T) {
		tmpDir, cleanup := setupTestWithCwd(t)
		defer cleanup()

		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "doc1.md"), []byte("doc1"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "doc2.md"), []byte("doc2"), 0o644))

		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: unique1
    title: First Doc
    description: First doc
    url: docs/doc1.md
  - name: unique2
    title: Second Doc
    description: Second doc
    url: docs/doc2.md
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		errs := Validate(cfg)
		assert.Empty(t, errs)
	})
}

func TestValidate_Rule8_URLMustBeRelative(t *testing.T) {
	// Cannot be parallel because some subtests change working directory

	t.Run("error when URL starts with http://", func(t *testing.T) {
		// This subtest is safe to run but must wait for cwd-changing subtests

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: test
    title: Test
    description: Test doc
    url: http://example.com/docs/test.md
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		errs := Validate(cfg)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "must be a relative path, not an absolute URL")
	})

	t.Run("error when URL starts with https://", func(t *testing.T) {
		// This subtest is safe to run but must wait for cwd-changing subtests

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: test
    title: Test
    description: Test doc
    url: https://example.com/docs/test.md
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		errs := Validate(cfg)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "must be a relative path, not an absolute URL")
	})

	t.Run("error when URL starts with /", func(t *testing.T) {
		// This subtest is safe to run but must wait for cwd-changing subtests

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: test
    title: Test
    description: Test doc
    url: /absolute/path/docs/test.md
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		errs := Validate(cfg)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "must be a relative path, not an absolute path starting with /")
	})

	t.Run("success with relative path", func(t *testing.T) {
		// Cannot be parallel because changes working directory
		tmpDir, cleanup := setupTestWithCwd(t)
		defer cleanup()

		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "test.md"), []byte("content"), 0o644))

		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: test
    title: Test
    description: Test doc
    url: docs/test.md
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		errs := Validate(cfg)
		assert.Empty(t, errs)
	})
}

func TestValidate_AllRulesPass(t *testing.T) {
	// Cannot be parallel because changes working directory

	t.Run("valid complete store returns no errors", func(t *testing.T) {
		tmpDir, cleanup := setupTestWithCwd(t)
		defer cleanup()

		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "usage.md"), []byte("# Usage\n\nUsage guide."), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "schema.md"), []byte("# Schema\n\nSchema docs."), 0o644))

		listYAML := `version: "1.0"
engine: my-engine
baseURL: https://raw.githubusercontent.com/alexandremahdhaoui/forge/refs/heads/main
docs:
  - name: usage
    title: Usage Guide
    description: How to use my-engine
    url: docs/usage.md
    tags:
      - guide
      - tutorial
    required: true
  - name: schema
    title: Configuration Schema
    description: JSON schema for my-engine
    url: docs/schema.md
    required: true
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName:   "my-engine",
			LocalDir:     docsDir,
			BaseURL:      "https://raw.githubusercontent.com/alexandremahdhaoui/forge/refs/heads/main",
			RequiredDocs: []string{"usage", "schema"},
		}

		errs := Validate(cfg)
		assert.Empty(t, errs, "expected no validation errors for valid store")
	})
}

func TestValidate_MultipleErrors(t *testing.T) {
	t.Parallel()

	t.Run("returns all errors not just the first one", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		listYAML := `version: "2.0"
engine: wrong-engine
baseURL: https://example.com
docs:
  - name: ""
    title: ""
    description: ""
    url: ""
`
		createTestDocStore(t, docsDir, listYAML)

		cfg := Config{
			EngineName:   "correct-engine",
			LocalDir:     docsDir,
			RequiredDocs: []string{"required-doc"},
		}

		errs := Validate(cfg)
		// Should have errors for: version, engine, name, title, description, url, required doc
		assert.GreaterOrEqual(t, len(errs), 6, "expected at least 6 validation errors")
	})
}
