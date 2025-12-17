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
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"sigs.k8s.io/yaml"
)

const (
	httpTimeout = 10 * time.Second
)

const (
	docsListURL   = "https://raw.githubusercontent.com/alexandremahdhaoui/forge/refs/heads/main/docs/docs-list.yaml"
	localDocsList = "docs/docs-list.yaml"
	localDocsDir  = "docs"
)

// DocStore represents the docs-list.yaml structure
type DocStore struct {
	Version string     `yaml:"version"`
	BaseURL string     `yaml:"baseURL"`
	Docs    []DocEntry `yaml:"docs"`
}

// DocEntry represents a single document in the list
type DocEntry struct {
	Name        string   `yaml:"name"`
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	URL         string   `yaml:"url"`
	Tags        []string `yaml:"tags"`
}

// runDocs handles the "forge docs" command
func runDocs(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: forge docs <list|get> [doc-name]")
	}

	operation := args[0]

	switch operation {
	case "list":
		return docsList()
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: forge docs get <doc-name>")
		}
		return docsGet(args[1])
	default:
		return fmt.Errorf("unknown operation: %s (valid: list, get)", operation)
	}
}

// docsList lists all available documentation
func docsList() error {
	// Fetch docs store
	store, err := fetchDocsStore()
	if err != nil {
		return fmt.Errorf("failed to fetch docs list: %w", err)
	}

	// Print header
	fmt.Printf("Available documentation (version %s):\n\n", store.Version)

	// Find longest name for alignment
	maxNameLen := 0
	for _, doc := range store.Docs {
		if len(doc.Name) > maxNameLen {
			maxNameLen = len(doc.Name)
		}
	}

	// Print docs
	for _, doc := range store.Docs {
		// Format: name (padded) - title
		fmt.Printf("  %-*s  %s\n", maxNameLen, doc.Name, doc.Title)
		fmt.Printf("  %*s  %s\n", maxNameLen, "", doc.Description)

		// Print tags
		if len(doc.Tags) > 0 {
			fmt.Printf("  %*s  Tags: %s\n", maxNameLen, "", strings.Join(doc.Tags, ", "))
		}
		fmt.Println()
	}

	fmt.Printf("Usage: forge docs get <name>\n")
	fmt.Printf("Example: forge docs get architecture\n")

	return nil
}

// docsGet fetches and displays a specific document
func docsGet(name string) error {
	// Fetch docs store
	store, err := fetchDocsStore()
	if err != nil {
		return fmt.Errorf("failed to fetch docs list: %w", err)
	}

	// Find document
	var doc *DocEntry
	for i := range store.Docs {
		if store.Docs[i].Name == name {
			doc = &store.Docs[i]
			break
		}
	}

	if doc == nil {
		return fmt.Errorf("document not found: %s\nRun 'forge docs list' to see available documentation", name)
	}

	// Try to read from local file first (when running inside the repo)
	localPath := filepath.Join(doc.URL)
	if content, err := os.ReadFile(localPath); err == nil {
		// Print header
		fmt.Printf("# %s\n", doc.Title)
		fmt.Printf("# %s\n", doc.Description)
		fmt.Printf("# Source: local (%s)\n", localPath)
		fmt.Println()

		// Print content
		fmt.Print(string(content))
		return nil
	}

	// Fall back to fetching from URL
	docURL := store.BaseURL + "/" + doc.URL
	content, err := fetchURL(docURL)
	if err != nil {
		return fmt.Errorf("failed to fetch document: %w", err)
	}

	// Print header
	fmt.Printf("# %s\n", doc.Title)
	fmt.Printf("# %s\n", doc.Description)
	fmt.Printf("# URL: %s\n", docURL)
	fmt.Println()

	// Print content
	fmt.Print(content)

	return nil
}

// fetchDocsStore fetches and parses the docs-list.yaml
// It first checks if running inside the forge repository and reads locally if available
func fetchDocsStore() (*DocStore, error) {
	// Try to read from local file first (when running inside the repo)
	if content, err := os.ReadFile(localDocsList); err == nil {
		var store DocStore
		if err := yaml.Unmarshal(content, &store); err != nil {
			return nil, fmt.Errorf("failed to parse local docs list: %w", err)
		}
		return &store, nil
	}

	// Fall back to fetching from URL
	content, err := fetchURL(docsListURL)
	if err != nil {
		return nil, err
	}

	var store DocStore
	if err := yaml.Unmarshal([]byte(content), &store); err != nil {
		return nil, fmt.Errorf("failed to parse docs list: %w", err)
	}

	return &store, nil
}

// AggregatedDocEntry extends DocEntry with engine information
type AggregatedDocEntry struct {
	DocEntry
	Engine string `yaml:"engine" json:"engine"`
}

// AggregatedDocsResult represents the combined result of all documentation sources
type AggregatedDocsResult struct {
	GlobalDocs []DocEntry           `json:"globalDocs"`
	EngineDocs []AggregatedDocEntry `json:"engineDocs"`
	Errors     []AggregationError   `json:"errors,omitempty"`
}

// AggregationError represents an error when loading docs from an engine
type AggregationError struct {
	Engine string `json:"engine"`
	Error  string `json:"error"`
}

// aggregateDocsList scans cmd/*/docs/list.yaml files and returns aggregated documentation
// It returns both global forge docs and engine-specific docs
func aggregateDocsList() (*AggregatedDocsResult, error) {
	result := &AggregatedDocsResult{
		GlobalDocs: []DocEntry{},
		EngineDocs: []AggregatedDocEntry{},
		Errors:     []AggregationError{},
	}

	// Load global docs first
	globalStore, err := fetchDocsStore()
	if err != nil {
		// Global docs are not required for aggregation to succeed
		result.Errors = append(result.Errors, AggregationError{
			Engine: "forge",
			Error:  err.Error(),
		})
	} else {
		result.GlobalDocs = globalStore.Docs
	}

	// Discover engine directories with docs
	engineDirs, err := discoverEngineDocs()
	if err != nil {
		// Continue with global docs only
		return result, nil
	}

	// Load docs from each engine
	for _, engineDir := range engineDirs {
		engineName := filepath.Base(engineDir)
		listPath := filepath.Join(engineDir, "docs", "list.yaml")

		content, err := os.ReadFile(listPath)
		if err != nil {
			result.Errors = append(result.Errors, AggregationError{
				Engine: engineName,
				Error:  fmt.Sprintf("failed to read list.yaml: %v", err),
			})
			continue
		}

		var store enginedocs.DocStore
		if err := yaml.Unmarshal(content, &store); err != nil {
			result.Errors = append(result.Errors, AggregationError{
				Engine: engineName,
				Error:  fmt.Sprintf("failed to parse list.yaml: %v", err),
			})
			continue
		}

		// Add engine docs with engine prefix
		for _, doc := range store.Docs {
			result.EngineDocs = append(result.EngineDocs, AggregatedDocEntry{
				DocEntry: DocEntry{
					Name:        engineName + "/" + doc.Name,
					Title:       doc.Title,
					Description: doc.Description,
					URL:         doc.URL,
					Tags:        doc.Tags,
				},
				Engine: engineName,
			})
		}
	}

	return result, nil
}

// discoverEngineDocs finds all engine directories that have docs/list.yaml
func discoverEngineDocs() ([]string, error) {
	var engines []string

	// Look for cmd/*/docs/list.yaml
	entries, err := os.ReadDir("cmd")
	if err != nil {
		return nil, fmt.Errorf("failed to read cmd directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip forge itself (it has global docs)
		if entry.Name() == "forge" {
			continue
		}

		listPath := filepath.Join("cmd", entry.Name(), "docs", "list.yaml")
		if _, err := os.Stat(listPath); err == nil {
			engines = append(engines, filepath.Join("cmd", entry.Name()))
		}
	}

	return engines, nil
}

// aggregatedDocsGet retrieves a document by name, routing to the correct engine
// For names like "go-build/usage", it reads from the go-build engine
// For names without a prefix, it reads from global docs
func aggregatedDocsGet(name string) (string, error) {
	// Check if this is an engine-prefixed name
	if strings.Contains(name, "/") {
		parts := strings.SplitN(name, "/", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid document name format: %s", name)
		}
		engineName := parts[0]
		docName := parts[1]

		return getEngineDoc(engineName, docName)
	}

	// Fall back to global docs
	return docsGetContent(name)
}

// getEngineDoc retrieves a document from a specific engine
func getEngineDoc(engineName, docName string) (string, error) {
	listPath := filepath.Join("cmd", engineName, "docs", "list.yaml")

	content, err := os.ReadFile(listPath)
	if err != nil {
		return "", fmt.Errorf("engine '%s' not found or has no docs: %w", engineName, err)
	}

	var store enginedocs.DocStore
	if err := yaml.Unmarshal(content, &store); err != nil {
		return "", fmt.Errorf("failed to parse list.yaml for engine '%s': %w", engineName, err)
	}

	// Find the document
	var doc *enginedocs.DocEntry
	for i := range store.Docs {
		if store.Docs[i].Name == docName {
			doc = &store.Docs[i]
			break
		}
	}

	if doc == nil {
		return "", fmt.Errorf("document '%s' not found in engine '%s'\nRun 'forge docs list' to see available documentation", docName, engineName)
	}

	// Try to read from local file first
	if docContent, err := os.ReadFile(doc.URL); err == nil {
		return string(docContent), nil
	}

	// Fall back to remote URL if BaseURL is set
	if store.BaseURL != "" {
		docURL := store.BaseURL + "/" + doc.URL
		return fetchURL(docURL)
	}

	return "", fmt.Errorf("document file not found: %s", doc.URL)
}

// docsGetContent retrieves a global document content by name
// This is a helper that returns just the content without printing
func docsGetContent(name string) (string, error) {
	store, err := fetchDocsStore()
	if err != nil {
		return "", fmt.Errorf("failed to fetch docs list: %w", err)
	}

	var doc *DocEntry
	for i := range store.Docs {
		if store.Docs[i].Name == name {
			doc = &store.Docs[i]
			break
		}
	}

	if doc == nil {
		return "", fmt.Errorf("document not found: %s\nRun 'forge docs list' to see available documentation", name)
	}

	// Try to read from local file first
	if content, err := os.ReadFile(doc.URL); err == nil {
		return string(content), nil
	}

	// Fall back to remote URL
	docURL := store.BaseURL + "/" + doc.URL
	return fetchURL(docURL)
}

// fetchURL fetches content from a URL with timeout
func fetchURL(url string) (string, error) {
	client := &http.Client{
		Timeout: httpTimeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}
