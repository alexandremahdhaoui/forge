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
	"errors"
	"strings"
	"testing"
)

func TestValidateTools(t *testing.T) {
	tests := []struct {
		name                 string
		mockCheckTool        func(name string, args []string) error
		mockCheckHelmVersion func(output string) error
		mockGetHelmVersion   func() (string, error)
		wantErr              bool
		wantErrContains      string
	}{
		{
			name: "all tools available",
			mockCheckTool: func(name string, args []string) error {
				return nil
			},
			mockCheckHelmVersion: func(output string) error {
				return nil
			},
			mockGetHelmVersion: func() (string, error) {
				return "version.BuildInfo{Version:\"v3.10.1\"}", nil
			},
			wantErr: false,
		},
		{
			name: "helm missing",
			mockCheckTool: func(name string, args []string) error {
				if name == "helm" {
					return errors.New("helm not found")
				}
				return nil
			},
			mockCheckHelmVersion: func(output string) error {
				return nil
			},
			mockGetHelmVersion: func() (string, error) {
				return "", errors.New("not found")
			},
			wantErr:         true,
			wantErrContains: "helm (>=3.8.0)",
		},
		{
			name: "git missing",
			mockCheckTool: func(name string, args []string) error {
				if name == "git" {
					return errors.New("git not found")
				}
				return nil
			},
			mockCheckHelmVersion: func(output string) error {
				return nil
			},
			mockGetHelmVersion: func() (string, error) {
				return "version.BuildInfo{Version:\"v3.10.1\"}", nil
			},
			wantErr:         true,
			wantErrContains: "git",
		},
		{
			name: "kubectl missing",
			mockCheckTool: func(name string, args []string) error {
				if name == "kubectl" {
					return errors.New("kubectl not found")
				}
				return nil
			},
			mockCheckHelmVersion: func(output string) error {
				return nil
			},
			mockGetHelmVersion: func() (string, error) {
				return "version.BuildInfo{Version:\"v3.10.1\"}", nil
			},
			wantErr:         true,
			wantErrContains: "kubectl",
		},
		{
			name: "multiple tools missing",
			mockCheckTool: func(name string, args []string) error {
				if name == "git" || name == "kubectl" {
					return errors.New("not found")
				}
				return nil
			},
			mockCheckHelmVersion: func(output string) error {
				return nil
			},
			mockGetHelmVersion: func() (string, error) {
				return "version.BuildInfo{Version:\"v3.10.1\"}", nil
			},
			wantErr:         true,
			wantErrContains: "git",
		},
		{
			name: "helm version too old",
			mockCheckTool: func(name string, args []string) error {
				return nil
			},
			mockCheckHelmVersion: func(output string) error {
				return errors.New("helm version 3.7.x is too old, requires >= 3.8.0 for OCI support")
			},
			mockGetHelmVersion: func() (string, error) {
				return "version.BuildInfo{Version:\"v3.7.0\"}", nil
			},
			wantErr:         true,
			wantErrContains: "3.8.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &ToolValidator{
				checkToolFn:        tt.mockCheckTool,
				checkHelmVersionFn: tt.mockCheckHelmVersion,
				getHelmVersionFn:   tt.mockGetHelmVersion,
			}

			err := v.ValidateTools()

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateTools() expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("ValidateTools() error = %v, want error containing %q", err, tt.wantErrContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateTools() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestCheckHelmVersion(t *testing.T) {
	tests := []struct {
		name        string
		versionStr  string
		wantErr     bool
		errContains string
	}{
		{
			name:       "valid version 3.8.0",
			versionStr: "version.BuildInfo{Version:\"v3.8.0\", GitCommit:\"...\", GitTreeState:\"clean\", GoVersion:\"go1.17.5\"}",
			wantErr:    false,
		},
		{
			name:       "valid version 3.10.1",
			versionStr: "version.BuildInfo{Version:\"v3.10.1\", GitCommit:\"...\", GitTreeState:\"clean\", GoVersion:\"go1.18\"}",
			wantErr:    false,
		},
		{
			name:        "invalid version 3.7.0",
			versionStr:  "version.BuildInfo{Version:\"v3.7.0\", GitCommit:\"...\", GitTreeState:\"clean\", GoVersion:\"go1.17\"}",
			wantErr:     true,
			errContains: "3.8.0",
		},
		{
			name:        "invalid version 2.x",
			versionStr:  "version.BuildInfo{Version:\"v2.17.0\", GitCommit:\"...\", GitTreeState:\"clean\", GoVersion:\"go1.15\"}",
			wantErr:     true,
			errContains: "3.8.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkHelmVersion(tt.versionStr)

			if tt.wantErr {
				if err == nil {
					t.Errorf("checkHelmVersion() expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("checkHelmVersion() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("checkHelmVersion() unexpected error = %v", err)
				}
			}
		})
	}
}
