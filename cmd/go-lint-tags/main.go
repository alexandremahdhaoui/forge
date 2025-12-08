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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexandremahdhaoui/forge/internal/cli"
	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
)

const Name = "go-lint-tags"

// Version information (set via ldflags during build)
var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
)

// docsConfig is the configuration for the docs subcommand.
var docsConfig = &enginedocs.Config{
	EngineName:   Name,
	LocalDir:     "cmd/go-lint-tags/docs",
	BaseURL:      "https://raw.githubusercontent.com/alexandremahdhaoui/forge/refs/heads/main",
	RequiredDocs: []string{"usage", "schema"},
}

func main() {
	cli.Bootstrap(cli.Config{
		Name:           Name,
		Version:        Version,
		CommitSHA:      CommitSHA,
		BuildTimestamp: BuildTimestamp,
		RunMCP:         runMCPServer,
		DocsConfig:     docsConfig,
	})
}

// findTestFiles recursively finds all *_test.go files
func findTestFiles(root string) ([]string, error) {
	var testFiles []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor, .git, .tmp directories
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == ".git" || name == ".tmp" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if it's a test file
		if strings.HasSuffix(path, "_test.go") {
			testFiles = append(testFiles, path)
		}

		return nil
	})

	return testFiles, err
}

// checkBuildTag checks if a file has a valid build tag
func checkBuildTag(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)

	// Check first few lines for build tag
	lineCount := 0
	for scanner.Scan() && lineCount < 5 {
		line := strings.TrimSpace(scanner.Text())
		lineCount++

		// Check for go:build directive
		if strings.HasPrefix(line, "//go:build") {
			// Verify it's one of our expected tags
			if strings.Contains(line, "unit") ||
				strings.Contains(line, "integration") ||
				strings.Contains(line, "e2e") {
				return true, nil
			}
		}

		// Skip empty lines and comments, but stop at package declaration
		if strings.HasPrefix(line, "package ") {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, nil
}

// verifyTags performs the tag verification and returns results.
// Returns (filesWithoutTags, totalFiles, error).
func verifyTags(rootDir string) ([]string, int, error) {
	// Find all test files
	testFiles, err := findTestFiles(rootDir)
	if err != nil {
		return nil, 0, fmt.Errorf("error finding test files: %w", err)
	}

	if len(testFiles) == 0 {
		return []string{}, 0, nil
	}

	// Verify each test file has a build tag
	var filesWithoutTags []string
	for _, file := range testFiles {
		hasBuildTag, err := checkBuildTag(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking %s: %v\n", file, err)
			continue
		}
		if !hasBuildTag {
			filesWithoutTags = append(filesWithoutTags, file)
		}
	}

	return filesWithoutTags, len(testFiles), nil
}
