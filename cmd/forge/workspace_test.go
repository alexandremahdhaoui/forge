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
	"testing"
)

// makeForgeRepo creates a minimal directory structure that passes forgepath.IsForgeRepo().
// It creates go.mod with the forge module path and cmd/forge/main.go.
func makeForgeRepo(t *testing.T, dir string) {
	t.Helper()

	goModContent := "module github.com/alexandremahdhaoui/forge\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goModContent), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	cmdDir := filepath.Join(dir, "cmd", "forge")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatalf("failed to create cmd/forge dir: %v", err)
	}

	mainContent := "package main\n\nfunc main() {}\n"
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainContent), 0o644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
}

func TestResolveWorkspace_NoGoWork(t *testing.T) {
	savedDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(savedDir)
		skipWorkspaceResolution = false
	})

	// Use an isolated temp directory with no go.work anywhere above it
	tmpDir := t.TempDir()
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to eval symlinks: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to tmpDir: %v", err)
	}

	// Clear any workspace env vars that might be set
	os.Unsetenv("FORGE_RUN_LOCAL_ENABLED")
	os.Unsetenv("FORGE_RUN_LOCAL_BASEDIR")

	err = resolveWorkspace()
	if err != nil {
		t.Fatalf("resolveWorkspace() unexpected error: %v", err)
	}

	// CWD should not change
	cwd, _ := os.Getwd()
	if cwd != tmpDir {
		t.Errorf("CWD changed unexpectedly: got %q, want %q", cwd, tmpDir)
	}

	// Env vars should NOT be set
	if val := os.Getenv("FORGE_RUN_LOCAL_ENABLED"); val != "" {
		t.Errorf("FORGE_RUN_LOCAL_ENABLED should be unset, got %q", val)
	}
	if val := os.Getenv("FORGE_RUN_LOCAL_BASEDIR"); val != "" {
		t.Errorf("FORGE_RUN_LOCAL_BASEDIR should be unset, got %q", val)
	}
}

func TestResolveWorkspace_CWDInsideUseDir(t *testing.T) {
	savedDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(savedDir)
		skipWorkspaceResolution = false
		os.Unsetenv("FORGE_RUN_LOCAL_ENABLED")
		os.Unsetenv("FORGE_RUN_LOCAL_BASEDIR")
	})

	// Create workspace structure:
	//   tmpDir/go.work         -> use ./forge-repo
	//   tmpDir/forge-repo/     -> forge repo (go.mod + cmd/forge/main.go)
	tmpDir := t.TempDir()
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to eval symlinks: %v", err)
	}

	forgeDir := filepath.Join(tmpDir, "forge-repo")
	if err := os.MkdirAll(forgeDir, 0o755); err != nil {
		t.Fatalf("failed to create forge-repo dir: %v", err)
	}
	makeForgeRepo(t, forgeDir)

	goWorkContent := "go 1.21\n\nuse ./forge-repo\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.work"), []byte(goWorkContent), 0o644); err != nil {
		t.Fatalf("failed to write go.work: %v", err)
	}

	// CWD is inside the use directory
	if err := os.Chdir(forgeDir); err != nil {
		t.Fatalf("failed to chdir to forge-repo: %v", err)
	}

	os.Unsetenv("FORGE_RUN_LOCAL_ENABLED")
	os.Unsetenv("FORGE_RUN_LOCAL_BASEDIR")

	err = resolveWorkspace()
	if err != nil {
		t.Fatalf("resolveWorkspace() unexpected error: %v", err)
	}

	// CWD should remain unchanged (already inside member)
	cwd, _ := os.Getwd()
	if cwd != forgeDir {
		t.Errorf("CWD should not change: got %q, want %q", cwd, forgeDir)
	}

	// Env vars should be set
	if val := os.Getenv("FORGE_RUN_LOCAL_ENABLED"); val != "true" {
		t.Errorf("FORGE_RUN_LOCAL_ENABLED = %q, want %q", val, "true")
	}
	if val := os.Getenv("FORGE_RUN_LOCAL_BASEDIR"); val != forgeDir {
		t.Errorf("FORGE_RUN_LOCAL_BASEDIR = %q, want %q", val, forgeDir)
	}
}

func TestResolveWorkspace_CWDIsWorkspaceRoot(t *testing.T) {
	savedDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(savedDir)
		skipWorkspaceResolution = false
		os.Unsetenv("FORGE_RUN_LOCAL_ENABLED")
		os.Unsetenv("FORGE_RUN_LOCAL_BASEDIR")
	})

	// Create workspace structure:
	//   tmpDir/go.work          -> use ./other-repo \n use ./forge-repo
	//   tmpDir/other-repo/      -> non-forge repo (just go.mod)
	//   tmpDir/forge-repo/      -> forge repo (go.mod + cmd/forge/main.go)
	tmpDir := t.TempDir()
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to eval symlinks: %v", err)
	}

	// Create non-forge repo
	otherDir := filepath.Join(tmpDir, "other-repo")
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatalf("failed to create other-repo dir: %v", err)
	}
	otherGoMod := "module github.com/test/other-repo\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(otherDir, "go.mod"), []byte(otherGoMod), 0o644); err != nil {
		t.Fatalf("failed to write other go.mod: %v", err)
	}

	// Create forge repo
	forgeDir := filepath.Join(tmpDir, "forge-repo")
	if err := os.MkdirAll(forgeDir, 0o755); err != nil {
		t.Fatalf("failed to create forge-repo dir: %v", err)
	}
	makeForgeRepo(t, forgeDir)

	goWorkContent := "go 1.21\n\nuse (\n\t./other-repo\n\t./forge-repo\n)\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.work"), []byte(goWorkContent), 0o644); err != nil {
		t.Fatalf("failed to write go.work: %v", err)
	}

	// CWD is the workspace root
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to workspace root: %v", err)
	}

	os.Unsetenv("FORGE_RUN_LOCAL_ENABLED")
	os.Unsetenv("FORGE_RUN_LOCAL_BASEDIR")

	err = resolveWorkspace()
	if err != nil {
		t.Fatalf("resolveWorkspace() unexpected error: %v", err)
	}

	// CWD should change to forge-repo member
	cwd, _ := os.Getwd()
	if cwd != forgeDir {
		t.Errorf("CWD should change to forge repo member: got %q, want %q", cwd, forgeDir)
	}

	// Env vars should be set
	if val := os.Getenv("FORGE_RUN_LOCAL_ENABLED"); val != "true" {
		t.Errorf("FORGE_RUN_LOCAL_ENABLED = %q, want %q", val, "true")
	}
	if val := os.Getenv("FORGE_RUN_LOCAL_BASEDIR"); val != forgeDir {
		t.Errorf("FORGE_RUN_LOCAL_BASEDIR = %q, want %q", val, forgeDir)
	}
}

func TestResolveWorkspace_SkipFlag(t *testing.T) {
	savedDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(savedDir)
		skipWorkspaceResolution = false
		os.Unsetenv("FORGE_RUN_LOCAL_ENABLED")
		os.Unsetenv("FORGE_RUN_LOCAL_BASEDIR")
	})

	// Create workspace structure that would normally trigger resolution
	tmpDir := t.TempDir()
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to eval symlinks: %v", err)
	}

	forgeDir := filepath.Join(tmpDir, "forge-repo")
	if err := os.MkdirAll(forgeDir, 0o755); err != nil {
		t.Fatalf("failed to create forge-repo dir: %v", err)
	}
	makeForgeRepo(t, forgeDir)

	goWorkContent := "go 1.21\n\nuse ./forge-repo\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.work"), []byte(goWorkContent), 0o644); err != nil {
		t.Fatalf("failed to write go.work: %v", err)
	}

	// CWD is the workspace root (would normally chdir to forge-repo)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to workspace root: %v", err)
	}

	os.Unsetenv("FORGE_RUN_LOCAL_ENABLED")
	os.Unsetenv("FORGE_RUN_LOCAL_BASEDIR")

	// Set the skip flag
	skipWorkspaceResolution = true

	err = resolveWorkspace()
	if err != nil {
		t.Fatalf("resolveWorkspace() unexpected error: %v", err)
	}

	// CWD should NOT change
	cwd, _ := os.Getwd()
	if cwd != tmpDir {
		t.Errorf("CWD should not change when skip flag is set: got %q, want %q", cwd, tmpDir)
	}

	// Env vars should NOT be set
	if val := os.Getenv("FORGE_RUN_LOCAL_ENABLED"); val != "" {
		t.Errorf("FORGE_RUN_LOCAL_ENABLED should be unset when skip flag is set, got %q", val)
	}
	if val := os.Getenv("FORGE_RUN_LOCAL_BASEDIR"); val != "" {
		t.Errorf("FORGE_RUN_LOCAL_BASEDIR should be unset when skip flag is set, got %q", val)
	}
}

func TestIsInsideDir(t *testing.T) {
	tests := []struct {
		name string
		path string
		dir  string
		want bool
	}{
		{
			name: "exact match",
			path: "/a/b/c",
			dir:  "/a/b/c",
			want: true,
		},
		{
			name: "subdirectory",
			path: "/a/b/c/d",
			dir:  "/a/b/c",
			want: true,
		},
		{
			name: "not inside",
			path: "/a/b/other",
			dir:  "/a/b/c",
			want: false,
		},
		{
			name: "prefix but not subdir",
			path: "/a/b/candy",
			dir:  "/a/b/can",
			want: false,
		},
		{
			name: "parent of dir",
			path: "/a/b",
			dir:  "/a/b/c",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInsideDir(tt.path, tt.dir)
			if got != tt.want {
				t.Errorf("isInsideDir(%q, %q) = %v, want %v", tt.path, tt.dir, got, tt.want)
			}
		})
	}
}

func TestParseGlobalFlags_CWDSpaceSyntax(t *testing.T) {
	savedCwd := cwdOverride
	defer func() { cwdOverride = savedCwd }()

	cwdOverride = ""
	args := parseGlobalFlags([]string{"--cwd", "/tmp/mydir", "build"})
	if cwdOverride != "/tmp/mydir" {
		t.Errorf("cwdOverride should be %q, got %q", "/tmp/mydir", cwdOverride)
	}
	if len(args) != 1 || args[0] != "build" {
		t.Errorf("remaining args should be [build], got %v", args)
	}
}

func TestParseGlobalFlags_CWDEqualsSyntax(t *testing.T) {
	savedCwd := cwdOverride
	defer func() { cwdOverride = savedCwd }()

	cwdOverride = ""
	args := parseGlobalFlags([]string{"--cwd=/tmp/mydir", "build"})
	if cwdOverride != "/tmp/mydir" {
		t.Errorf("cwdOverride should be %q, got %q", "/tmp/mydir", cwdOverride)
	}
	if len(args) != 1 || args[0] != "build" {
		t.Errorf("remaining args should be [build], got %v", args)
	}
}

func TestParseGlobalFlags_SkipWorkspaceResolution(t *testing.T) {
	savedSkip := skipWorkspaceResolution
	defer func() { skipWorkspaceResolution = savedSkip }()

	skipWorkspaceResolution = false
	args := parseGlobalFlags([]string{"--skip-workspace-resolution", "build"})
	if !skipWorkspaceResolution {
		t.Error("skipWorkspaceResolution should be true after parsing --skip-workspace-resolution")
	}
	if len(args) != 1 || args[0] != "build" {
		t.Errorf("remaining args should be [build], got %v", args)
	}
}
