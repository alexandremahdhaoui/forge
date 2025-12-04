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

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

func TestProcessTemplatedArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		input    mcptypes.BuildInput
		expected []string
		wantErr  bool
	}{
		{
			name: "simple template substitution",
			args: []string{"build", "-o", "{{ .Dest }}/{{ .Name }}", "{{ .Src }}"},
			input: mcptypes.BuildInput{
				Name: "myapp",
				Src:  "./cmd/myapp",
				Dest: "./build/bin",
			},
			expected: []string{"build", "-o", "./build/bin/myapp", "./cmd/myapp"},
			wantErr:  false,
		},
		{
			name: "no templates",
			args: []string{"echo", "hello", "world"},
			input: mcptypes.BuildInput{
				Name: "test",
			},
			expected: []string{"echo", "hello", "world"},
			wantErr:  false,
		},
		{
			name: "empty args",
			args: []string{},
			input: mcptypes.BuildInput{
				Name: "test",
			},
			expected: []string{},
			wantErr:  false,
		},
		{
			name: "all template variables",
			args: []string{"Name={{ .Name }}", "Src={{ .Src }}", "Dest={{ .Dest }}", "Engine={{ .Engine }}"},
			input: mcptypes.BuildInput{
				Name:   "myapp",
				Src:    "./src",
				Dest:   "./dest",
				Engine: "go://go-build",
			},
			expected: []string{"Name=myapp", "Src=./src", "Dest=./dest", "Engine=go://go-build"},
			wantErr:  false,
		},
		{
			name: "invalid template",
			args: []string{"{{ .Invalid }"},
			input: mcptypes.BuildInput{
				Name: "test",
			},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processTemplatedArgs(tt.args, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("processTemplatedArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.expected) {
				t.Errorf("processTemplatedArgs() length = %v, want %v", len(got), len(tt.expected))
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("processTemplatedArgs()[%d] = %v, want %v", i, got[i], tt.expected[i])
				}
			}
		})
	}
}
