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
	"strings"
	"testing"
)

func TestChangeToProjectDir_EmptyConfigPath(t *testing.T) {
	savedDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(savedDir)

	configPath = ""
	if err := changeToProjectDir(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if configPath != "" {
		t.Errorf("configPath should remain empty, got %q", configPath)
	}

	cwd, _ := os.Getwd()
	if cwd != savedDir {
		t.Errorf("CWD should not change: got %q, want %q", cwd, savedDir)
	}
}

func TestChangeToProjectDir_BareFilename(t *testing.T) {
	savedDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(savedDir)

	configPath = "forge.yaml"
	if err := changeToProjectDir(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if configPath != "forge.yaml" {
		t.Errorf("configPath should remain %q, got %q", "forge.yaml", configPath)
	}

	cwd, _ := os.Getwd()
	if cwd != savedDir {
		t.Errorf("CWD should not change: got %q, want %q", cwd, savedDir)
	}
}

func TestChangeToProjectDir_RelativeSubdirectory(t *testing.T) {
	savedDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(savedDir)

	tmpDir := t.TempDir()
	subdir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to tmpDir: %v", err)
	}

	configPath = "subdir/forge.yaml"
	if err := changeToProjectDir(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if configPath != "forge.yaml" {
		t.Errorf("configPath should be %q, got %q", "forge.yaml", configPath)
	}

	cwd, _ := os.Getwd()
	if !strings.HasSuffix(cwd, "/subdir") {
		t.Errorf("CWD should end with /subdir, got %q", cwd)
	}
}

func TestChangeToProjectDir_AbsolutePath(t *testing.T) {
	savedDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(savedDir)

	tmpDir := t.TempDir()
	// Resolve symlinks so comparisons work on systems where /tmp is a symlink.
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to eval symlinks: %v", err)
	}

	configPath = filepath.Join(tmpDir, "forge.yaml")
	if err := changeToProjectDir(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if configPath != "forge.yaml" {
		t.Errorf("configPath should be %q, got %q", "forge.yaml", configPath)
	}

	cwd, _ := os.Getwd()
	if cwd != tmpDir {
		t.Errorf("CWD should be %q, got %q", tmpDir, cwd)
	}
}

func TestChangeToProjectDir_DotslashPrefix(t *testing.T) {
	savedDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(savedDir)

	tmpDir := t.TempDir()
	subdir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to tmpDir: %v", err)
	}

	configPath = "./subdir/forge.yaml"
	if err := changeToProjectDir(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if configPath != "forge.yaml" {
		t.Errorf("configPath should be %q, got %q", "forge.yaml", configPath)
	}

	cwd, _ := os.Getwd()
	if !strings.HasSuffix(cwd, "/subdir") {
		t.Errorf("CWD should end with /subdir, got %q", cwd)
	}
}

func TestChangeToProjectDir_NonexistentDirectory(t *testing.T) {
	savedDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(savedDir)

	configPath = "nonexistent/forge.yaml"
	err = changeToProjectDir()
	if err == nil {
		t.Fatal("expected error for nonexistent directory, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should contain %q, got %q", "nonexistent", err.Error())
	}
	if configPath != "nonexistent/forge.yaml" {
		t.Errorf("configPath should remain unchanged, got %q", configPath)
	}
}

func TestChangeToProjectDir_VerifyCWDSideEffect(t *testing.T) {
	savedDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(savedDir)

	tmpDir := t.TempDir()
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to eval symlinks: %v", err)
	}

	configPath = filepath.Join(tmpDir, "forge.yaml")
	if err := changeToProjectDir(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the actual working directory changed via os.Getwd()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory after changeToProjectDir: %v", err)
	}
	if cwd != tmpDir {
		t.Errorf("os.Getwd() should return %q after changeToProjectDir, got %q", tmpDir, cwd)
	}
}

func TestParseGlobalFlags_ConfigEqualsyntax(t *testing.T) {
	// Save and restore configPath
	savedConfigPath := configPath
	defer func() { configPath = savedConfigPath }()

	configPath = ""
	args := parseGlobalFlags([]string{"--config=./myrepo/forge.yaml", "build"})
	if configPath != "./myrepo/forge.yaml" {
		t.Errorf("configPath should be %q, got %q", "./myrepo/forge.yaml", configPath)
	}
	if len(args) != 1 || args[0] != "build" {
		t.Errorf("remaining args should be [build], got %v", args)
	}
}

func TestParseGlobalFlags_ConfigSpaceSyntax(t *testing.T) {
	savedConfigPath := configPath
	defer func() { configPath = savedConfigPath }()

	configPath = ""
	args := parseGlobalFlags([]string{"--config", "./myrepo/forge.yaml", "build"})
	if configPath != "./myrepo/forge.yaml" {
		t.Errorf("configPath should be %q, got %q", "./myrepo/forge.yaml", configPath)
	}
	if len(args) != 1 || args[0] != "build" {
		t.Errorf("remaining args should be [build], got %v", args)
	}
}
