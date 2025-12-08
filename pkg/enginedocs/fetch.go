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
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"sigs.k8s.io/yaml"
)

const (
	// httpTimeout is the timeout for remote HTTP requests.
	httpTimeout = 10 * time.Second
	// listFileName is the standard name for the documentation registry file.
	listFileName = "list.yaml"
)

// FetchDocStore retrieves the DocStore for an engine using local-first, remote-fallback logic.
// It first attempts to read from {cfg.LocalDir}/list.yaml. If the local file does not exist
// and cfg.BaseURL is set, it fetches from {cfg.BaseURL}/{cfg.LocalDir}/list.yaml.
func FetchDocStore(cfg Config) (*DocStore, error) {
	localPath := filepath.Join(cfg.LocalDir, listFileName)

	// Try to read from local file first
	content, err := os.ReadFile(localPath)
	if err == nil {
		var store DocStore
		if err := yaml.Unmarshal(content, &store); err != nil {
			return nil, fmt.Errorf("failed to parse local docs list at %s: %w", localPath, err)
		}
		return &store, nil
	}

	// If local file doesn't exist and we have a BaseURL, try remote
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read local docs list at %s: %w", localPath, err)
	}

	// Local file doesn't exist - try remote if BaseURL is configured
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("local docs list not found at %s and no BaseURL configured for remote fetch", localPath)
	}

	// Construct remote URL: {BaseURL}/{LocalDir}/list.yaml
	remoteURL := fmt.Sprintf("%s/%s/%s", cfg.BaseURL, cfg.LocalDir, listFileName)

	remoteContent, err := fetchURL(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch remote docs list from %s: %w", remoteURL, err)
	}

	var store DocStore
	if err := yaml.Unmarshal([]byte(remoteContent), &store); err != nil {
		return nil, fmt.Errorf("failed to parse remote docs list from %s: %w", remoteURL, err)
	}

	return &store, nil
}

// fetchURL fetches content from the given URL with a 10-second timeout.
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
