//go:build unit

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

package licenseheader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestHasHeaderDetectsExistingHeader(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "lib.rs", "// Copyright 2020 Someone\n//\n// Licensed under the Apache License.\n\npub fn foo() {}\n")

	got, err := HasHeader(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatal("expected header to be detected")
	}
}

// hack/boilerplate.go.txt is a /* */ block, and three files carrying it
// verbatim read as unlicensed before the walk spoke that form.
func TestHasHeaderDetectsTheBoilerplateBlockForm(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "main.go",
		"/*\nCopyright 2024 Alexandre Mahdhaoui\n\nLicensed under the Apache License.\n*/\n\npackage main\n")

	got, err := HasHeader(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatal("expected the block-comment boilerplate to be detected")
	}
}

// A block comment that never mentions a license is still not a header, and
// the code after it is still where the search ends.
func TestHasHeaderABlockCommentWithoutALicenseIsNotOne(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "main.go",
		"/*\nA small helper module.\n*/\n\npackage main\n")

	got, err := HasHeader(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatal("expected a non-license block comment to not be treated as a header")
	}
}

func TestHasHeaderNoHeader(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "lib.rs", "use std::fmt;\n\npub fn foo() {}\n")

	got, err := HasHeader(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatal("expected no header to be detected")
	}
}

func TestHasHeaderStopsAtCode(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "lib.rs", "// A small helper module.\n\npub fn foo() {}\n")

	got, err := HasHeader(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatal("expected a non-license comment to not be treated as a header")
	}
}

func TestHasHeaderNoFixedLineCap(t *testing.T) {
	dir := t.TempDir()
	// Two build-constraint lines plus a full 13-line Apache header pushes
	// "Copyright" past where a 15-line cap would have stopped looking.
	content := "//go:build unit\n// +build unit\n\n" + ApacheHeader("Example Corp", 2024) + "package main\n"
	path := writeFile(t, dir, "lib.go", content)

	got, err := HasHeader(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatal("expected header to be found despite long comment preamble")
	}
}

func TestIsGeneratedFile(t *testing.T) {
	dir := t.TempDir()
	generated := writeFile(t, dir, "zz_generated.rs", "// Code generated. DO NOT EDIT.\n\npub fn foo() {}\n")
	normal := writeFile(t, dir, "lib.rs", "use std::fmt;\n")

	got, err := IsGeneratedFile(generated)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatal("expected generated file to be detected")
	}

	got, err = IsGeneratedFile(normal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatal("expected normal file to not be marked generated")
	}
}

func TestAddHeadersAddsHeaderAndIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "lib.rs", "use std::fmt;\n\npub fn foo() {}\n")

	stats, err := AddHeaders(dir, "Alexandre Mahdhaoui", []string{".rs"}, DefaultExcludeDirs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Added != 1 {
		t.Fatalf("expected 1 header added, got %d", stats.Added)
	}

	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(first), "Copyright") {
		t.Fatal("expected header to be present after first run")
	}
	if !strings.Contains(string(first), "Alexandre Mahdhaoui") {
		t.Fatal("expected holder name in header")
	}
	if !strings.Contains(string(first), "use std::fmt;") {
		t.Fatal("expected original content to be preserved")
	}

	stats, err = AddHeaders(dir, "Alexandre Mahdhaoui", []string{".rs"}, DefaultExcludeDirs())
	if err != nil {
		t.Fatalf("unexpected error on second run: %v", err)
	}
	if stats.Added != 0 {
		t.Fatalf("expected 0 headers added on second run, got %d", stats.Added)
	}
	if stats.AlreadyLicensed != 1 {
		t.Fatalf("expected 1 file already licensed on second run, got %d", stats.AlreadyLicensed)
	}

	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(first) != string(second) {
		t.Fatal("expected file content to be unchanged after a second run")
	}
}

func TestAddHeadersPreservesExistingYear(t *testing.T) {
	dir := t.TempDir()
	original := "// Copyright 2020 Someone Else\n//\n// Licensed under the Apache License, Version 2.0.\n\npub fn foo() {}\n"
	path := writeFile(t, dir, "lib.rs", original)

	stats, err := AddHeaders(dir, "Alexandre Mahdhaoui", []string{".rs"}, DefaultExcludeDirs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Added != 0 {
		t.Fatalf("expected existing header to be left alone, got %d added", stats.Added)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != original {
		t.Fatal("expected file with an existing header to be completely untouched, including its year and holder")
	}
}

func TestAddHeadersRespectsExtensionFilter(t *testing.T) {
	dir := t.TempDir()
	rsPath := writeFile(t, dir, "lib.rs", "use std::fmt;\n")
	goPath := writeFile(t, dir, "lib.go", "package main\n")

	stats, err := AddHeaders(dir, "Alexandre Mahdhaoui", []string{".go"}, DefaultExcludeDirs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Total != 1 || stats.Added != 1 {
		t.Fatalf("expected only the .go file to be touched, got total=%d added=%d", stats.Total, stats.Added)
	}

	rsContent, _ := os.ReadFile(rsPath)
	if strings.Contains(string(rsContent), "Copyright") {
		t.Fatal("expected .rs file to be untouched when filtering for .go")
	}
	goContent, _ := os.ReadFile(goPath)
	if !strings.Contains(string(goContent), "Copyright") {
		t.Fatal("expected .go file to get a header")
	}
}

func TestAddHeadersSkipsExcludedDirs(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, filepath.Join("target", "debug", "lib.rs"), "use std::fmt;\n")

	stats, err := AddHeaders(dir, "Alexandre Mahdhaoui", []string{".rs"}, DefaultExcludeDirs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Total != 0 {
		t.Fatalf("expected files under target/ to be skipped entirely, scanned %d", stats.Total)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(got), "Copyright") {
		t.Fatal("expected excluded file to be untouched")
	}
}

func TestAddHeadersSkipsGeneratedFiles(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "zz_generated.rs", "// Code generated. DO NOT EDIT.\n\npub fn foo() {}\n")

	stats, err := AddHeaders(dir, "Alexandre Mahdhaoui", []string{".rs"}, DefaultExcludeDirs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.GeneratedSkipped != 1 {
		t.Fatalf("expected 1 generated file skipped, got %d", stats.GeneratedSkipped)
	}
	if stats.Added != 0 {
		t.Fatalf("expected generated file to not get a header, got %d added", stats.Added)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(got), "Copyright") {
		t.Fatal("expected generated file to be untouched")
	}
}

func TestFindFilesFindsMatchingExtensions(t *testing.T) {
	dir := t.TempDir()
	aPath := writeFile(t, dir, "a.go", "package main\n")
	writeFile(t, dir, "b.rs", "use std::fmt;\n")

	files, err := FindFiles(dir, []string{".go"}, DefaultExcludeDirs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || files[0] != aPath {
		t.Fatalf("expected only %s, got %v", aPath, files)
	}
}

func TestFindFilesSkipsGenerated(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "zz_generated.go", "// Code generated. DO NOT EDIT.\n\npackage main\n")
	bPath := writeFile(t, dir, "b.go", "package main\n")

	files, err := FindFiles(dir, []string{".go"}, DefaultExcludeDirs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || files[0] != bPath {
		t.Fatalf("expected only %s, got %v", bPath, files)
	}
}
