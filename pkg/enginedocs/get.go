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
	"fmt"
	"os"
)

// DocsGet retrieves the content of a specific document by name.
// It first fetches the DocStore to find the document entry, then attempts to read
// the document content using local-first, remote-fallback logic.
//
// The function:
// 1. Calls FetchDocStore(cfg) to get the registry
// 2. Finds the entry where DocEntry.Name matches the provided name
// 3. If not found, returns an error guiding the user to run 'docs list'
// 4. Tries reading the local file at DocEntry.URL (relative path from repo root)
// 5. If local file is missing and BaseURL is set, fetches from {cfg.BaseURL}/{entry.URL}
// 6. Returns the content as a string
func DocsGet(cfg Config, name string) (string, error) {
	// Fetch the documentation store
	store, err := FetchDocStore(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to fetch docs list: %w", err)
	}

	// Find the document entry by name
	var doc *DocEntry
	for i := range store.Docs {
		if store.Docs[i].Name == name {
			doc = &store.Docs[i]
			break
		}
	}

	if doc == nil {
		return "", fmt.Errorf("document not found: %s\nRun 'docs list' to see available documentation", name)
	}

	// Try to read from local file first (DocEntry.URL is relative to repo root)
	content, err := os.ReadFile(doc.URL)
	if err == nil {
		return string(content), nil
	}

	// If local file doesn't exist and we have a BaseURL, try remote
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read local document at %s: %w", doc.URL, err)
	}

	// Local file doesn't exist - try remote if BaseURL is configured
	if cfg.BaseURL == "" {
		return "", fmt.Errorf("local document not found at %s and no BaseURL configured for remote fetch", doc.URL)
	}

	// Construct remote URL: {BaseURL}/{entry.URL}
	remoteURL := fmt.Sprintf("%s/%s", cfg.BaseURL, doc.URL)

	remoteContent, err := fetchURL(remoteURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch remote document from %s: %w", remoteURL, err)
	}

	return remoteContent, nil
}
