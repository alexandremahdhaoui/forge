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

package gosource_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandremahdhaoui/forge/internal/gosource"
)

func write(t *testing.T, dir, name string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte("package p\n"), 0o600); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

func names(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, filepath.Base(p))
	}

	return out
}

// A package is every file in it. Answering with the first one is what left
// forge-ci tracking reconcilecontroller/index.go and missing the other three.
func TestEveryNonTestFileOfThePackage(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go")
	write(t, dir, "m.go")
	write(t, dir, "z.go")

	files, err := gosource.ListGoFiles(dir)
	if err != nil {
		t.Fatalf("ListGoFiles: %v", err)
	}

	got := names(files)
	if len(got) != 3 {
		t.Fatalf("expected all 3 files, got %v", got)
	}

	for _, want := range []string{"a.go", "m.go", "z.go"} {
		if !contains(got, want) {
			t.Errorf("%s is missing from %v", want, got)
		}
	}
}

// Test files are not compiled into the artifact, so an edit to one must not
// invalidate a build that does not contain it.
func TestTestFilesAreExcluded(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go")
	write(t, dir, "a_test.go")

	files, err := gosource.ListGoFiles(dir)
	if err != nil {
		t.Fatalf("ListGoFiles: %v", err)
	}

	if got := names(files); len(got) != 1 || got[0] != "a.go" {
		t.Errorf("expected only a.go, got %v", got)
	}
}

// Subdirectories are other packages and answering with their files would
// merge two packages into one.
func TestSubdirectoriesAreNotThisPackage(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go")

	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	write(t, sub, "b.go")

	files, err := gosource.ListGoFiles(dir)
	if err != nil {
		t.Fatalf("ListGoFiles: %v", err)
	}

	if got := names(files); len(got) != 1 || got[0] != "a.go" {
		t.Errorf("expected only a.go, got %v", got)
	}
}

// The paths are handed to os.Stat by callers that have their own working
// directory, so a relative one would resolve somewhere else.
func TestPathsAreAbsolute(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go")

	files, err := gosource.ListGoFiles(dir)
	if err != nil {
		t.Fatalf("ListGoFiles: %v", err)
	}

	if !filepath.IsAbs(files[0]) {
		t.Errorf("expected an absolute path, got %q", files[0])
	}
}

// A directory with no Go files is a broken import, not an empty package.
func TestADirectoryWithNoGoFilesIsAnError(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "README.md")

	if _, err := gosource.ListGoFiles(dir); err == nil {
		t.Error("expected an error for a directory holding no .go files")
	}
}

func TestAMissingDirectoryIsAnError(t *testing.T) {
	if _, err := gosource.ListGoFiles(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("expected an error for a directory that does not exist")
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}

	return false
}
