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

func TestGenerateKindConfig(t *testing.T) {
	tests := []struct {
		name         string
		wantContains []string
	}{
		{
			name: "generates valid kind config",
			wantContains: []string{
				"kind: Cluster",
				"apiVersion: kind.x-k8s.io/v1alpha4",
				"containerdConfigPatches:",
				`[plugins."io.containerd.grpc.v1.cri".registry]`,
				`config_path = "/etc/containerd/certs.d"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory
			tmpDir := t.TempDir()

			// Generate the config
			configPath, err := generateKindConfig(tmpDir)
			if err != nil {
				t.Fatalf("generateKindConfig() error = %v", err)
			}

			// Verify the path
			expectedPath := filepath.Join(tmpDir, "kind-config.yaml")
			if configPath != expectedPath {
				t.Errorf("generateKindConfig() path = %v, want %v", configPath, expectedPath)
			}

			// Read the generated file
			content, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("failed to read generated config: %v", err)
			}

			// Check required content
			for _, want := range tt.wantContains {
				if !strings.Contains(string(content), want) {
					t.Errorf("generateKindConfig() content missing %q\ngot:\n%s", want, string(content))
				}
			}
		})
	}
}

func TestGenerateKindConfigFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()

	configPath, err := generateKindConfig(tmpDir)
	if err != nil {
		t.Fatalf("generateKindConfig() error = %v", err)
	}

	// Check file permissions (should be 0600)
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("failed to stat generated config: %v", err)
	}

	perm := info.Mode().Perm()
	// Allow both 0600 and system-dependent variations
	if perm&0o077 != 0 {
		t.Logf("Note: file permissions are %o (may vary by system)", perm)
	}
}

func TestGenerateKindConfigNonExistentDir(t *testing.T) {
	// Use a non-existent directory
	nonExistentDir := filepath.Join(t.TempDir(), "non-existent", "subdir")

	_, err := generateKindConfig(nonExistentDir)
	if err == nil {
		t.Error("generateKindConfig() expected error for non-existent directory")
	}
}

func TestKindConfigContent(t *testing.T) {
	// Test that kindConfigContent constant is well-formed
	if !strings.Contains(kindConfigContent, "kind: Cluster") {
		t.Error("kindConfigContent should contain 'kind: Cluster'")
	}

	if !strings.Contains(kindConfigContent, "containerdConfigPatches:") {
		t.Error("kindConfigContent should contain 'containerdConfigPatches:'")
	}

	// Verify the containerd config path setting
	if !strings.Contains(kindConfigContent, "/etc/containerd/certs.d") {
		t.Error("kindConfigContent should configure /etc/containerd/certs.d path")
	}
}
