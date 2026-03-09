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

func TestParseGitRepoURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "SSH with .git suffix",
			input: "git@github.com:alexandremahdhaoui/forge-workspace.git",
			want:  "github.com/alexandremahdhaoui/forge-workspace",
		},
		{
			name:  "SSH without .git suffix",
			input: "git@github.com:user/repo",
			want:  "github.com/user/repo",
		},
		{
			name:  "HTTPS without .git suffix",
			input: "https://github.com/alexandremahdhaoui/something-else",
			want:  "github.com/alexandremahdhaoui/something-else",
		},
		{
			name:  "HTTPS with .git suffix",
			input: "https://github.com/user/repo.git",
			want:  "github.com/user/repo",
		},
		{
			name:  "SSH protocol with .git suffix",
			input: "ssh://git@github.com/user/repo.git",
			want:  "github.com/user/repo",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "not a URL",
			input:   "not-a-url",
			wantErr: true,
		},
		{
			name:    "unsupported scheme",
			input:   "ftp://example.com/repo",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGitRepoURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseGitRepoURL(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseGitRepoURL(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("parseGitRepoURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsGitURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "SSH git URL",
			input: "git@github.com:user/repo.git",
			want:  true,
		},
		{
			name:  "HTTPS URL",
			input: "https://github.com/user/repo",
			want:  true,
		},
		{
			name:  "HTTP URL",
			input: "http://github.com/user/repo",
			want:  true,
		},
		{
			name:  "SSH protocol URL",
			input: "ssh://git@github.com/user/repo",
			want:  true,
		},
		{
			name:  "dot",
			input: ".",
			want:  false,
		},
		{
			name:  "relative path",
			input: "./some/path",
			want:  false,
		},
		{
			name:  "absolute path",
			input: "/absolute/path",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGitURL(tt.input)
			if got != tt.want {
				t.Errorf("isGitURL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveContextDir_EmptyAndDot(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	tests := []struct {
		name  string
		input string
	}{
		{name: "empty string", input: ""},
		{name: "dot", input: "."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, cleanup, err := resolveContextDir(tt.input)
			if err != nil {
				t.Fatalf("resolveContextDir(%q) unexpected error: %v", tt.input, err)
			}
			if cleanup == nil {
				t.Fatal("cleanup function should not be nil")
			}
			// Calling cleanup should not panic.
			cleanup()

			if dir != cwd {
				t.Errorf("resolveContextDir(%q) = %q, want CWD %q", tt.input, dir, cwd)
			}
		})
	}
}

func TestResolveContextDir_RelativePath(t *testing.T) {
	savedDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(savedDir) })

	tmpDir := t.TempDir()
	// Resolve symlinks so comparisons work on systems where /tmp is a symlink.
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to eval symlinks: %v", err)
	}

	// Create a subdirectory inside tmpDir.
	subDir := filepath.Join(tmpDir, "myproject")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// Change into tmpDir so that ./myproject is a valid relative path.
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to tmpDir: %v", err)
	}

	t.Run("valid relative path", func(t *testing.T) {
		dir, cleanup, err := resolveContextDir("./myproject")
		if err != nil {
			t.Fatalf("resolveContextDir(\"./myproject\") unexpected error: %v", err)
		}
		cleanup()

		if dir != subDir {
			t.Errorf("resolveContextDir(\"./myproject\") = %q, want %q", dir, subDir)
		}
	})

	t.Run("nonexistent relative path", func(t *testing.T) {
		_, _, err := resolveContextDir("./nonexistent")
		if err == nil {
			t.Fatal("resolveContextDir(\"./nonexistent\") expected error, got nil")
		}
	})
}

func TestResolveContextDir_AbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	// Resolve symlinks so comparisons work on systems where /tmp is a symlink.
	var err error
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to eval symlinks: %v", err)
	}

	t.Run("valid absolute path", func(t *testing.T) {
		dir, cleanup, err := resolveContextDir(tmpDir)
		if err != nil {
			t.Fatalf("resolveContextDir(%q) unexpected error: %v", tmpDir, err)
		}
		cleanup()

		if dir != tmpDir {
			t.Errorf("resolveContextDir(%q) = %q, want %q", tmpDir, dir, tmpDir)
		}
	})

	t.Run("nonexistent absolute path", func(t *testing.T) {
		_, _, err := resolveContextDir("/tmp/nonexistent-forge-test-path-12345")
		if err == nil {
			t.Fatal("resolveContextDir(\"/tmp/nonexistent-forge-test-path-12345\") expected error, got nil")
		}
	})
}

func TestResolveViaGoWork(t *testing.T) {
	savedDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(savedDir) })

	// Create a temp workspace structure:
	//   tmpdir/go.work        -> use ./member-a
	//   tmpdir/member-a/go.mod -> module github.com/test/member-a
	tmpDir := t.TempDir()
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to eval symlinks: %v", err)
	}

	goWorkContent := "go 1.21\n\nuse ./member-a\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.work"), []byte(goWorkContent), 0o644); err != nil {
		t.Fatalf("failed to write go.work: %v", err)
	}

	memberDir := filepath.Join(tmpDir, "member-a")
	if err := os.Mkdir(memberDir, 0o755); err != nil {
		t.Fatalf("failed to create member-a dir: %v", err)
	}

	goModContent := "module github.com/test/member-a\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(memberDir, "go.mod"), []byte(goModContent), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Change to tmpdir so that FindGoWork can discover go.work.
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to tmpDir: %v", err)
	}

	t.Run("matching module path", func(t *testing.T) {
		dir, ok := resolveViaGoWork("github.com/test/member-a")
		if !ok {
			t.Fatal("resolveViaGoWork(\"github.com/test/member-a\") returned false, want true")
		}
		if dir != memberDir {
			t.Errorf("resolveViaGoWork(\"github.com/test/member-a\") = %q, want %q", dir, memberDir)
		}
	})

	t.Run("nonexistent module path", func(t *testing.T) {
		dir, ok := resolveViaGoWork("github.com/test/nonexistent")
		if ok {
			t.Errorf("resolveViaGoWork(\"github.com/test/nonexistent\") returned true with dir %q, want false", dir)
		}
		if dir != "" {
			t.Errorf("resolveViaGoWork(\"github.com/test/nonexistent\") dir = %q, want empty", dir)
		}
	})
}
