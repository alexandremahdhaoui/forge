//go:build unit

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

// TestExtractBpf2goOptions tests the extractBpf2goOptions function.
func TestExtractBpf2goOptions(t *testing.T) {
	tests := []struct {
		name              string
		spec              map[string]any
		dest              string
		wantIdent         string
		wantBpf2goVersion string
		wantGoPackage     string
		wantOutputStem    string
		wantTags          []string
		wantTypes         []string
		wantCFlags        []string
		wantCC            string
		wantErr           bool
		wantErrSubstr     string
	}{
		{
			name:              "minimal spec with just ident",
			spec:              map[string]any{"ident": "myapp"},
			dest:              "/output/bpf",
			wantIdent:         "myapp",
			wantBpf2goVersion: "latest",
			wantGoPackage:     "bpf",
			wantOutputStem:    "zz_generated",
			wantTags:          []string{"linux"},
			wantTypes:         nil,
			wantCFlags:        nil,
			wantCC:            "",
			wantErr:           false,
		},
		{
			name:          "missing required ident returns error",
			spec:          map[string]any{},
			dest:          "/output/bpf",
			wantErr:       true,
			wantErrSubstr: "ident",
		},
		{
			name:          "nil spec returns error",
			spec:          nil,
			dest:          "/output/bpf",
			wantErr:       true,
			wantErrSubstr: "ident",
		},
		{
			name: "all fields populated",
			spec: map[string]any{
				"ident":         "tracing",
				"bpf2goVersion": "v0.12.3",
				"goPackage":     "bpftracing",
				"outputStem":    "gen_bpf",
				"tags":          []any{"linux", "amd64"},
				"types":         []any{"event", "config"},
				"cflags":        []any{"-O2", "-g", "-Wall"},
				"cc":            "clang-15",
			},
			dest:              "/output/pkg",
			wantIdent:         "tracing",
			wantBpf2goVersion: "v0.12.3",
			wantGoPackage:     "bpftracing",
			wantOutputStem:    "gen_bpf",
			wantTags:          []string{"linux", "amd64"},
			wantTypes:         []string{"event", "config"},
			wantCFlags:        []string{"-O2", "-g", "-Wall"},
			wantCC:            "clang-15",
			wantErr:           false,
		},
		{
			name: "goPackage defaults to dest basename",
			spec: map[string]any{
				"ident": "prog",
			},
			dest:              "/path/to/mypackage",
			wantIdent:         "prog",
			wantBpf2goVersion: "latest",
			wantGoPackage:     "mypackage",
			wantOutputStem:    "zz_generated",
			wantTags:          []string{"linux"},
			wantTypes:         nil,
			wantCFlags:        nil,
			wantCC:            "",
			wantErr:           false,
		},
		{
			name: "tags as []string",
			spec: map[string]any{
				"ident": "app",
				"tags":  []string{"linux", "arm64"},
			},
			dest:              "/out",
			wantIdent:         "app",
			wantBpf2goVersion: "latest",
			wantGoPackage:     "out",
			wantOutputStem:    "zz_generated",
			wantTags:          []string{"linux", "arm64"},
			wantTypes:         nil,
			wantCFlags:        nil,
			wantCC:            "",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractBpf2goOptions(tt.spec, tt.dest)

			if tt.wantErr {
				if err == nil {
					t.Errorf("extractBpf2goOptions() expected error, got nil")
					return
				}
				if tt.wantErrSubstr != "" && !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("extractBpf2goOptions() error = %v, want error containing %q", err, tt.wantErrSubstr)
				}
				return
			}

			if err != nil {
				t.Errorf("extractBpf2goOptions() unexpected error: %v", err)
				return
			}

			if got.Ident != tt.wantIdent {
				t.Errorf("Ident = %q, want %q", got.Ident, tt.wantIdent)
			}
			if got.Bpf2goVersion != tt.wantBpf2goVersion {
				t.Errorf("Bpf2goVersion = %q, want %q", got.Bpf2goVersion, tt.wantBpf2goVersion)
			}
			if got.GoPackage != tt.wantGoPackage {
				t.Errorf("GoPackage = %q, want %q", got.GoPackage, tt.wantGoPackage)
			}
			if got.OutputStem != tt.wantOutputStem {
				t.Errorf("OutputStem = %q, want %q", got.OutputStem, tt.wantOutputStem)
			}
			if got.CC != tt.wantCC {
				t.Errorf("CC = %q, want %q", got.CC, tt.wantCC)
			}

			// Check tags
			if len(got.Tags) != len(tt.wantTags) {
				t.Errorf("Tags length = %d, want %d", len(got.Tags), len(tt.wantTags))
			} else {
				for i, tag := range got.Tags {
					if tag != tt.wantTags[i] {
						t.Errorf("Tags[%d] = %q, want %q", i, tag, tt.wantTags[i])
					}
				}
			}

			// Check types
			if len(got.Types) != len(tt.wantTypes) {
				t.Errorf("Types length = %d, want %d", len(got.Types), len(tt.wantTypes))
			} else {
				for i, typ := range got.Types {
					if typ != tt.wantTypes[i] {
						t.Errorf("Types[%d] = %q, want %q", i, typ, tt.wantTypes[i])
					}
				}
			}

			// Check cflags
			if len(got.CFlags) != len(tt.wantCFlags) {
				t.Errorf("CFlags length = %d, want %d", len(got.CFlags), len(tt.wantCFlags))
			} else {
				for i, flag := range got.CFlags {
					if flag != tt.wantCFlags[i] {
						t.Errorf("CFlags[%d] = %q, want %q", i, flag, tt.wantCFlags[i])
					}
				}
			}
		})
	}
}

// TestBuildBpf2goArgs tests the buildBpf2goArgs function.
func TestBuildBpf2goArgs(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		dest      string
		opts      *Bpf2goOptions
		wantArgs  []string
		checkArgs func(t *testing.T, args []string)
	}{
		{
			name: "minimal options with defaults",
			src:  "./bpf/prog.c",
			dest: "./bpf",
			opts: &Bpf2goOptions{
				Ident:         "prog",
				Bpf2goVersion: "latest",
				GoPackage:     "bpf",
				OutputStem:    "zz_generated",
				Tags:          []string{"linux"},
				Types:         nil,
				CFlags:        nil,
				CC:            "",
			},
			wantArgs: []string{
				"--go-package", "bpf",
				"--output-dir", "./bpf",
				"--output-stem", "zz_generated",
				"--tags", "linux",
				"prog",
				"./bpf/prog.c",
			},
		},
		{
			name: "multiple tags joined with comma",
			src:  "./bpf/app.c",
			dest: "./output",
			opts: &Bpf2goOptions{
				Ident:         "app",
				Bpf2goVersion: "v0.12.3",
				GoPackage:     "bpfapp",
				OutputStem:    "gen",
				Tags:          []string{"linux", "amd64", "cgo"},
				Types:         nil,
				CFlags:        nil,
				CC:            "",
			},
			checkArgs: func(t *testing.T, args []string) {
				// Find --tags and verify it's comma-joined
				for i, arg := range args {
					if arg == "--tags" && i+1 < len(args) {
						if args[i+1] != "linux,amd64,cgo" {
							t.Errorf("tags value = %q, want %q", args[i+1], "linux,amd64,cgo")
						}
						return
					}
				}
				t.Errorf("--tags flag not found in args: %v", args)
			},
		},
		{
			name: "types are added with --type flag each",
			src:  "./bpf/trace.c",
			dest: "./pkg/bpf",
			opts: &Bpf2goOptions{
				Ident:         "trace",
				Bpf2goVersion: "latest",
				GoPackage:     "bpf",
				OutputStem:    "generated",
				Tags:          []string{"linux"},
				Types:         []string{"event", "config", "stats"},
				CFlags:        nil,
				CC:            "",
			},
			checkArgs: func(t *testing.T, args []string) {
				// Count --type flags
				typeCount := 0
				for i, arg := range args {
					if arg == "--type" {
						typeCount++
						if i+1 >= len(args) {
							t.Errorf("--type flag missing value at index %d", i)
						}
					}
				}
				if typeCount != 3 {
					t.Errorf("expected 3 --type flags, got %d", typeCount)
				}
			},
		},
		{
			name: "cflags come after -- separator",
			src:  "./bpf/app.c",
			dest: "./out",
			opts: &Bpf2goOptions{
				Ident:         "app",
				Bpf2goVersion: "latest",
				GoPackage:     "out",
				OutputStem:    "gen",
				Tags:          []string{"linux"},
				Types:         nil,
				CFlags:        []string{"-O2", "-g", "-I./include"},
				CC:            "",
			},
			checkArgs: func(t *testing.T, args []string) {
				// Find "--" separator
				separatorIdx := -1
				for i, arg := range args {
					if arg == "--" {
						separatorIdx = i
						break
					}
				}
				if separatorIdx == -1 {
					t.Errorf("-- separator not found in args: %v", args)
					return
				}

				// Verify cflags come after separator
				cflags := args[separatorIdx+1:]
				expected := []string{"-O2", "-g", "-I./include"}
				if len(cflags) != len(expected) {
					t.Errorf("cflags length = %d, want %d", len(cflags), len(expected))
					return
				}
				for i, flag := range cflags {
					if flag != expected[i] {
						t.Errorf("cflags[%d] = %q, want %q", i, flag, expected[i])
					}
				}
			},
		},
		{
			name: "ident and src come after all flags",
			src:  "./bpf/myapp.c",
			dest: "./output",
			opts: &Bpf2goOptions{
				Ident:         "myapp",
				Bpf2goVersion: "latest",
				GoPackage:     "output",
				OutputStem:    "zz_generated",
				Tags:          []string{"linux"},
				Types:         nil,
				CFlags:        nil,
				CC:            "",
			},
			checkArgs: func(t *testing.T, args []string) {
				// Last two elements should be ident and src
				if len(args) < 2 {
					t.Errorf("expected at least 2 args, got %d", len(args))
					return
				}
				if args[len(args)-2] != "myapp" {
					t.Errorf("second to last arg = %q, want %q", args[len(args)-2], "myapp")
				}
				if args[len(args)-1] != "./bpf/myapp.c" {
					t.Errorf("last arg = %q, want %q", args[len(args)-1], "./bpf/myapp.c")
				}
			},
		},
		{
			name: "no cflags means no -- separator",
			src:  "./bpf/simple.c",
			dest: "./out",
			opts: &Bpf2goOptions{
				Ident:         "simple",
				Bpf2goVersion: "latest",
				GoPackage:     "out",
				OutputStem:    "gen",
				Tags:          []string{"linux"},
				Types:         nil,
				CFlags:        nil,
				CC:            "",
			},
			checkArgs: func(t *testing.T, args []string) {
				for _, arg := range args {
					if arg == "--" {
						t.Errorf("found -- separator when no cflags specified")
					}
				}
			},
		},
		{
			name: "empty tags array means no --tags flag",
			src:  "./bpf/notags.c",
			dest: "./out",
			opts: &Bpf2goOptions{
				Ident:         "notags",
				Bpf2goVersion: "latest",
				GoPackage:     "out",
				OutputStem:    "gen",
				Tags:          []string{},
				Types:         nil,
				CFlags:        nil,
				CC:            "",
			},
			checkArgs: func(t *testing.T, args []string) {
				for _, arg := range args {
					if arg == "--tags" {
						t.Errorf("found --tags flag when tags array is empty")
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildBpf2goArgs(tt.src, tt.dest, tt.opts)

			if tt.wantArgs != nil {
				if len(got) != len(tt.wantArgs) {
					t.Errorf("args length = %d, want %d\ngot: %v\nwant: %v", len(got), len(tt.wantArgs), got, tt.wantArgs)
					return
				}
				for i, want := range tt.wantArgs {
					if got[i] != want {
						t.Errorf("args[%d] = %q, want %q", i, got[i], want)
					}
				}
			}

			if tt.checkArgs != nil {
				tt.checkArgs(t, got)
			}
		})
	}
}

// TestBuildDependencies tests the buildDependencies function.
func TestBuildDependencies(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, dir string) (string, os.FileInfo) // returns srcPath and FileInfo
		wantCount int
		wantErr   bool
		checkDeps func(t *testing.T, deps []forge.ArtifactDependency, srcPath string)
	}{
		{
			name: "single source file dependency",
			setup: func(t *testing.T, dir string) (string, os.FileInfo) {
				srcPath := filepath.Join(dir, "prog.c")
				err := os.WriteFile(srcPath, []byte("// BPF program"), 0o644)
				if err != nil {
					t.Fatalf("failed to write prog.c: %v", err)
				}
				info, err := os.Stat(srcPath)
				if err != nil {
					t.Fatalf("failed to stat prog.c: %v", err)
				}
				return srcPath, info
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "dependency has Type DependencyTypeFile",
			setup: func(t *testing.T, dir string) (string, os.FileInfo) {
				srcPath := filepath.Join(dir, "app.c")
				err := os.WriteFile(srcPath, []byte("// BPF code"), 0o644)
				if err != nil {
					t.Fatalf("failed to write app.c: %v", err)
				}
				info, err := os.Stat(srcPath)
				if err != nil {
					t.Fatalf("failed to stat app.c: %v", err)
				}
				return srcPath, info
			},
			wantCount: 1,
			wantErr:   false,
			checkDeps: func(t *testing.T, deps []forge.ArtifactDependency, srcPath string) {
				if len(deps) != 1 {
					t.Errorf("expected 1 dependency, got %d", len(deps))
					return
				}
				if deps[0].Type != forge.DependencyTypeFile {
					t.Errorf("dependency.Type = %q, want %q", deps[0].Type, forge.DependencyTypeFile)
				}
			},
		},
		{
			name: "dependency has absolute path",
			setup: func(t *testing.T, dir string) (string, os.FileInfo) {
				subDir := filepath.Join(dir, "bpf")
				err := os.MkdirAll(subDir, 0o755)
				if err != nil {
					t.Fatalf("failed to create subdir: %v", err)
				}
				srcPath := filepath.Join(subDir, "nested.c")
				err = os.WriteFile(srcPath, []byte("// nested BPF"), 0o644)
				if err != nil {
					t.Fatalf("failed to write nested.c: %v", err)
				}
				info, err := os.Stat(srcPath)
				if err != nil {
					t.Fatalf("failed to stat nested.c: %v", err)
				}
				return srcPath, info
			},
			wantCount: 1,
			wantErr:   false,
			checkDeps: func(t *testing.T, deps []forge.ArtifactDependency, srcPath string) {
				if len(deps) != 1 {
					t.Errorf("expected 1 dependency, got %d", len(deps))
					return
				}
				if !filepath.IsAbs(deps[0].FilePath) {
					t.Errorf("dependency.FilePath = %q is not absolute", deps[0].FilePath)
				}
			},
		},
		{
			name: "dependency has valid RFC3339 timestamp",
			setup: func(t *testing.T, dir string) (string, os.FileInfo) {
				srcPath := filepath.Join(dir, "timestamped.c")
				err := os.WriteFile(srcPath, []byte("// BPF program"), 0o644)
				if err != nil {
					t.Fatalf("failed to write timestamped.c: %v", err)
				}
				info, err := os.Stat(srcPath)
				if err != nil {
					t.Fatalf("failed to stat timestamped.c: %v", err)
				}
				return srcPath, info
			},
			wantCount: 1,
			wantErr:   false,
			checkDeps: func(t *testing.T, deps []forge.ArtifactDependency, srcPath string) {
				if len(deps) != 1 {
					t.Errorf("expected 1 dependency, got %d", len(deps))
					return
				}
				if deps[0].Timestamp == "" {
					t.Errorf("dependency.Timestamp is empty")
					return
				}
				_, err := time.Parse(time.RFC3339, deps[0].Timestamp)
				if err != nil {
					t.Errorf("dependency.Timestamp = %q is not valid RFC3339: %v", deps[0].Timestamp, err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath, srcInfo := tt.setup(t, dir)

			got, err := buildDependencies(srcPath, srcInfo)

			if tt.wantErr {
				if err == nil {
					t.Errorf("buildDependencies() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("buildDependencies() unexpected error: %v", err)
				return
			}

			if len(got) != tt.wantCount {
				t.Errorf("buildDependencies() returned %d dependencies, want %d", len(got), tt.wantCount)
			}

			if tt.checkDeps != nil {
				tt.checkDeps(t, got, srcPath)
			}
		})
	}
}
