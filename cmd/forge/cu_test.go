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
	"testing"
)

func TestCUToolName(t *testing.T) {
	tests := []struct {
		subcmd  string
		want    string
		wantErr bool
	}{
		{subcmd: "status", want: "cu-status"},
		{subcmd: "commit", want: "cu-commit"},
		{subcmd: "checkout", want: "cu-checkout"},
		{subcmd: "list-branches", want: "cu-list-branches"},
		{subcmd: "go-get", want: "cu-go-get"},
		{subcmd: "unknown", wantErr: true},
		{subcmd: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.subcmd, func(t *testing.T) {
			got, err := cuToolName(tt.subcmd)
			if tt.wantErr {
				if err == nil {
					t.Errorf("cuToolName(%q) expected error, got nil", tt.subcmd)
				}
				return
			}
			if err != nil {
				t.Errorf("cuToolName(%q) unexpected error: %v", tt.subcmd, err)
				return
			}
			if got != tt.want {
				t.Errorf("cuToolName(%q) = %q, want %q", tt.subcmd, got, tt.want)
			}
		})
	}
}

func TestParseCUFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want map[string]any
	}{
		{
			name: "no flags",
			args: []string{},
			want: map[string]any{},
		},
		{
			name: "message flag",
			args: []string{"--message", "fix typo"},
			want: map[string]any{"message": "fix typo"},
		},
		{
			name: "branch flag",
			args: []string{"--branch", "feature/foo"},
			want: map[string]any{"branch": "feature/foo"},
		},
		{
			name: "package and version flags",
			args: []string{"--package", "github.com/foo/bar", "--version", "v1.2.3"},
			want: map[string]any{"package": "github.com/foo/bar", "version": "v1.2.3"},
		},
		{
			name: "cu-repo-path flag",
			args: []string{"--cu-repo-path", "/tmp/repo"},
			want: map[string]any{"cu-repo-path": "/tmp/repo"},
		},
		{
			name: "multiple flags",
			args: []string{"--message", "hello", "--cu-repo-path", "/tmp"},
			want: map[string]any{"message": "hello", "cu-repo-path": "/tmp"},
		},
		{
			name: "trailing flag without value is ignored",
			args: []string{"--message"},
			want: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCUFlags(tt.args)
			if len(got) != len(tt.want) {
				t.Errorf("parseCUFlags(%v) returned %d params, want %d", tt.args, len(got), len(tt.want))
				return
			}
			for k, v := range tt.want {
				gv, ok := got[k]
				if !ok {
					t.Errorf("parseCUFlags(%v) missing key %q", tt.args, k)
					continue
				}
				if gv != v {
					t.Errorf("parseCUFlags(%v)[%q] = %v, want %v", tt.args, k, gv, v)
				}
			}
		})
	}
}

func TestRunCU_NoSubcommand(t *testing.T) {
	err := runCU(nil)
	if err == nil {
		t.Fatal("runCU(nil) expected error, got nil")
	}

	expected := "cu subcommand required (status, commit, checkout, list-branches, go-get)"
	if err.Error() != expected {
		t.Errorf("runCU(nil) error = %q, want %q", err.Error(), expected)
	}

	err = runCU([]string{})
	if err == nil {
		t.Fatal("runCU([]) expected error, got nil")
	}
	if err.Error() != expected {
		t.Errorf("runCU([]) error = %q, want %q", err.Error(), expected)
	}
}
