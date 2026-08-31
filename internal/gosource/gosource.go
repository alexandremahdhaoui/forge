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

// Package gosource answers what a Go package is made of.
//
// A Go package spans every .go file in its directory, so a dependency
// detector that reads one of them has read a fraction of the package. The
// dependency detector used to take the first file it found and walk that
// file's imports alone: forge-ci's binary tracked
// reconcilecontroller/index.go and never reconcilecontroller.go, so 14 of its
// 24 packages were invisible to the rebuild decision and an edit to any of
// them left the build reporting "unchanged".
//
// Both detectors read a package through this one function so they cannot
// answer the question differently.
package gosource

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ListGoFiles returns the absolute path of every non-test .go file directly
// in dir, sorted as the filesystem lists them. Subdirectories are other
// packages and are not included.
//
// A directory with no .go files is an error: the caller asked about a package
// and there is none, which is a broken import rather than an empty answer.
func ListGoFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	var files []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		absPath, err := filepath.Abs(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for %s: %w", name, err)
		}

		files = append(files, absPath)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no .go files found in directory %s", dir)
	}

	return files, nil
}
