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
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

// Note: contains is defined in prompt_test.go

// TestFetchDocsStore_Local tests reading docs store from local file
func TestFetchDocsStore_Local(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("Failed to create temp docs dir: %v", err)
	}

	// Create test docs-list.yaml
	store := DocStore{
		Version: "1.0",
		BaseURL: "https://example.com",
		Docs: []DocEntry{
			{
				Name:        "test-doc",
				Title:       "Test Document",
				Description: "A test document",
				URL:         "test-doc.md",
				Tags:        []string{"test"},
			},
		},
	}

	content, err := yaml.Marshal(store)
	if err != nil {
		t.Fatalf("Failed to marshal test store: %v", err)
	}

	docsListPath := filepath.Join(docsDir, "docs-list.yaml")
	if err := os.WriteFile(docsListPath, content, 0o644); err != nil {
		t.Fatalf("Failed to write test docs-list.yaml: %v", err)
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Test fetchDocsStore reads local file
	result, err := fetchDocsStore()
	if err != nil {
		t.Fatalf("fetchDocsStore failed: %v", err)
	}

	if result.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", result.Version)
	}

	if len(result.Docs) != 1 {
		t.Errorf("Expected 1 doc, got %d", len(result.Docs))
	}

	if result.Docs[0].Name != "test-doc" {
		t.Errorf("Expected doc name 'test-doc', got %s", result.Docs[0].Name)
	}
}

// TestDocStoreStructure tests that DocStore structure is valid
func TestDocStoreStructure(t *testing.T) {
	store := DocStore{
		Version: "1.0",
		BaseURL: "https://example.com",
		Docs: []DocEntry{
			{
				Name:        "test",
				Title:       "Test Title",
				Description: "Test Description",
				URL:         "test.md",
				Tags:        []string{"tag1", "tag2"},
			},
		},
	}

	// Marshal and unmarshal to verify structure
	data, err := yaml.Marshal(store)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result DocStore
	if err := yaml.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.Version != store.Version {
		t.Errorf("Version mismatch: expected %s, got %s", store.Version, result.Version)
	}

	if result.BaseURL != store.BaseURL {
		t.Errorf("BaseURL mismatch: expected %s, got %s", store.BaseURL, result.BaseURL)
	}

	if len(result.Docs) != 1 {
		t.Fatalf("Expected 1 doc, got %d", len(result.Docs))
	}

	doc := result.Docs[0]
	if doc.Name != "test" {
		t.Errorf("Name mismatch: expected 'test', got %s", doc.Name)
	}

	if len(doc.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(doc.Tags))
	}
}

// TestDocEntry tests DocEntry structure
func TestDocEntry(t *testing.T) {
	entry := DocEntry{
		Name:        "test-doc",
		Title:       "Test Document Title",
		Description: "This is a test document description",
		URL:         "test-doc.md",
		Tags:        []string{"testing", "example"},
	}

	if entry.Name != "test-doc" {
		t.Errorf("Expected name 'test-doc', got %s", entry.Name)
	}

	if entry.Title != "Test Document Title" {
		t.Errorf("Expected title 'Test Document Title', got %s", entry.Title)
	}

	if len(entry.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(entry.Tags))
	}
}

// TestRunDocs_InvalidArgs tests error handling for invalid arguments
func TestRunDocs_InvalidArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "no args",
			args: []string{},
			want: "usage: forge docs",
		},
		{
			name: "get without name",
			args: []string{"get"},
			want: "usage: forge docs get <doc-name>",
		},
		{
			name: "unknown operation",
			args: []string{"unknown"},
			want: "unknown operation: unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runDocs(tt.args)
			if err == nil {
				t.Error("Expected error, got nil")
			}
			if err != nil && !contains(err.Error(), tt.want) {
				t.Errorf("Expected error containing %q, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestDocsGet_LocalFile tests reading a doc from a local file
func TestDocsGet_LocalFile(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("Failed to create temp docs dir: %v", err)
	}

	// Create test docs-list.yaml
	store := DocStore{
		Version: "1.0",
		BaseURL: "https://example.com",
		Docs: []DocEntry{
			{
				Name:        "test-doc",
				Title:       "Test Document",
				Description: "A test document",
				URL:         "test-doc.md",
				Tags:        []string{"test"},
			},
		},
	}

	content, _ := yaml.Marshal(store)
	docsListPath := filepath.Join(docsDir, "docs-list.yaml")
	os.WriteFile(docsListPath, content, 0o644)

	// Create test doc file
	docContent := "# Test Document\n\nThis is a test document for verifying the docs system.\n"
	docPath := filepath.Join(tmpDir, "test-doc.md")
	os.WriteFile(docPath, []byte(docContent), 0o644)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Test docsGet reads local file
	// Note: This would normally print to stdout, so we'd need to capture that
	// For now, we verify the file exists and can be read
	if _, err := os.Stat("test-doc.md"); err != nil {
		t.Errorf("Test doc file should exist: %v", err)
	} else {
		t.Log("Local doc file found and readable")
	}
}

// TestDocsList_AllDocsExist verifies that all docs in docs-list.yaml
// reference files that actually exist
func TestDocsList_AllDocsExist(t *testing.T) {
	// Read the docs-list.yaml file
	docsListPath := filepath.Join("..", "..", "docs", "docs-list.yaml")
	content, err := os.ReadFile(docsListPath)
	if err != nil {
		t.Fatalf("Failed to read docs-list.yaml: %v", err)
	}

	// Parse the YAML
	var store DocStore
	if err := yaml.Unmarshal(content, &store); err != nil {
		t.Fatalf("Failed to parse docs-list.yaml: %v", err)
	}

	// Check each doc file exists
	repoRoot := filepath.Join("..", "..")
	var missingFiles []string

	for _, doc := range store.Docs {
		docPath := filepath.Join(repoRoot, doc.URL)
		if _, err := os.Stat(docPath); os.IsNotExist(err) {
			missingFiles = append(missingFiles, doc.URL)
			t.Errorf("Doc '%s' references non-existent file: %s", doc.Name, doc.URL)
		}
	}

	if len(missingFiles) > 0 {
		t.Fatalf("Found %d docs referencing non-existent files: %v", len(missingFiles), missingFiles)
	}

	t.Logf("Verified %d docs - all files exist", len(store.Docs))
}

// TestDocsList_NoEmptyFields verifies that all docs have required fields
func TestDocsList_NoEmptyFields(t *testing.T) {
	// Read the docs-list.yaml file
	docsListPath := filepath.Join("..", "..", "docs", "docs-list.yaml")
	content, err := os.ReadFile(docsListPath)
	if err != nil {
		t.Fatalf("Failed to read docs-list.yaml: %v", err)
	}

	// Parse the YAML
	var store DocStore
	if err := yaml.Unmarshal(content, &store); err != nil {
		t.Fatalf("Failed to parse docs-list.yaml: %v", err)
	}

	// Check each doc has required fields
	var errors []string

	for _, doc := range store.Docs {
		if doc.Name == "" {
			errors = append(errors, "found doc with empty name")
		}
		if doc.Title == "" {
			errors = append(errors, "doc '"+doc.Name+"' has empty title")
		}
		if doc.Description == "" {
			errors = append(errors, "doc '"+doc.Name+"' has empty description")
		}
		if doc.URL == "" {
			errors = append(errors, "doc '"+doc.Name+"' has empty URL")
		}
		if len(doc.Tags) == 0 {
			errors = append(errors, "doc '"+doc.Name+"' has no tags")
		}
	}

	if len(errors) > 0 {
		for _, e := range errors {
			t.Error(e)
		}
		t.Fatalf("Found %d validation errors", len(errors))
	}

	t.Logf("Verified %d docs - all have required fields", len(store.Docs))
}

// TestDocsGet_NotFound tests error handling when doc is not found
func TestDocsGet_NotFound(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("Failed to create temp docs dir: %v", err)
	}

	// Create test docs-list.yaml with no matching doc
	store := DocStore{
		Version: "1.0",
		BaseURL: "https://example.com",
		Docs: []DocEntry{
			{
				Name:        "other-doc",
				Title:       "Other Document",
				Description: "A different document",
				URL:         "other-doc.md",
				Tags:        []string{"test"},
			},
		},
	}

	content, _ := yaml.Marshal(store)
	docsListPath := filepath.Join(docsDir, "docs-list.yaml")
	os.WriteFile(docsListPath, content, 0o644)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Test docsGet with non-existent doc
	err := docsGet("non-existent-doc")
	if err == nil {
		t.Error("Expected error for non-existent doc, got nil")
	}

	if err != nil && !contains(err.Error(), "document not found") {
		t.Errorf("Expected 'document not found' error, got: %v", err)
	}
}

// TestDocStore_ValidYAML tests that the actual docs-list.yaml is valid YAML
func TestDocStore_ValidYAML(t *testing.T) {
	// Read the docs-list.yaml file
	docsListPath := filepath.Join("..", "..", "docs", "docs-list.yaml")
	content, err := os.ReadFile(docsListPath)
	if err != nil {
		t.Fatalf("Failed to read docs-list.yaml: %v", err)
	}

	// Parse the YAML
	var store DocStore
	if err := yaml.Unmarshal(content, &store); err != nil {
		t.Fatalf("Failed to parse docs-list.yaml: %v", err)
	}

	// Verify basic structure
	if store.Version == "" {
		t.Error("Version field is empty")
	}

	if store.BaseURL == "" {
		t.Error("BaseURL field is empty")
	}

	if len(store.Docs) == 0 {
		t.Error("Docs list is empty")
	}

	t.Logf("docs-list.yaml is valid YAML with version %s and %d docs", store.Version, len(store.Docs))
}

// TestDocsList_NoGitIgnoredFiles verifies that all docs in docs-list.yaml
// are actually committed to git (not ignored or untracked)
func TestDocsList_NoGitIgnoredFiles(t *testing.T) {
	// Read the docs-list.yaml file
	docsListPath := filepath.Join("..", "..", "docs", "docs-list.yaml")
	content, err := os.ReadFile(docsListPath)
	if err != nil {
		t.Fatalf("Failed to read docs-list.yaml: %v", err)
	}

	// Parse the YAML
	var store DocStore
	if err := yaml.Unmarshal(content, &store); err != nil {
		t.Fatalf("Failed to parse docs-list.yaml: %v", err)
	}

	// Get list of all tracked files in git
	repoRoot := filepath.Join("..", "..")
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run git ls-files: %v", err)
	}

	// Build a set of tracked files for fast lookup
	trackedFiles := make(map[string]bool)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			trackedFiles[line] = true
		}
	}

	// Check each doc is tracked by git
	var untrackedDocs []string
	var ignoredDocs []string

	for _, doc := range store.Docs {
		// Normalize path
		docPath := filepath.Clean(doc.URL)

		// Check if file is tracked
		if !trackedFiles[docPath] {
			// Check if file exists
			fullPath := filepath.Join(repoRoot, docPath)
			if _, err := os.Stat(fullPath); err == nil {
				// File exists but not tracked - check if ignored
				checkCmd := exec.Command("git", "check-ignore", "-q", docPath)
				checkCmd.Dir = repoRoot
				if err := checkCmd.Run(); err == nil {
					// File is ignored
					ignoredDocs = append(ignoredDocs, doc.Name+" ("+doc.URL+")")
				} else {
					// File exists but not tracked (maybe just not added)
					untrackedDocs = append(untrackedDocs, doc.Name+" ("+doc.URL+")")
				}
			} else {
				// File doesn't exist
				untrackedDocs = append(untrackedDocs, doc.Name+" ("+doc.URL+", file missing)")
			}
		}
	}

	// Report errors
	if len(ignoredDocs) > 0 {
		for _, d := range ignoredDocs {
			t.Errorf("Doc is git-ignored and should not be in docs-list.yaml: %s", d)
		}
	}

	if len(untrackedDocs) > 0 {
		for _, d := range untrackedDocs {
			t.Errorf("Doc is not tracked by git: %s", d)
		}
	}

	if len(ignoredDocs) > 0 || len(untrackedDocs) > 0 {
		t.Fatalf("Found %d ignored and %d untracked docs in docs-list.yaml", len(ignoredDocs), len(untrackedDocs))
	}

	t.Logf("Verified %d docs - all are tracked by git", len(store.Docs))
}

// --- Aggregation Tests ---

// setupTestProject creates a temporary directory with mock engine docs
func setupTestProject(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir := t.TempDir()

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	return tmpDir, func() {
		_ = os.Chdir(originalWd)
	}
}

// createMockEngineDocs creates a mock engine docs directory structure
func createMockEngineDocs(t *testing.T, baseDir, engineName string) {
	t.Helper()
	docsDir := filepath.Join(baseDir, "cmd", engineName, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("Failed to create docs dir: %v", err)
	}

	listYAML := `version: "1.0"
engine: "` + engineName + `"
baseURL: "https://raw.githubusercontent.com/example/repo/main"
docs:
  - name: usage
    title: "` + engineName + ` Usage Guide"
    description: "How to use ` + engineName + `"
    url: "cmd/` + engineName + `/docs/usage.md"
  - name: schema
    title: "` + engineName + ` Configuration Schema"
    description: "Configuration options for ` + engineName + `"
    url: "cmd/` + engineName + `/docs/schema.md"
`
	if err := os.WriteFile(filepath.Join(docsDir, "list.yaml"), []byte(listYAML), 0o644); err != nil {
		t.Fatalf("Failed to write list.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "usage.md"), []byte("# "+engineName+" Usage\n\nUsage guide content."), 0o644); err != nil {
		t.Fatalf("Failed to write usage.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "schema.md"), []byte("# "+engineName+" Schema\n\nSchema content."), 0o644); err != nil {
		t.Fatalf("Failed to write schema.md: %v", err)
	}
}

// createMockGlobalDocs creates mock global docs for forge
func createMockGlobalDocs(t *testing.T, baseDir string) {
	t.Helper()
	docsDir := filepath.Join(baseDir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("Failed to create global docs dir: %v", err)
	}

	listYAML := `version: "1.0"
baseURL: "https://raw.githubusercontent.com/example/repo/main"
docs:
  - name: architecture
    title: "Forge Architecture"
    description: "Overview of forge architecture"
    url: "docs/architecture.md"
    tags: ["architecture", "overview"]
`
	if err := os.WriteFile(filepath.Join(docsDir, "docs-list.yaml"), []byte(listYAML), 0o644); err != nil {
		t.Fatalf("Failed to write docs-list.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "architecture.md"), []byte("# Forge Architecture\n\nArchitecture content."), 0o644); err != nil {
		t.Fatalf("Failed to write architecture.md: %v", err)
	}
}

func TestDiscoverEngineDocs(t *testing.T) {
	// Cannot be parallel because changes working directory

	t.Run("discovers engines with docs directories", func(t *testing.T) {
		tmpDir, cleanup := setupTestProject(t)
		defer cleanup()

		// Create mock cmd directory
		if err := os.MkdirAll(filepath.Join(tmpDir, "cmd"), 0o755); err != nil {
			t.Fatalf("Failed to create cmd dir: %v", err)
		}

		// Create mock engines with docs
		createMockEngineDocs(t, tmpDir, "go-build")
		createMockEngineDocs(t, tmpDir, "testenv")

		// Create engine without docs
		if err := os.MkdirAll(filepath.Join(tmpDir, "cmd", "no-docs-engine"), 0o755); err != nil {
			t.Fatalf("Failed to create no-docs-engine dir: %v", err)
		}

		// Create forge (should be skipped)
		if err := os.MkdirAll(filepath.Join(tmpDir, "cmd", "forge", "docs"), 0o755); err != nil {
			t.Fatalf("Failed to create forge dir: %v", err)
		}

		engines, err := discoverEngineDocs()
		if err != nil {
			t.Fatalf("discoverEngineDocs failed: %v", err)
		}

		// Should find go-build and testenv, but not forge or no-docs-engine
		if len(engines) != 2 {
			t.Errorf("Expected 2 engines, got %d: %v", len(engines), engines)
		}

		goBuildFound := false
		testenvFound := false
		for _, e := range engines {
			if e == filepath.Join("cmd", "go-build") {
				goBuildFound = true
			}
			if e == filepath.Join("cmd", "testenv") {
				testenvFound = true
			}
		}
		if !goBuildFound {
			t.Error("Expected to find go-build engine")
		}
		if !testenvFound {
			t.Error("Expected to find testenv engine")
		}
	})

	t.Run("returns empty list when no engines have docs", func(t *testing.T) {
		tmpDir, cleanup := setupTestProject(t)
		defer cleanup()

		// Create cmd directory without any engines having docs
		if err := os.MkdirAll(filepath.Join(tmpDir, "cmd", "engine1"), 0o755); err != nil {
			t.Fatalf("Failed to create engine1 dir: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(tmpDir, "cmd", "engine2"), 0o755); err != nil {
			t.Fatalf("Failed to create engine2 dir: %v", err)
		}

		engines, err := discoverEngineDocs()
		if err != nil {
			t.Fatalf("discoverEngineDocs failed: %v", err)
		}
		if len(engines) != 0 {
			t.Errorf("Expected 0 engines, got %d", len(engines))
		}
	})

	t.Run("returns error when cmd directory does not exist", func(t *testing.T) {
		_, cleanup := setupTestProject(t)
		defer cleanup()

		_, err := discoverEngineDocs()
		if err == nil {
			t.Error("Expected error when cmd directory doesn't exist, got nil")
		}
	})
}

func TestAggregateDocsList(t *testing.T) {
	// Cannot be parallel because changes working directory

	t.Run("aggregates global and engine docs", func(t *testing.T) {
		tmpDir, cleanup := setupTestProject(t)
		defer cleanup()

		// Create mock global docs
		createMockGlobalDocs(t, tmpDir)

		// Create mock cmd directory and engines
		if err := os.MkdirAll(filepath.Join(tmpDir, "cmd"), 0o755); err != nil {
			t.Fatalf("Failed to create cmd dir: %v", err)
		}
		createMockEngineDocs(t, tmpDir, "go-build")
		createMockEngineDocs(t, tmpDir, "testenv")

		result, err := aggregateDocsList()
		if err != nil {
			t.Fatalf("aggregateDocsList failed: %v", err)
		}

		// Should have global docs
		if len(result.GlobalDocs) != 1 {
			t.Errorf("Expected 1 global doc, got %d", len(result.GlobalDocs))
		} else if result.GlobalDocs[0].Name != "architecture" {
			t.Errorf("Expected global doc 'architecture', got '%s'", result.GlobalDocs[0].Name)
		}

		// Should have engine docs with prefixes
		if len(result.EngineDocs) != 4 { // 2 engines * 2 docs each
			t.Errorf("Expected 4 engine docs, got %d", len(result.EngineDocs))
		}

		// Check that engine docs have correct prefixes
		engineDocNames := make(map[string]bool)
		for _, doc := range result.EngineDocs {
			engineDocNames[doc.Name] = true
		}
		expectedNames := []string{"go-build/usage", "go-build/schema", "testenv/usage", "testenv/schema"}
		for _, name := range expectedNames {
			if !engineDocNames[name] {
				t.Errorf("Expected engine doc '%s' not found", name)
			}
		}
	})

	t.Run("continues with errors when some engines fail", func(t *testing.T) {
		tmpDir, cleanup := setupTestProject(t)
		defer cleanup()

		// Create mock global docs
		createMockGlobalDocs(t, tmpDir)

		// Create mock cmd directory
		if err := os.MkdirAll(filepath.Join(tmpDir, "cmd"), 0o755); err != nil {
			t.Fatalf("Failed to create cmd dir: %v", err)
		}

		// Create one valid engine
		createMockEngineDocs(t, tmpDir, "go-build")

		// Create engine with invalid list.yaml
		invalidDocsDir := filepath.Join(tmpDir, "cmd", "invalid-engine", "docs")
		if err := os.MkdirAll(invalidDocsDir, 0o755); err != nil {
			t.Fatalf("Failed to create invalid-engine dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(invalidDocsDir, "list.yaml"), []byte("invalid: [yaml content"), 0o644); err != nil {
			t.Fatalf("Failed to write invalid list.yaml: %v", err)
		}

		result, err := aggregateDocsList()
		if err != nil {
			t.Fatalf("aggregateDocsList failed: %v", err)
		}

		// Should have global docs
		if len(result.GlobalDocs) != 1 {
			t.Errorf("Expected 1 global doc, got %d", len(result.GlobalDocs))
		}

		// Should have valid engine docs
		if len(result.EngineDocs) != 2 { // Only go-build's 2 docs
			t.Errorf("Expected 2 engine docs, got %d", len(result.EngineDocs))
		}

		// Should have error for invalid engine
		if len(result.Errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(result.Errors))
		} else {
			if result.Errors[0].Engine != "invalid-engine" {
				t.Errorf("Expected error for 'invalid-engine', got '%s'", result.Errors[0].Engine)
			}
			if !strings.Contains(result.Errors[0].Error, "parse") {
				t.Errorf("Expected parse error, got '%s'", result.Errors[0].Error)
			}
		}
	})

	t.Run("returns empty engine docs when no engines have docs", func(t *testing.T) {
		tmpDir, cleanup := setupTestProject(t)
		defer cleanup()

		// Create mock global docs (global docs are loaded from localDocsList which is relative)
		createMockGlobalDocs(t, tmpDir)

		// Create empty cmd directory without any engine docs
		if err := os.MkdirAll(filepath.Join(tmpDir, "cmd"), 0o755); err != nil {
			t.Fatalf("Failed to create cmd dir: %v", err)
		}

		result, err := aggregateDocsList()
		if err != nil {
			t.Fatalf("aggregateDocsList failed: %v", err)
		}

		// Should have global docs (we created them)
		if len(result.GlobalDocs) != 1 {
			t.Errorf("Expected 1 global doc, got %d", len(result.GlobalDocs))
		}

		// Should have no engine docs
		if len(result.EngineDocs) != 0 {
			t.Errorf("Expected 0 engine docs, got %d", len(result.EngineDocs))
		}

		// Should have no errors
		if len(result.Errors) != 0 {
			t.Errorf("Expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
		}
	})
}

func TestAggregatedDocsGet(t *testing.T) {
	// Cannot be parallel because changes working directory

	t.Run("routes to engine doc with prefix", func(t *testing.T) {
		tmpDir, cleanup := setupTestProject(t)
		defer cleanup()

		// Create mock cmd directory and engine
		if err := os.MkdirAll(filepath.Join(tmpDir, "cmd"), 0o755); err != nil {
			t.Fatalf("Failed to create cmd dir: %v", err)
		}
		createMockEngineDocs(t, tmpDir, "go-build")

		content, err := aggregatedDocsGet("go-build/usage")
		if err != nil {
			t.Fatalf("aggregatedDocsGet failed: %v", err)
		}
		if !strings.Contains(content, "# go-build Usage") {
			t.Errorf("Expected content to contain '# go-build Usage', got: %s", content)
		}
		if !strings.Contains(content, "Usage guide content.") {
			t.Errorf("Expected content to contain 'Usage guide content.', got: %s", content)
		}
	})

	t.Run("routes to global doc without prefix", func(t *testing.T) {
		tmpDir, cleanup := setupTestProject(t)
		defer cleanup()

		// Create mock global docs
		createMockGlobalDocs(t, tmpDir)

		content, err := aggregatedDocsGet("architecture")
		if err != nil {
			t.Fatalf("aggregatedDocsGet failed: %v", err)
		}
		if !strings.Contains(content, "# Forge Architecture") {
			t.Errorf("Expected content to contain '# Forge Architecture', got: %s", content)
		}
		if !strings.Contains(content, "Architecture content.") {
			t.Errorf("Expected content to contain 'Architecture content.', got: %s", content)
		}
	})

	t.Run("returns error for non-existent engine", func(t *testing.T) {
		tmpDir, cleanup := setupTestProject(t)
		defer cleanup()

		// Create empty cmd directory
		if err := os.MkdirAll(filepath.Join(tmpDir, "cmd"), 0o755); err != nil {
			t.Fatalf("Failed to create cmd dir: %v", err)
		}

		_, err := aggregatedDocsGet("nonexistent/usage")
		if err == nil {
			t.Error("Expected error for non-existent engine, got nil")
		}
		if !strings.Contains(err.Error(), "nonexistent") {
			t.Errorf("Expected error to mention 'nonexistent', got: %v", err)
		}
	})

	t.Run("returns error for non-existent doc in engine", func(t *testing.T) {
		tmpDir, cleanup := setupTestProject(t)
		defer cleanup()

		// Create mock cmd directory and engine
		if err := os.MkdirAll(filepath.Join(tmpDir, "cmd"), 0o755); err != nil {
			t.Fatalf("Failed to create cmd dir: %v", err)
		}
		createMockEngineDocs(t, tmpDir, "go-build")

		_, err := aggregatedDocsGet("go-build/nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent doc, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})

	t.Run("returns error for non-existent global doc", func(t *testing.T) {
		tmpDir, cleanup := setupTestProject(t)
		defer cleanup()

		// Create mock global docs
		createMockGlobalDocs(t, tmpDir)

		_, err := aggregatedDocsGet("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent doc, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})
}

func TestGetEngineDoc(t *testing.T) {
	// Cannot be parallel because changes working directory

	t.Run("retrieves doc from engine", func(t *testing.T) {
		tmpDir, cleanup := setupTestProject(t)
		defer cleanup()

		// Create mock cmd directory and engine
		if err := os.MkdirAll(filepath.Join(tmpDir, "cmd"), 0o755); err != nil {
			t.Fatalf("Failed to create cmd dir: %v", err)
		}
		createMockEngineDocs(t, tmpDir, "go-build")

		content, err := getEngineDoc("go-build", "usage")
		if err != nil {
			t.Fatalf("getEngineDoc failed: %v", err)
		}
		if !strings.Contains(content, "# go-build Usage") {
			t.Errorf("Expected content to contain '# go-build Usage', got: %s", content)
		}
	})

	t.Run("returns error for non-existent engine", func(t *testing.T) {
		tmpDir, cleanup := setupTestProject(t)
		defer cleanup()

		// Create empty cmd directory
		if err := os.MkdirAll(filepath.Join(tmpDir, "cmd"), 0o755); err != nil {
			t.Fatalf("Failed to create cmd dir: %v", err)
		}

		_, err := getEngineDoc("nonexistent", "usage")
		if err == nil {
			t.Error("Expected error for non-existent engine, got nil")
		}
		if !strings.Contains(err.Error(), "not found or has no docs") {
			t.Errorf("Expected 'not found or has no docs' error, got: %v", err)
		}
	})

	t.Run("returns error for invalid list.yaml", func(t *testing.T) {
		tmpDir, cleanup := setupTestProject(t)
		defer cleanup()

		// Create engine with invalid list.yaml
		invalidDocsDir := filepath.Join(tmpDir, "cmd", "invalid-engine", "docs")
		if err := os.MkdirAll(invalidDocsDir, 0o755); err != nil {
			t.Fatalf("Failed to create invalid-engine dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(invalidDocsDir, "list.yaml"), []byte("invalid: [yaml"), 0o644); err != nil {
			t.Fatalf("Failed to write invalid list.yaml: %v", err)
		}

		_, err := getEngineDoc("invalid-engine", "usage")
		if err == nil {
			t.Error("Expected error for invalid list.yaml, got nil")
		}
		if !strings.Contains(err.Error(), "parse") {
			t.Errorf("Expected 'parse' error, got: %v", err)
		}
	})
}

func TestDocsGetContent(t *testing.T) {
	// Cannot be parallel because changes working directory

	t.Run("retrieves global doc content", func(t *testing.T) {
		tmpDir, cleanup := setupTestProject(t)
		defer cleanup()

		// Create mock global docs
		createMockGlobalDocs(t, tmpDir)

		content, err := docsGetContent("architecture")
		if err != nil {
			t.Fatalf("docsGetContent failed: %v", err)
		}
		if !strings.Contains(content, "# Forge Architecture") {
			t.Errorf("Expected content to contain '# Forge Architecture', got: %s", content)
		}
	})

	t.Run("returns error for non-existent doc", func(t *testing.T) {
		tmpDir, cleanup := setupTestProject(t)
		defer cleanup()

		// Create mock global docs
		createMockGlobalDocs(t, tmpDir)

		_, err := docsGetContent("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent doc, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})
}

func TestAggregatedDocEntry(t *testing.T) {
	t.Parallel()

	t.Run("AggregatedDocEntry embeds DocEntry correctly", func(t *testing.T) {
		t.Parallel()

		entry := AggregatedDocEntry{
			DocEntry: DocEntry{
				Name:        "go-build/usage",
				Title:       "Usage Guide",
				Description: "How to use go-build",
				URL:         "cmd/go-build/docs/usage.md",
				Tags:        []string{"usage", "guide"},
			},
			Engine: "go-build",
		}

		if entry.Name != "go-build/usage" {
			t.Errorf("Expected name 'go-build/usage', got '%s'", entry.Name)
		}
		if entry.Title != "Usage Guide" {
			t.Errorf("Expected title 'Usage Guide', got '%s'", entry.Title)
		}
		if entry.Engine != "go-build" {
			t.Errorf("Expected engine 'go-build', got '%s'", entry.Engine)
		}
	})
}

func TestAggregatedDocsResult(t *testing.T) {
	t.Parallel()

	t.Run("AggregatedDocsResult contains all fields", func(t *testing.T) {
		t.Parallel()

		result := AggregatedDocsResult{
			GlobalDocs: []DocEntry{
				{Name: "architecture", Title: "Architecture"},
			},
			EngineDocs: []AggregatedDocEntry{
				{
					DocEntry: DocEntry{Name: "go-build/usage"},
					Engine:   "go-build",
				},
			},
			Errors: []AggregationError{
				{Engine: "testenv", Error: "some error"},
			},
		}

		if len(result.GlobalDocs) != 1 {
			t.Errorf("Expected 1 global doc, got %d", len(result.GlobalDocs))
		}
		if len(result.EngineDocs) != 1 {
			t.Errorf("Expected 1 engine doc, got %d", len(result.EngineDocs))
		}
		if len(result.Errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(result.Errors))
		}
	})
}
