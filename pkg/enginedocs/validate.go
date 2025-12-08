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
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

// Validate checks the documentation completeness and correctness for an engine.
// It returns a slice of all validation errors found (not just the first one).
// This allows fixing all issues at once.
//
// Validation checks:
//  1. list.yaml exists and is valid YAML
//  2. Version field is "1.0"
//  3. Engine field matches cfg.EngineName
//  4. Each doc entry has non-empty: Name, Title, Description, URL
//  5. Each doc URL points to existing file (check local filesystem)
//  6. Required docs (cfg.RequiredDocs) are present in the docs array
//  7. No duplicate names in docs array
//  8. URLs must be relative paths (no absolute URLs starting with http:// or /)
func Validate(cfg Config) []error {
	var errs []error

	// Determine list.yaml path
	listPath := filepath.Join(cfg.LocalDir, "list.yaml")

	// 1. Check list.yaml exists and is valid YAML
	data, err := os.ReadFile(listPath)
	if err != nil {
		if os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("list.yaml: file not found at %s", listPath))
		} else {
			errs = append(errs, fmt.Errorf("list.yaml: failed to read file: %w", err))
		}
		// Cannot continue validation without the file
		return errs
	}

	var store DocStore
	if err := yaml.Unmarshal(data, &store); err != nil {
		errs = append(errs, fmt.Errorf("list.yaml: invalid YAML: %w", err))
		// Cannot continue validation with invalid YAML
		return errs
	}

	// 2. Version field is "1.0"
	if store.Version != "1.0" {
		errs = append(errs, fmt.Errorf("version: expected \"1.0\", got %q", store.Version))
	}

	// 3. Engine field matches cfg.EngineName
	if store.Engine != cfg.EngineName {
		errs = append(errs, fmt.Errorf("engine: expected %q, got %q", cfg.EngineName, store.Engine))
	}

	// Track seen names for duplicate detection
	seenNames := make(map[string]bool)
	// Track present doc names for required docs check
	presentDocs := make(map[string]bool)

	for i, doc := range store.Docs {
		prefix := fmt.Sprintf("docs[%d]", i)

		// 4. Each doc entry has non-empty: Name, Title, Description, URL
		if doc.Name == "" {
			errs = append(errs, fmt.Errorf("%s.name: field is required but empty", prefix))
		}
		if doc.Title == "" {
			errs = append(errs, fmt.Errorf("%s.title: field is required but empty (name=%q)", prefix, doc.Name))
		}
		if doc.Description == "" {
			errs = append(errs, fmt.Errorf("%s.description: field is required but empty (name=%q)", prefix, doc.Name))
		}
		if doc.URL == "" {
			errs = append(errs, fmt.Errorf("%s.url: field is required but empty (name=%q)", prefix, doc.Name))
		}

		// 8. URLs must be relative paths (no absolute URLs starting with http:// or /)
		if doc.URL != "" {
			if strings.HasPrefix(doc.URL, "http://") || strings.HasPrefix(doc.URL, "https://") {
				errs = append(errs, fmt.Errorf("%s.url: must be a relative path, not an absolute URL (name=%q, url=%q)", prefix, doc.Name, doc.URL))
			} else if strings.HasPrefix(doc.URL, "/") {
				errs = append(errs, fmt.Errorf("%s.url: must be a relative path, not an absolute path starting with / (name=%q, url=%q)", prefix, doc.Name, doc.URL))
			}
		}

		// 7. No duplicate names in docs array
		if doc.Name != "" {
			if seenNames[doc.Name] {
				errs = append(errs, fmt.Errorf("%s.name: duplicate name %q", prefix, doc.Name))
			}
			seenNames[doc.Name] = true
			presentDocs[doc.Name] = true
		}

		// 5. Each doc URL points to existing file (check local filesystem)
		if doc.URL != "" && !strings.HasPrefix(doc.URL, "http://") && !strings.HasPrefix(doc.URL, "https://") && !strings.HasPrefix(doc.URL, "/") {
			if _, err := os.Stat(doc.URL); err != nil {
				if os.IsNotExist(err) {
					errs = append(errs, fmt.Errorf("%s.url: file not found at %q (name=%q)", prefix, doc.URL, doc.Name))
				} else {
					errs = append(errs, fmt.Errorf("%s.url: failed to check file %q: %w (name=%q)", prefix, doc.URL, err, doc.Name))
				}
			}
		}
	}

	// 6. Required docs (cfg.RequiredDocs) are present in the docs array
	for _, requiredDoc := range cfg.RequiredDocs {
		if !presentDocs[requiredDoc] {
			errs = append(errs, fmt.Errorf("required doc %q is missing from docs array", requiredDoc))
		}
	}

	return errs
}
