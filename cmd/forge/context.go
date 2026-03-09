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
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alexandremahdhaoui/forge/internal/forgepath"
)

// parseGitRepoURL converts a git URL to a Go module-style path.
// Supported formats:
//   - git@github.com:user/repo.git -> github.com/user/repo
//   - git@github.com:user/repo     -> github.com/user/repo
//   - https://github.com/user/repo -> github.com/user/repo
//   - https://github.com/user/repo.git -> github.com/user/repo
//   - ssh://git@github.com/user/repo.git -> github.com/user/repo
func parseGitRepoURL(rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("empty git URL")
	}

	var host, path string

	switch {
	case strings.HasPrefix(rawURL, "git@"):
		// SSH shorthand: git@host:user/repo.git
		withoutPrefix := strings.TrimPrefix(rawURL, "git@")
		colonIdx := strings.Index(withoutPrefix, ":")
		if colonIdx < 0 {
			return "", fmt.Errorf("invalid SSH git URL (missing ':'): %s", rawURL)
		}
		host = withoutPrefix[:colonIdx]
		path = withoutPrefix[colonIdx+1:]

	case strings.HasPrefix(rawURL, "https://") || strings.HasPrefix(rawURL, "http://"):
		parsed, err := url.Parse(rawURL)
		if err != nil {
			return "", fmt.Errorf("failed to parse URL %s: %w", rawURL, err)
		}
		host = parsed.Host
		path = strings.TrimPrefix(parsed.Path, "/")

	case strings.HasPrefix(rawURL, "ssh://"):
		parsed, err := url.Parse(rawURL)
		if err != nil {
			return "", fmt.Errorf("failed to parse URL %s: %w", rawURL, err)
		}
		host = parsed.Host
		// Strip user info (e.g., git@) from host
		if parsed.User != nil {
			host = parsed.Hostname()
		}
		path = strings.TrimPrefix(parsed.Path, "/")

	default:
		return "", fmt.Errorf("unsupported git URL format: %s", rawURL)
	}

	// Strip .git suffix
	path = strings.TrimSuffix(path, ".git")

	if host == "" || path == "" {
		return "", fmt.Errorf("could not extract host/path from git URL: %s", rawURL)
	}

	return host + "/" + path, nil
}

// isGitURL returns true if the string looks like a git URL.
// Checks prefixes: git@, https://, http://, ssh://
func isGitURL(s string) bool {
	return strings.HasPrefix(s, "git@") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "ssh://")
}

// resolveContextDir resolves a context value to an absolute directory path.
// It returns the absolute directory path, a cleanup function, and any error.
//
// Resolution rules:
//   - Empty or "." -> current working directory
//   - Absolute path -> validated with os.Stat
//   - Relative path (starts with ./ or ../) -> converted to absolute, validated
//   - Git URL -> parsed to module path, resolved via go.work, or cloned to temp dir
//
// The cleanup function is a no-op for local paths and removes the temp directory
// for cloned repos. Callers must call cleanup when done.
func resolveContextDir(contextValue string) (string, func(), error) {
	noop := func() {}

	// Empty or "." -> current working directory
	if contextValue == "" || contextValue == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return "", noop, fmt.Errorf("failed to get working directory: %w", err)
		}
		return cwd, noop, nil
	}

	// Absolute path
	if filepath.IsAbs(contextValue) {
		if _, err := os.Stat(contextValue); err != nil {
			return "", noop, fmt.Errorf("context directory does not exist: %w", err)
		}
		return contextValue, noop, nil
	}

	// Relative path (starts with ./ or ../)
	if strings.HasPrefix(contextValue, "./") || strings.HasPrefix(contextValue, "../") {
		absPath, err := filepath.Abs(contextValue)
		if err != nil {
			return "", noop, fmt.Errorf("failed to resolve relative path %s: %w", contextValue, err)
		}
		if _, err := os.Stat(absPath); err != nil {
			return "", noop, fmt.Errorf("context directory does not exist: %w", err)
		}
		return absPath, noop, nil
	}

	// Git URL
	if isGitURL(contextValue) {
		modulePath, err := parseGitRepoURL(contextValue)
		if err != nil {
			return "", noop, fmt.Errorf("failed to parse git URL: %w", err)
		}

		// Try to resolve via go.work first
		if dir, ok := resolveViaGoWork(modulePath); ok {
			return dir, noop, nil
		}

		// Fall back to git clone
		return gitCloneToTemp(contextValue)
	}

	return "", noop, fmt.Errorf("unsupported context value: %s", contextValue)
}

// resolveViaGoWork resolves a Go module path to a local directory using the
// nearest go.work file. It walks up from CWD looking for go.work, parses its
// use directives, and checks each member's go.mod for a matching module path.
// Returns (absoluteDir, true) if found, ("", false) otherwise.
func resolveViaGoWork(modulePath string) (string, bool) {
	goWorkDir := forgepath.FindGoWork()
	if goWorkDir == "" {
		return "", false
	}

	goWorkPath := filepath.Join(goWorkDir, "go.work")
	content, err := os.ReadFile(goWorkPath)
	if err != nil {
		return "", false
	}

	useDirs := forgepath.ParseGoWorkUseDirs(string(content))

	for _, useDir := range useDirs {
		var absDir string
		if filepath.IsAbs(useDir) {
			absDir = useDir
		} else {
			absDir = filepath.Join(goWorkDir, useDir)
		}

		modPath := forgepath.ReadModulePath(filepath.Join(absDir, "go.mod"))
		if modPath != "" && modPath == modulePath {
			return absDir, true
		}
	}

	return "", false
}

// gitCloneToTemp clones a git repository into a temporary directory.
// Returns the temp directory path, a cleanup function that removes it, and any error.
// The clone uses --depth=1 to minimize download size.
func gitCloneToTemp(gitURL string) (string, func(), error) {
	noop := func() {}

	tmpDir, err := os.MkdirTemp("", "forge-context-*")
	if err != nil {
		return "", noop, fmt.Errorf("failed to create temp directory: %w", err)
	}

	cleanup := func() { _ = os.RemoveAll(tmpDir) }

	cmd := exec.Command("git", "clone", "--depth=1", gitURL, tmpDir)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		cleanup()
		return "", noop, fmt.Errorf("git clone failed for %s: %w", gitURL, err)
	}

	return tmpDir, cleanup, nil
}
