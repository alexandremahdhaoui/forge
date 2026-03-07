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
	"strings"
	"testing"
)

func TestWsToolName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		subcmd  string
		want    string
		wantErr bool
	}{
		{name: "list", subcmd: "list", want: "list-workspaces"},
		{name: "create", subcmd: "create", want: "create-workspace"},
		{name: "get", subcmd: "get", want: "get-workspace"},
		{name: "delete", subcmd: "delete", want: "delete-workspace"},
		{name: "suspend", subcmd: "suspend", want: "suspend-workspace"},
		{name: "resume", subcmd: "resume", want: "resume-workspace"},
		{name: "unknown", subcmd: "unknown", wantErr: true},
		{name: "empty", subcmd: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := wsToolName(tt.subcmd)
			if tt.wantErr {
				if err == nil {
					t.Errorf("wsToolName(%q) expected error, got nil", tt.subcmd)
				}
				return
			}
			if err != nil {
				t.Errorf("wsToolName(%q) unexpected error: %v", tt.subcmd, err)
				return
			}
			if got != tt.want {
				t.Errorf("wsToolName(%q) = %q, want %q", tt.subcmd, got, tt.want)
			}
		})
	}
}

func TestParseWSFlags(t *testing.T) {
	t.Parallel()

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
			name: "name flag",
			args: []string{"--name", "my-workspace"},
			want: map[string]any{"name": "my-workspace"},
		},
		{
			name: "image and namespace flags",
			args: []string{"--image", "ubuntu:22.04", "--namespace", "dev"},
			want: map[string]any{"image": "ubuntu:22.04", "namespace": "dev"},
		},
		{
			name: "flag without value is ignored",
			args: []string{"--orphan"},
			want: map[string]any{},
		},
		{
			name: "non-flag args are ignored",
			args: []string{"positional", "--name", "ws1"},
			want: map[string]any{"name": "ws1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseWSFlags(tt.args)
			if len(got) != len(tt.want) {
				t.Errorf("parseWSFlags(%v) returned %d params, want %d", tt.args, len(got), len(tt.want))
				return
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("parseWSFlags(%v)[%q] = %v, want %v", tt.args, k, got[k], v)
				}
			}
		})
	}
}

func TestRunWS_NoSubcommand(t *testing.T) {
	t.Parallel()

	err := runWS([]string{})
	if err == nil {
		t.Fatal("runWS with no args should return error")
	}
	if !strings.Contains(err.Error(), "ws subcommand required") {
		t.Errorf("expected 'ws subcommand required' error, got: %v", err)
	}
}

func TestRunWS_UnknownSubcommand(t *testing.T) {
	t.Parallel()

	err := runWS([]string{"invalid"})
	if err == nil {
		t.Fatal("runWS with unknown subcommand should return error")
	}
	if !strings.Contains(err.Error(), "unknown ws subcommand") {
		t.Errorf("expected 'unknown ws subcommand' error, got: %v", err)
	}
}
