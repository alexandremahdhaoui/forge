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
	"path/filepath"
	"sort"
	"testing"
)

// createTempFile is a helper function to create test files.
func createTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestHasLicenseHeader(t *testing.T) {
	t.Run("file_with_copyright", func(t *testing.T) {
		dir := t.TempDir()
		path := createTempFile(t, dir, "test.go", `// Copyright 2024 Example Corp
package main

func main() {}
`)
		hasLicense, err := hasLicenseHeader(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !hasLicense {
			t.Error("expected hasLicenseHeader to return true for file with copyright")
		}
	})

	t.Run("file_with_spdx", func(t *testing.T) {
		dir := t.TempDir()
		path := createTempFile(t, dir, "test.go", `// SPDX-License-Identifier: Apache-2.0
package main

func main() {}
`)
		hasLicense, err := hasLicenseHeader(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !hasLicense {
			t.Error("expected hasLicenseHeader to return true for file with SPDX identifier")
		}
	})

	t.Run("file_with_licensed_under", func(t *testing.T) {
		dir := t.TempDir()
		path := createTempFile(t, dir, "test.go", `// Licensed under the MIT License
package main

func main() {}
`)
		hasLicense, err := hasLicenseHeader(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !hasLicense {
			t.Error("expected hasLicenseHeader to return true for file with 'Licensed under'")
		}
	})

	t.Run("file_without_license", func(t *testing.T) {
		dir := t.TempDir()
		path := createTempFile(t, dir, "test.go", `package main

func main() {}
`)
		hasLicense, err := hasLicenseHeader(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hasLicense {
			t.Error("expected hasLicenseHeader to return false for file without license")
		}
	})

	t.Run("empty_file", func(t *testing.T) {
		dir := t.TempDir()
		path := createTempFile(t, dir, "test.go", "")

		hasLicense, err := hasLicenseHeader(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hasLicense {
			t.Error("expected hasLicenseHeader to return false for empty file")
		}
	})

	t.Run("license_after_package", func(t *testing.T) {
		dir := t.TempDir()
		path := createTempFile(t, dir, "test.go", `package main

// Copyright 2024 Example Corp
func main() {}
`)
		hasLicense, err := hasLicenseHeader(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hasLicense {
			t.Error("expected hasLicenseHeader to return false when license is after package declaration")
		}
	})
}

func TestFindGoFiles(t *testing.T) {
	t.Run("finds_go_files", func(t *testing.T) {
		dir := t.TempDir()
		aPath := createTempFile(t, dir, "a.go", "package main\n")
		bPath := createTempFile(t, dir, "b.go", "package main\n")

		files, err := findGoFiles(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Sort for consistent comparison
		sort.Strings(files)
		expected := []string{aPath, bPath}
		sort.Strings(expected)

		if len(files) != len(expected) {
			t.Fatalf("expected %d files, got %d: %v", len(expected), len(files), files)
		}
		for i, f := range files {
			if f != expected[i] {
				t.Errorf("expected file %s, got %s", expected[i], f)
			}
		}
	})

	t.Run("skips_vendor", func(t *testing.T) {
		dir := t.TempDir()
		createTempFile(t, dir, "vendor/c.go", "package vendor\n")
		createTempFile(t, dir, "main.go", "package main\n")

		files, err := findGoFiles(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, f := range files {
			if filepath.Base(f) == "c.go" {
				t.Error("expected vendor/c.go to be skipped")
			}
		}
		if len(files) != 1 {
			t.Errorf("expected 1 file, got %d: %v", len(files), files)
		}
	})

	t.Run("skips_git", func(t *testing.T) {
		dir := t.TempDir()
		createTempFile(t, dir, ".git/hooks/d.go", "package hooks\n")
		createTempFile(t, dir, "main.go", "package main\n")

		files, err := findGoFiles(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, f := range files {
			if filepath.Base(f) == "d.go" {
				t.Error("expected .git/hooks/d.go to be skipped")
			}
		}
		if len(files) != 1 {
			t.Errorf("expected 1 file, got %d: %v", len(files), files)
		}
	})

	t.Run("skips_generated", func(t *testing.T) {
		dir := t.TempDir()
		createTempFile(t, dir, "generated.go", "// Code generated by tool. DO NOT EDIT.\npackage main\n")
		createTempFile(t, dir, "main.go", "package main\n")

		files, err := findGoFiles(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, f := range files {
			if filepath.Base(f) == "generated.go" {
				t.Error("expected generated.go to be skipped (generated files should not be in returned slice)")
			}
		}
		if len(files) != 1 {
			t.Errorf("expected 1 file (only main.go), got %d: %v", len(files), files)
		}
	})

	t.Run("finds_nested", func(t *testing.T) {
		dir := t.TempDir()
		ePath := createTempFile(t, dir, "sub/e.go", "package sub\n")

		files, err := findGoFiles(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d: %v", len(files), files)
		}
		if files[0] != ePath {
			t.Errorf("expected %s, got %s", ePath, files[0])
		}
	})
}

func TestVerifyLicenses(t *testing.T) {
	t.Run("all_pass", func(t *testing.T) {
		dir := t.TempDir()
		createTempFile(t, dir, "a.go", "// Copyright 2024 Example Corp\npackage main\n")
		createTempFile(t, dir, "b.go", "// SPDX-License-Identifier: Apache-2.0\npackage main\n")

		violations, total, err := verifyLicenses(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(violations) != 0 {
			t.Errorf("expected 0 violations, got %d: %v", len(violations), violations)
		}
		if total != 2 {
			t.Errorf("expected total 2, got %d", total)
		}
	})

	t.Run("some_fail", func(t *testing.T) {
		dir := t.TempDir()
		createTempFile(t, dir, "licensed.go", "// Copyright 2024 Example Corp\npackage main\n")
		noLicensePath := createTempFile(t, dir, "no_license.go", "package main\n")

		violations, total, err := verifyLicenses(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(violations) != 1 {
			t.Errorf("expected 1 violation, got %d: %v", len(violations), violations)
		}
		if len(violations) > 0 && violations[0] != noLicensePath {
			t.Errorf("expected violation for %s, got %s", noLicensePath, violations[0])
		}
		if total != 2 {
			t.Errorf("expected total 2, got %d", total)
		}
	})

	t.Run("empty_dir", func(t *testing.T) {
		dir := t.TempDir()

		violations, total, err := verifyLicenses(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(violations) != 0 {
			t.Errorf("expected 0 violations, got %d: %v", len(violations), violations)
		}
		if total != 0 {
			t.Errorf("expected total 0, got %d", total)
		}
	})
}
