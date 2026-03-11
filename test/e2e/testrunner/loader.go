//go:build e2e || unit

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

package testrunner

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadTestFiles recursively discovers all *.yaml files under dir using
// filepath.WalkDir and parses each into a TestFile. It returns the parsed
// test files, their file paths (for naming test cases), and any error
// encountered.
func LoadTestFiles(dir string) ([]TestFile, []string, error) {
	var allPaths []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".yaml") {
			allPaths = append(allPaths, path)
		}
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("walking %q: %w", dir, err)
	}

	if len(allPaths) == 0 {
		return nil, nil, fmt.Errorf("no YAML files found in %q", dir)
	}

	var testFiles []TestFile
	var filePaths []string
	for _, path := range allPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, fmt.Errorf("reading %q: %w", path, err)
		}

		var tf TestFile
		if err := yaml.Unmarshal(data, &tf); err != nil {
			return nil, nil, fmt.Errorf("parsing %q: %w", path, err)
		}

		testFiles = append(testFiles, tf)
		filePaths = append(filePaths, path)
	}

	return testFiles, filePaths, nil
}
