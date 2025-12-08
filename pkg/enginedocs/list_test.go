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

func TestDocsList_Success(t *testing.T) {
	t.Parallel()

	t.Run("returns docs from local file", func(t *testing.T) {
		t.Parallel()

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
  - name: schema
    title: Schema
    description: Configuration schema
    url: docs/schema.md
`
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644))

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		docs, err := DocsList(cfg)
		require.NoError(t, err)
		require.Len(t, docs, 2)

		assert.Equal(t, "usage", docs[0].Name)
		assert.Equal(t, "Usage Guide", docs[0].Title)
		assert.Equal(t, "schema", docs[1].Name)
		assert.Equal(t, "Schema", docs[1].Title)
	})

	t.Run("returns empty slice for empty docs array", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))

		listYAML := `version: "1.0"
engine: empty-engine
baseURL: https://example.com
docs: []
`
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644))

		cfg := Config{
			EngineName: "empty-engine",
			LocalDir:   docsDir,
		}

		docs, err := DocsList(cfg)
		require.NoError(t, err)
		assert.Empty(t, docs)
	})
}

func TestDocsList_Error(t *testing.T) {
	t.Parallel()

	t.Run("returns error when FetchDocStore fails", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   "/non/existent/path",
			BaseURL:    "", // No fallback
		}

		docs, err := DocsList(cfg)
		assert.Error(t, err)
		assert.Nil(t, docs)
	})

	t.Run("propagates FetchDocStore error message", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   "/non/existent/path",
			BaseURL:    "",
		}

		_, err := DocsList(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no BaseURL configured")
	})
}

func TestDocsList_WithTags(t *testing.T) {
	t.Parallel()

	t.Run("returns docs with tags preserved", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))

		listYAML := `version: "1.0"
engine: tagged-engine
baseURL: https://example.com
docs:
  - name: guide
    title: Guide
    description: A guide
    url: docs/guide.md
    tags:
      - tutorial
      - beginner
    required: true
`
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644))

		cfg := Config{
			EngineName: "tagged-engine",
			LocalDir:   docsDir,
		}

		docs, err := DocsList(cfg)
		require.NoError(t, err)
		require.Len(t, docs, 1)

		assert.Equal(t, []string{"tutorial", "beginner"}, docs[0].Tags)
		assert.True(t, docs[0].Required)
	})
}
