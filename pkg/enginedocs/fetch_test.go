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

func TestFetchDocStore_LocalFile(t *testing.T) {
	t.Parallel()

	t.Run("successful local file read", func(t *testing.T) {
		t.Parallel()

		// Create temp directory with list.yaml
		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))

		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: usage
    title: Usage Guide
    description: How to use
    url: docs/usage.md
`
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644))

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
			BaseURL:    "https://example.com",
		}

		store, err := FetchDocStore(cfg)
		require.NoError(t, err)
		require.NotNil(t, store)

		assert.Equal(t, "1.0", store.Version)
		assert.Equal(t, "test-engine", store.Engine)
		assert.Equal(t, "https://example.com", store.BaseURL)
		assert.Len(t, store.Docs, 1)
		assert.Equal(t, "usage", store.Docs[0].Name)
	})

	t.Run("invalid YAML in local file", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))

		// Write invalid YAML
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte("invalid: yaml: content:"), 0o644))

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		_, err := FetchDocStore(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse local docs list")
	})

	t.Run("local file read error (not exists without baseURL)", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   "/non/existent/path",
			BaseURL:    "", // No BaseURL, so no fallback
		}

		_, err := FetchDocStore(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no BaseURL configured")
	})
}

func TestFetchDocStore_RemoteFallback(t *testing.T) {
	t.Parallel()

	t.Run("successful remote fallback when local file missing", func(t *testing.T) {
		t.Parallel()

		remoteYAML := `version: "1.0"
engine: remote-engine
baseURL: https://example.com
docs:
  - name: remote-doc
    title: Remote Doc
    description: From remote
    url: docs/remote.md
`

		// Create mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/non-existent-local/list.yaml", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(remoteYAML))
		}))
		defer server.Close()

		cfg := Config{
			EngineName: "remote-engine",
			LocalDir:   "non-existent-local",
			BaseURL:    server.URL,
		}

		store, err := FetchDocStore(cfg)
		require.NoError(t, err)
		require.NotNil(t, store)

		assert.Equal(t, "1.0", store.Version)
		assert.Equal(t, "remote-engine", store.Engine)
		assert.Len(t, store.Docs, 1)
		assert.Equal(t, "remote-doc", store.Docs[0].Name)
	})

	t.Run("remote fetch returns HTTP error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   "non-existent-local",
			BaseURL:    server.URL,
		}

		_, err := FetchDocStore(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch remote docs list")
	})

	t.Run("remote fetch returns invalid YAML", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("invalid: yaml: content:"))
		}))
		defer server.Close()

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   "non-existent-local",
			BaseURL:    server.URL,
		}

		_, err := FetchDocStore(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse remote docs list")
	})
}

func TestFetchDocStore_LocalFilePreferred(t *testing.T) {
	t.Parallel()

	t.Run("local file preferred over remote", func(t *testing.T) {
		t.Parallel()

		// Create temp directory with local list.yaml
		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))

		localYAML := `version: "1.0"
engine: local-engine
baseURL: https://example.com
docs:
  - name: local-doc
    title: Local Doc
    description: From local
    url: docs/local.md
`
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(localYAML), 0o644))

		// Create mock HTTP server that should NOT be called
		serverCalled := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serverCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`version: "1.0"`))
		}))
		defer server.Close()

		cfg := Config{
			EngineName: "local-engine",
			LocalDir:   docsDir,
			BaseURL:    server.URL,
		}

		store, err := FetchDocStore(cfg)
		require.NoError(t, err)
		require.NotNil(t, store)

		// Verify local file was used
		assert.Equal(t, "local-engine", store.Engine)
		assert.Equal(t, "local-doc", store.Docs[0].Name)

		// Verify remote was NOT called
		assert.False(t, serverCalled, "remote server should not be called when local file exists")
	})
}

func TestFetchURL(t *testing.T) {
	t.Parallel()

	t.Run("successful fetch", func(t *testing.T) {
		t.Parallel()

		expectedContent := "Hello, World!"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(expectedContent))
		}))
		defer server.Close()

		content, err := fetchURL(server.URL)
		require.NoError(t, err)
		assert.Equal(t, expectedContent, content)
	})

	t.Run("HTTP error status", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		_, err := fetchURL(server.URL)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP 500")
	})

	t.Run("connection error", func(t *testing.T) {
		t.Parallel()

		_, err := fetchURL("http://localhost:99999/invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP request failed")
	})

	t.Run("HTTP 404 status", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		_, err := fetchURL(server.URL)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP 404")
	})
}
