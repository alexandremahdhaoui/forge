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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestDocStore_YAMLMarshaling(t *testing.T) {
	t.Parallel()

	t.Run("marshal and unmarshal DocStore", func(t *testing.T) {
		t.Parallel()

		original := DocStore{
			Version: "1.0",
			Engine:  "go-build",
			BaseURL: "https://raw.githubusercontent.com/alexandremahdhaoui/forge/refs/heads/main",
			Docs: []DocEntry{
				{
					Name:        "usage",
					Title:       "Usage Guide",
					Description: "How to use the go-build engine",
					URL:         "cmd/go-build/docs/usage.md",
					Tags:        []string{"guide", "usage"},
					Required:    true,
				},
				{
					Name:        "schema",
					Title:       "Configuration Schema",
					Description: "JSON schema for go-build configuration",
					URL:         "cmd/go-build/docs/schema.md",
					Tags:        nil,
					Required:    false,
				},
			},
		}

		// Marshal to YAML
		data, err := yaml.Marshal(original)
		require.NoError(t, err)

		// Unmarshal back
		var unmarshaled DocStore
		err = yaml.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		// Compare
		assert.Equal(t, original.Version, unmarshaled.Version)
		assert.Equal(t, original.Engine, unmarshaled.Engine)
		assert.Equal(t, original.BaseURL, unmarshaled.BaseURL)
		assert.Len(t, unmarshaled.Docs, 2)
		assert.Equal(t, original.Docs[0].Name, unmarshaled.Docs[0].Name)
		assert.Equal(t, original.Docs[0].Tags, unmarshaled.Docs[0].Tags)
		assert.Equal(t, original.Docs[0].Required, unmarshaled.Docs[0].Required)
	})

	t.Run("empty docs array", func(t *testing.T) {
		t.Parallel()

		original := DocStore{
			Version: "1.0",
			Engine:  "test-engine",
			BaseURL: "https://example.com",
			Docs:    []DocEntry{},
		}

		data, err := yaml.Marshal(original)
		require.NoError(t, err)

		var unmarshaled DocStore
		err = yaml.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Empty(t, unmarshaled.Docs)
	})
}

func TestDocStore_JSONMarshaling(t *testing.T) {
	t.Parallel()

	t.Run("marshal and unmarshal DocStore", func(t *testing.T) {
		t.Parallel()

		original := DocStore{
			Version: "1.0",
			Engine:  "testenv",
			BaseURL: "https://example.com",
			Docs: []DocEntry{
				{
					Name:        "config",
					Title:       "Configuration",
					Description: "Configuration options",
					URL:         "cmd/testenv/docs/config.md",
					Tags:        []string{"config"},
					Required:    true,
				},
			},
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)

		// Unmarshal back
		var unmarshaled DocStore
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		// Compare
		assert.Equal(t, original.Version, unmarshaled.Version)
		assert.Equal(t, original.Engine, unmarshaled.Engine)
		assert.Equal(t, original.BaseURL, unmarshaled.BaseURL)
		assert.Len(t, unmarshaled.Docs, 1)
		assert.Equal(t, original.Docs[0], unmarshaled.Docs[0])
	})
}

func TestDocEntry_YAMLMarshaling(t *testing.T) {
	t.Parallel()

	t.Run("omit empty optional fields", func(t *testing.T) {
		t.Parallel()

		entry := DocEntry{
			Name:        "minimal",
			Title:       "Minimal Entry",
			Description: "A minimal doc entry",
			URL:         "docs/minimal.md",
			// Tags and Required omitted
		}

		data, err := yaml.Marshal(entry)
		require.NoError(t, err)

		// Should not contain "tags" or "required" in output
		assert.NotContains(t, string(data), "tags:")
		assert.NotContains(t, string(data), "required:")
	})

	t.Run("include non-empty optional fields", func(t *testing.T) {
		t.Parallel()

		entry := DocEntry{
			Name:        "full",
			Title:       "Full Entry",
			Description: "A full doc entry",
			URL:         "docs/full.md",
			Tags:        []string{"tag1", "tag2"},
			Required:    true,
		}

		data, err := yaml.Marshal(entry)
		require.NoError(t, err)

		// Should contain "tags" and "required"
		assert.Contains(t, string(data), "tags:")
		assert.Contains(t, string(data), "required:")
	})
}

func TestDocEntry_JSONMarshaling(t *testing.T) {
	t.Parallel()

	t.Run("omit empty optional fields", func(t *testing.T) {
		t.Parallel()

		entry := DocEntry{
			Name:        "minimal",
			Title:       "Minimal Entry",
			Description: "A minimal doc entry",
			URL:         "docs/minimal.md",
		}

		data, err := json.Marshal(entry)
		require.NoError(t, err)

		// Should not contain "tags" in output (omitempty)
		assert.NotContains(t, string(data), `"tags"`)
		// Required defaults to false, but without omitempty it will be present
	})
}

func TestConfig_YAMLMarshaling(t *testing.T) {
	t.Parallel()

	t.Run("marshal and unmarshal Config", func(t *testing.T) {
		t.Parallel()

		original := Config{
			EngineName:   "go-build",
			LocalDir:     "cmd/go-build/docs",
			BaseURL:      "https://raw.githubusercontent.com/alexandremahdhaoui/forge/refs/heads/main",
			RequiredDocs: []string{"usage", "schema"},
		}

		data, err := yaml.Marshal(original)
		require.NoError(t, err)

		var unmarshaled Config
		err = yaml.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, original.EngineName, unmarshaled.EngineName)
		assert.Equal(t, original.LocalDir, unmarshaled.LocalDir)
		assert.Equal(t, original.BaseURL, unmarshaled.BaseURL)
		assert.Equal(t, original.RequiredDocs, unmarshaled.RequiredDocs)
	})
}

func TestConfig_JSONMarshaling(t *testing.T) {
	t.Parallel()

	t.Run("marshal and unmarshal Config", func(t *testing.T) {
		t.Parallel()

		original := Config{
			EngineName:   "testenv",
			LocalDir:     "cmd/testenv/docs",
			BaseURL:      "https://example.com",
			RequiredDocs: []string{"config"},
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var unmarshaled Config
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, original, unmarshaled)
	})
}
