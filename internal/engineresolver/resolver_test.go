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

package engineresolver

import (
	"strings"
	"testing"
)

func TestParseEngineURI_GoProtocol(t *testing.T) {
	tests := []struct {
		name             string
		engineURI        string
		forgeVersion     string
		wantEngineType   string
		wantCommand      string
		wantArgsContains string
		wantErr          bool
		wantErrContains  string
	}{
		{
			name:             "short name go-build",
			engineURI:        "go://go-build",
			forgeVersion:     "v1.0.0",
			wantEngineType:   EngineTypeMCP,
			wantCommand:      "go",
			wantArgsContains: "go-build",
			wantErr:          false,
		},
		{
			name:             "short name testenv-kind",
			engineURI:        "go://testenv-kind",
			forgeVersion:     "v0.9.0",
			wantEngineType:   EngineTypeMCP,
			wantCommand:      "go",
			wantArgsContains: "testenv-kind",
			wantErr:          false,
		},
		{
			name:             "full path extracts last component",
			engineURI:        "go://github.com/user/repo/cmd/mytool",
			forgeVersion:     "v1.2.3",
			wantEngineType:   EngineTypeMCP,
			wantCommand:      "go",
			wantArgsContains: "mytool",
			wantErr:          false,
		},
		{
			name:             "with version suffix is ignored",
			engineURI:        "go://go-test@v1.0.0",
			forgeVersion:     "v2.0.0",
			wantEngineType:   EngineTypeMCP,
			wantCommand:      "go",
			wantArgsContains: "go-test",
			wantErr:          false,
		},
		{
			name:            "empty path after go://",
			engineURI:       "go://",
			forgeVersion:    "v1.0.0",
			wantErr:         true,
			wantErrContains: "empty engine path",
		},
		{
			name:            "empty forge version",
			engineURI:       "go://go-build",
			forgeVersion:    "",
			wantErr:         true,
			wantErrContains: "forge version cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engineType, command, args, err := ParseEngineURI(tt.engineURI, tt.forgeVersion)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseEngineURI() expected error, got nil")
					return
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("ParseEngineURI() error = %v, want error containing %q", err, tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseEngineURI() unexpected error = %v", err)
				return
			}

			if engineType != tt.wantEngineType {
				t.Errorf("ParseEngineURI() engineType = %v, want %v", engineType, tt.wantEngineType)
			}

			if command != tt.wantCommand {
				t.Errorf("ParseEngineURI() command = %v, want %v", command, tt.wantCommand)
			}

			if len(args) == 0 {
				t.Errorf("ParseEngineURI() args is empty, expected non-empty")
				return
			}

			// Check that args contain the expected package name
			argsJoined := strings.Join(args, " ")
			if !strings.Contains(argsJoined, tt.wantArgsContains) {
				t.Errorf("ParseEngineURI() args = %v, want args containing %q", args, tt.wantArgsContains)
			}
		})
	}
}

func TestParseEngineURI_AliasProtocol(t *testing.T) {
	tests := []struct {
		name            string
		engineURI       string
		forgeVersion    string
		wantEngineType  string
		wantCommand     string
		wantArgs        []string
		wantErr         bool
		wantErrContains string
	}{
		{
			name:           "valid alias",
			engineURI:      "alias://my-engine",
			forgeVersion:   "v1.0.0",
			wantEngineType: EngineTypeAlias,
			wantCommand:    "my-engine",
			wantArgs:       nil,
			wantErr:        false,
		},
		{
			name:           "alias with dashes",
			engineURI:      "alias://my-custom-build-engine",
			forgeVersion:   "v1.0.0",
			wantEngineType: EngineTypeAlias,
			wantCommand:    "my-custom-build-engine",
			wantArgs:       nil,
			wantErr:        false,
		},
		{
			name:            "empty alias name",
			engineURI:       "alias://",
			forgeVersion:    "v1.0.0",
			wantErr:         true,
			wantErrContains: "empty alias name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engineType, command, args, err := ParseEngineURI(tt.engineURI, tt.forgeVersion)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseEngineURI() expected error, got nil")
					return
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("ParseEngineURI() error = %v, want error containing %q", err, tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseEngineURI() unexpected error = %v", err)
				return
			}

			if engineType != tt.wantEngineType {
				t.Errorf("ParseEngineURI() engineType = %v, want %v", engineType, tt.wantEngineType)
			}

			if command != tt.wantCommand {
				t.Errorf("ParseEngineURI() command = %v, want %v", command, tt.wantCommand)
			}

			if len(args) != len(tt.wantArgs) {
				t.Errorf("ParseEngineURI() args = %v, want %v", args, tt.wantArgs)
			}
		})
	}
}

func TestParseEngineURI_UnsupportedProtocol(t *testing.T) {
	tests := []struct {
		name            string
		engineURI       string
		forgeVersion    string
		wantErrContains string
	}{
		{
			name:            "http protocol",
			engineURI:       "http://example.com/engine",
			forgeVersion:    "v1.0.0",
			wantErrContains: "unsupported engine protocol",
		},
		{
			name:            "https protocol",
			engineURI:       "https://example.com/engine",
			forgeVersion:    "v1.0.0",
			wantErrContains: "unsupported engine protocol",
		},
		{
			name:            "no protocol",
			engineURI:       "my-engine",
			forgeVersion:    "v1.0.0",
			wantErrContains: "unsupported engine protocol",
		},
		{
			name:            "docker protocol",
			engineURI:       "docker://myimage:latest",
			forgeVersion:    "v1.0.0",
			wantErrContains: "unsupported engine protocol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := ParseEngineURI(tt.engineURI, tt.forgeVersion)

			if err == nil {
				t.Errorf("ParseEngineURI() expected error for unsupported protocol, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Errorf("ParseEngineURI() error = %v, want error containing %q", err, tt.wantErrContains)
			}
		})
	}
}

func TestParseEngineURI_Constants(t *testing.T) {
	// Verify constants have expected values
	if EngineTypeMCP != "mcp" {
		t.Errorf("EngineTypeMCP = %q, want %q", EngineTypeMCP, "mcp")
	}
	if EngineTypeAlias != "alias" {
		t.Errorf("EngineTypeAlias = %q, want %q", EngineTypeAlias, "alias")
	}
}
