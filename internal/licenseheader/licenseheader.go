// Copyright 2026 Alexandre Mahdhaoui
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

// Package licenseheader holds the language-agnostic core shared by every
// forge license-header engine: go-lint-licenses (check), go-license-header
// (fix Go files), and rust-license-header (fix Rust files). All three
// languages this package supports use `//` line comments, so one
// implementation covers all of them; only the file extension differs.
package licenseheader

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultExcludeDirs are directory names skipped by every language.
func DefaultExcludeDirs() map[string]bool {
	return map[string]bool{
		"target":       true,
		"vendor":       true,
		".git":         true,
		"build":        true,
		"tmp":          true,
		".tmp":         true,
		"node_modules": true,
	}
}

// HasHeader reports whether a file already carries a license header.
//
// It reads the leading run of blank lines and `//` line comments, since a
// real license header always sits before any code. The scan stops at the
// first line that is neither blank nor a `//` comment (a package
// declaration, an import, a build tag, whatever comes first), and returns
// true if any comment line in that leading run contains "Copyright",
// "SPDX-License-Identifier", or "Licensed under".
//
// There is no fixed line-count cap. A cap sized for a bare header plus one
// build tag false-negatives the moment a file has slightly more preamble: a
// second build-constraint line, a short doc comment above the license
// block. Stopping at "where the comments actually end" instead can't
// under-read a real header no matter how much preamble precedes it.
func HasHeader(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "//") {
			// Reached real code with no header seen yet.
			return false, nil
		}

		if strings.Contains(line, "Copyright") ||
			strings.Contains(line, "SPDX-License-Identifier") ||
			strings.Contains(line, "Licensed under") {
			return true, nil
		}
	}

	return false, scanner.Err()
}

// IsGeneratedFile reports whether a file's first non-empty line marks it as
// generated ("// Code generated ..."), the convention Go tooling uses and
// that this project also applies to Rust for consistency.
func IsGeneratedFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		return strings.HasPrefix(line, "// Code generated"), nil
	}

	return false, scanner.Err()
}

// ApacheHeader renders an Apache License 2.0 header as `//` line comments,
// followed by a blank line.
func ApacheHeader(holder string, year int) string {
	return fmt.Sprintf(`// Copyright %d %s
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

`, year, holder)
}

// PrependHeader writes the Apache License 2.0 header followed by the file's
// original content, unchanged.
func PrependHeader(path, holder string, year int) error {
	original, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	updated := append([]byte(ApacheHeader(holder, year)), original...)

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	return os.WriteFile(path, updated, info.Mode())
}

// Stats reports what AddHeaders did.
type Stats struct {
	Total            int
	AlreadyLicensed  int
	GeneratedSkipped int
	Added            int
}

// AddHeaders walks rootDir, finds files whose extension is in extensions,
// and prepends an Apache License 2.0 header to any file that doesn't
// already have one. A file that already has a header, or that is marked
// generated, is never touched. Safe to run repeatedly: a second run over
// the same tree always reports zero added.
func AddHeaders(rootDir, holder string, extensions []string, exclude map[string]bool) (Stats, error) {
	var stats Stats

	year := time.Now().Year()

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if exclude[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		if !hasAnySuffix(path, extensions) {
			return nil
		}

		stats.Total++

		generated, err := IsGeneratedFile(path)
		if err != nil {
			return fmt.Errorf("checking %s: %w", path, err)
		}
		if generated {
			stats.GeneratedSkipped++
			return nil
		}

		has, err := HasHeader(path)
		if err != nil {
			return fmt.Errorf("checking %s: %w", path, err)
		}
		if has {
			stats.AlreadyLicensed++
			return nil
		}

		if err := PrependHeader(path, holder, year); err != nil {
			return fmt.Errorf("writing header to %s: %w", path, err)
		}
		stats.Added++

		return nil
	})
	if err != nil {
		return stats, err
	}

	return stats, nil
}

func hasAnySuffix(path string, extensions []string) bool {
	for _, ext := range extensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// FindFiles recursively finds every file with an extension in extensions,
// skipping the given directories and files marked generated.
func FindFiles(root string, extensions []string, exclude map[string]bool) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if exclude[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		if !hasAnySuffix(path, extensions) {
			return nil
		}

		generated, err := IsGeneratedFile(path)
		if err != nil {
			// Match the historical behavior: include the file rather than
			// silently drop it from the check when it can't be classified.
			files = append(files, path)
			return nil
		}
		if !generated {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}
