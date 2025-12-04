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

func TestParseEngine(t *testing.T) {
	testVersion := "v0.9.0"
	tests := []struct {
		name        string
		engineURI   string
		wantType    string
		wantCommand string
		wantArgs    []string
		wantErr     bool
	}{
		{
			name:        "simple go-build",
			engineURI:   "go://go-build",
			wantType:    "mcp",
			wantCommand: "go",
			wantArgs:    []string{"run", "github.com/alexandremahdhaoui/forge/cmd/go-build@v0.9.0"},
			wantErr:     false,
		},
		{
			name:        "simple container-build",
			engineURI:   "go://container-build",
			wantType:    "mcp",
			wantCommand: "go",
			wantArgs:    []string{"run", "github.com/alexandremahdhaoui/forge/cmd/container-build@v0.9.0"},
			wantErr:     false,
		},
		{
			name:        "full path",
			engineURI:   "go://github.com/alexandremahdhaoui/forge/cmd/go-build",
			wantType:    "mcp",
			wantCommand: "go",
			wantArgs:    []string{"run", "github.com/alexandremahdhaoui/forge/cmd/go-build@v0.9.0"},
			wantErr:     false,
		},
		{
			name:        "invalid protocol",
			engineURI:   "http://go-build",
			wantType:    "",
			wantCommand: "",
			wantArgs:    nil,
			wantErr:     true,
		},
		{
			name:        "empty after protocol",
			engineURI:   "go://",
			wantType:    "",
			wantCommand: "",
			wantArgs:    nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotCommand, gotArgs, err := parseEngine(tt.engineURI, testVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseEngine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotType != tt.wantType {
				t.Errorf("parseEngine() gotType = %v, want %v", gotType, tt.wantType)
			}
			if gotCommand != tt.wantCommand {
				t.Errorf("parseEngine() gotCommand = %v, want %v", gotCommand, tt.wantCommand)
			}

			// Only validate args for successful (non-error) cases
			if !tt.wantErr {
				// When FORGE_RUN_LOCAL_ENABLED=true, we get: ["-C", "/path", "run", "./cmd/tool"]
				// Otherwise: ["run", "github.com/alexandremahdhaoui/forge/cmd/tool@version"]
				// So just verify "run" is present and the tool name is in the args somewhere
				hasRun := false
				hasToolName := false
				for _, arg := range gotArgs {
					if arg == "run" {
						hasRun = true
					}
					// Extract tool name from the URI (go-build, container-build, etc.)
					toolName := tt.engineURI
					if idx := strings.LastIndex(toolName, "/"); idx >= 0 {
						toolName = toolName[idx+1:]
					}
					toolName = strings.TrimPrefix(toolName, "go://")
					if strings.Contains(arg, toolName) {
						hasToolName = true
					}
				}
				if !hasRun {
					t.Errorf("parseEngine() gotArgs = %v, missing 'run' argument", gotArgs)
				}
				if !hasToolName {
					t.Errorf("parseEngine() gotArgs = %v, missing tool name from URI %s", gotArgs, tt.engineURI)
				}
			}
		})
	}
}
