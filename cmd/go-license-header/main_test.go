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

// The scan/detect/write logic (idempotency, year preservation, exclude
// dirs, generated-file skip) is tested once in internal/licenseheader,
// shared with rust-license-header and go-lint-licenses. This file only
// covers this engine's own wiring: building a real Spec and running it end
// to end against a real directory, and that it only ever touches .go files.

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

func TestBuildAddsHeaderToGoFileUsingSrcFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lib.go")
	if err := os.WriteFile(path, []byte("package lib\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	spec := &Spec{Holder: "Alexandre Mahdhaoui"}
	input := mcptypes.BuildInput{Src: dir}

	artifact, err := Build(context.Background(), input, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if artifact == nil {
		t.Fatal("expected a non-nil artifact")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(got), "Copyright") {
		t.Fatal("expected a license header to be added")
	}
	if !strings.Contains(string(got), "Alexandre Mahdhaoui") {
		t.Fatal("expected the configured holder in the header")
	}
}

func TestBuildOnlyTouchesGoFiles(t *testing.T) {
	dir := t.TempDir()
	goPath := filepath.Join(dir, "lib.go")
	rsPath := filepath.Join(dir, "lib.rs")
	if err := os.WriteFile(goPath, []byte("package lib\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(rsPath, []byte("use std::fmt;\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	spec := &Spec{Holder: "Alexandre Mahdhaoui"}
	input := mcptypes.BuildInput{Src: dir}

	if _, err := Build(context.Background(), input, spec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	goContent, _ := os.ReadFile(goPath)
	if !strings.Contains(string(goContent), "Copyright") {
		t.Fatal("expected .go file to get a header")
	}
	rsContent, _ := os.ReadFile(rsPath)
	if strings.Contains(string(rsContent), "Copyright") {
		t.Fatal("expected .rs file to be left untouched by go-license-header")
	}
}
