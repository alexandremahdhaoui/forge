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

package cmdutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/alexandremahdhaoui/forge/internal/forgepath"

	"sigs.k8s.io/yaml"
)

// UserConfig represents the user-level forge configuration.
type UserConfig struct {
	Tools ToolsConfig `json:"tools"`
}

// ToolsConfig holds user-specified tool binary overrides.
type ToolsConfig struct {
	CU string `json:"cu,omitempty"`
	WS string `json:"ws,omitempty"`
	UI string `json:"ui,omitempty"`
}

// LoadUserConfig reads the user configuration from ~/.config/forge/config.yaml.
// Returns a zero-value UserConfig if the file does not exist (not an error).
// Returns an error only for invalid YAML.
func LoadUserConfig() (UserConfig, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return UserConfig{}, fmt.Errorf("failed to get user config dir: %w", err)
	}

	path := filepath.Join(configDir, "forge", "config.yaml")
	return loadUserConfigFromPath(path)
}

// loadUserConfigFromPath reads the user configuration from the given file path.
// Returns a zero-value UserConfig if the file does not exist (not an error).
// Returns an error only for invalid YAML.
func loadUserConfigFromPath(path string) (UserConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return UserConfig{}, nil
		}
		return UserConfig{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg UserConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return UserConfig{}, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// ResolveToolBinary resolves a tool binary using the following priority:
//  1. userOverride if non-empty
//  2. exec.LookPath(binaryName)
//  3. go run via forgepath.BuildExternalGoRunCommand if goModule is non-empty
//  4. error if nothing found
//
// Returns (binary, args, error) where binary is the executable path and args
// are additional arguments (non-nil only for the go run fallback).
func ResolveToolBinary(userOverride, binaryName, goModule, forgeVersion string) (string, []string, error) {
	// Priority 1: User override
	if userOverride != "" {
		return userOverride, nil, nil
	}

	// Priority 2: Binary on PATH
	if path, err := exec.LookPath(binaryName); err == nil {
		return path, nil, nil
	}

	// Priority 3: go run via external module
	if goModule != "" {
		args, err := forgepath.BuildExternalGoRunCommand(goModule+"/cmd/"+binaryName, forgeVersion)
		if err != nil {
			return "", nil, fmt.Errorf("failed to build go run command for %s: %w", binaryName, err)
		}
		return "go", args, nil
	}

	return "", nil, fmt.Errorf("tool binary %q not found: not in user config, not on PATH, and no go module specified", binaryName)
}
