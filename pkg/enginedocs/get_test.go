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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocsGet_LocalFile(t *testing.T) {
	t.Parallel()

	t.Run("successful local doc retrieval", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))

		// Create list.yaml with a doc pointing to a local file
		docFilePath := filepath.Join(docsDir, "usage.md")
		docContent := "# Usage Guide\n\nThis is the usage guide."

		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: usage
    title: Usage Guide
    description: How to use
    url: ` + docFilePath + `
`
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644))
		require.NoError(t, os.WriteFile(docFilePath, []byte(docContent), 0o644))

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		content, err := DocsGet(cfg, "usage")
		require.NoError(t, err)
		assert.Equal(t, docContent, content)
	})

	t.Run("document not found in store", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))

		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: existing
    title: Existing Doc
    description: An existing doc
    url: docs/existing.md
`
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644))

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		_, err := DocsGet(cfg, "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "document not found: non-existent")
		assert.Contains(t, err.Error(), "docs list")
	})
}

func TestDocsGet_RemoteFallback(t *testing.T) {
	t.Parallel()

	t.Run("successful remote doc retrieval when local file missing", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))

		remoteDocContent := "# Remote Usage Guide\n\nFrom remote server."

		// Create mock HTTP server for remote doc fetch
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/docs/remote-usage.md" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(remoteDocContent))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		// Create list.yaml pointing to a non-existent local file
		listYAML := `version: "1.0"
engine: test-engine
baseURL: ` + server.URL + `
docs:
  - name: usage
    title: Usage Guide
    description: How to use
    url: docs/remote-usage.md
`
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644))

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
			BaseURL:    server.URL,
		}

		content, err := DocsGet(cfg, "usage")
		require.NoError(t, err)
		assert.Equal(t, remoteDocContent, content)
	})

	t.Run("remote fetch fails with HTTP error", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		listYAML := `version: "1.0"
engine: test-engine
baseURL: ` + server.URL + `
docs:
  - name: usage
    title: Usage Guide
    description: How to use
    url: docs/missing.md
`
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644))

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
			BaseURL:    server.URL,
		}

		_, err := DocsGet(cfg, "usage")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch remote document")
	})

	t.Run("no BaseURL configured and local file missing", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))

		listYAML := `version: "1.0"
engine: test-engine
baseURL: ""
docs:
  - name: usage
    title: Usage Guide
    description: How to use
    url: docs/non-existent.md
`
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644))

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
			BaseURL:    "", // No BaseURL
		}

		_, err := DocsGet(cfg, "usage")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no BaseURL configured")
	})
}

func TestDocsGet_FetchDocStoreError(t *testing.T) {
	t.Parallel()

	t.Run("returns error when FetchDocStore fails", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   "/non/existent/path",
			BaseURL:    "",
		}

		_, err := DocsGet(cfg, "usage")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch docs list")
	})
}

func TestDocsGet_LocalFilePreferred(t *testing.T) {
	t.Parallel()

	t.Run("local file preferred over remote", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))

		localDocContent := "# Local Content\n\nThis is local."
		docFilePath := filepath.Join(docsDir, "usage.md")

		// Create mock HTTP server that should NOT be called for doc content
		docServerCalled := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/docs/list.yaml" {
				docServerCalled = true
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Remote content"))
		}))
		defer server.Close()

		listYAML := `version: "1.0"
engine: test-engine
baseURL: ` + server.URL + `
docs:
  - name: usage
    title: Usage Guide
    description: How to use
    url: ` + docFilePath + `
`
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644))
		require.NoError(t, os.WriteFile(docFilePath, []byte(localDocContent), 0o644))

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
			BaseURL:    server.URL,
		}

		content, err := DocsGet(cfg, "usage")
		require.NoError(t, err)

		// Verify local content was returned
		assert.Equal(t, localDocContent, content)

		// Verify remote was NOT called for doc
		assert.False(t, docServerCalled, "remote server should not be called for doc when local file exists")
	})
}

func TestDocsGet_MultipleDocuments(t *testing.T) {
	t.Parallel()

	t.Run("retrieve different documents by name", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))

		usageContent := "Usage content"
		schemaContent := "Schema content"
		usagePath := filepath.Join(docsDir, "usage.md")
		schemaPath := filepath.Join(docsDir, "schema.md")

		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: usage
    title: Usage Guide
    description: How to use
    url: ` + usagePath + `
  - name: schema
    title: Schema
    description: Configuration schema
    url: ` + schemaPath + `
`
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644))
		require.NoError(t, os.WriteFile(usagePath, []byte(usageContent), 0o644))
		require.NoError(t, os.WriteFile(schemaPath, []byte(schemaContent), 0o644))

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		content1, err := DocsGet(cfg, "usage")
		require.NoError(t, err)
		assert.Equal(t, usageContent, content1)

		content2, err := DocsGet(cfg, "schema")
		require.NoError(t, err)
		assert.Equal(t, schemaContent, content2)
	})
}
