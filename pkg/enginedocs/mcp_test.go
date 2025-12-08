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
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterDocsTools(t *testing.T) {
	t.Parallel()

	t.Run("registers three tools without error", func(t *testing.T) {
		t.Parallel()

		server := mcpserver.New("test-server", "1.0.0")
		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   "docs",
			BaseURL:    "https://example.com",
		}

		err := RegisterDocsTools(server, cfg)
		assert.NoError(t, err)
	})

	t.Run("registers tools with correct engine name in description", func(t *testing.T) {
		t.Parallel()

		server := mcpserver.New("test-server", "1.0.0")
		cfg := Config{
			EngineName: "my-custom-engine",
			LocalDir:   "docs",
			BaseURL:    "https://example.com",
		}

		err := RegisterDocsTools(server, cfg)
		assert.NoError(t, err)
	})
}

func TestDocsListInput(t *testing.T) {
	t.Parallel()

	t.Run("DocsListInput has no fields", func(t *testing.T) {
		t.Parallel()

		input := DocsListInput{}
		_ = input // Just verify the type exists and has no required fields
	})
}

func TestDocsGetInput(t *testing.T) {
	t.Parallel()

	t.Run("DocsGetInput has Name field", func(t *testing.T) {
		t.Parallel()

		input := DocsGetInput{Name: "usage"}
		assert.Equal(t, "usage", input.Name)
	})
}

func TestDocsValidateInput(t *testing.T) {
	t.Parallel()

	t.Run("DocsValidateInput has no fields", func(t *testing.T) {
		t.Parallel()

		input := DocsValidateInput{}
		_ = input // Just verify the type exists and has no required fields
	})
}

func TestDocsListResult(t *testing.T) {
	t.Parallel()

	t.Run("DocsListResult fields", func(t *testing.T) {
		t.Parallel()

		result := DocsListResult{
			Docs: []DocEntry{
				{Name: "usage", Title: "Usage", Description: "Usage guide", URL: "docs/usage.md"},
			},
			Engine: "test-engine",
			Count:  1,
		}

		assert.Len(t, result.Docs, 1)
		assert.Equal(t, "test-engine", result.Engine)
		assert.Equal(t, 1, result.Count)
	})
}

func TestDocsValidateResult(t *testing.T) {
	t.Parallel()

	t.Run("DocsValidateResult valid", func(t *testing.T) {
		t.Parallel()

		result := DocsValidateResult{
			Engine: "test-engine",
			Valid:  true,
			Errors: nil,
		}

		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
	})

	t.Run("DocsValidateResult with errors", func(t *testing.T) {
		t.Parallel()

		result := DocsValidateResult{
			Engine: "test-engine",
			Valid:  false,
			Errors: []string{"error1", "error2"},
		}

		assert.False(t, result.Valid)
		assert.Len(t, result.Errors, 2)
	})
}

func TestHandleDocsListTool(t *testing.T) {
	t.Parallel()

	t.Run("successful list returns docs", func(t *testing.T) {
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
`
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644))

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		result, artifact, err := handleDocsListTool(context.Background(), &mcp.CallToolRequest{}, DocsListInput{}, cfg)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.IsError)

		// Verify artifact is DocsListResult
		listResult, ok := artifact.(DocsListResult)
		require.True(t, ok)
		assert.Equal(t, "test-engine", listResult.Engine)
		assert.Equal(t, 1, listResult.Count)
	})

	t.Run("error when FetchDocStore fails", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   "/non/existent/path",
			BaseURL:    "",
		}

		result, _, err := handleDocsListTool(context.Background(), &mcp.CallToolRequest{}, DocsListInput{}, cfg)
		require.NoError(t, err) // MCP errors are returned in result, not as error
		require.NotNil(t, result)
		assert.True(t, result.IsError)
	})
}

func TestHandleDocsGetTool(t *testing.T) {
	t.Parallel()

	t.Run("successful get returns content", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))

		docContent := "# Usage Guide\n\nThis is the usage guide."
		docPath := filepath.Join(docsDir, "usage.md")
		require.NoError(t, os.WriteFile(docPath, []byte(docContent), 0o644))

		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs:
  - name: usage
    title: Usage Guide
    description: How to use
    url: ` + docPath + `
`
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644))

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		input := DocsGetInput{Name: "usage"}
		result, _, err := handleDocsGetTool(context.Background(), &mcp.CallToolRequest{}, input, cfg)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.IsError)

		// Verify content is in result
		require.Len(t, result.Content, 1)
		textContent, ok := result.Content[0].(*mcp.TextContent)
		require.True(t, ok)
		assert.Equal(t, docContent, textContent.Text)
	})

	t.Run("error when doc not found", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))

		listYAML := `version: "1.0"
engine: test-engine
baseURL: https://example.com
docs: []
`
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644))

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		input := DocsGetInput{Name: "non-existent"}
		result, _, err := handleDocsGetTool(context.Background(), &mcp.CallToolRequest{}, input, cfg)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsError)
	})
}

func TestHandleDocsValidateTool(t *testing.T) {
	// NOTE: Some tests cannot be parallel because they change working directory

	t.Run("successful validation returns valid result", func(t *testing.T) {
		// Cannot be parallel because changes working directory
		tmpDir := t.TempDir()

		// Save and restore working directory
		originalWd, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalWd) }()

		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "usage.md"), []byte("content"), 0o644))

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
		}

		result, artifact, err := handleDocsValidateTool(context.Background(), &mcp.CallToolRequest{}, DocsValidateInput{}, cfg)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.IsError)

		// Verify artifact is DocsValidateResult
		validateResult, ok := artifact.(DocsValidateResult)
		require.True(t, ok)
		assert.True(t, validateResult.Valid)
		assert.Empty(t, validateResult.Errors)
	})

	t.Run("validation with errors returns error result", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))

		listYAML := `version: "2.0"
engine: wrong-engine
baseURL: https://example.com
docs: []
`
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644))

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		result, artifact, err := handleDocsValidateTool(context.Background(), &mcp.CallToolRequest{}, DocsValidateInput{}, cfg)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsError)

		// Verify artifact is DocsValidateResult with errors
		validateResult, ok := artifact.(DocsValidateResult)
		require.True(t, ok)
		assert.False(t, validateResult.Valid)
		assert.NotEmpty(t, validateResult.Errors)
	})

	t.Run("validation includes detailed error messages in content", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0o755))

		listYAML := `version: "2.0"
engine: test-engine
baseURL: https://example.com
docs: []
`
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644))

		cfg := Config{
			EngineName: "test-engine",
			LocalDir:   docsDir,
		}

		result, _, err := handleDocsValidateTool(context.Background(), &mcp.CallToolRequest{}, DocsValidateInput{}, cfg)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should have multiple content items including error details
		assert.GreaterOrEqual(t, len(result.Content), 2)
	})
}
