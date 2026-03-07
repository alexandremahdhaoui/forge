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
	"fmt"
	"os"
	"syscall"

	"github.com/alexandremahdhaoui/forge/internal/cmdutil"
)

// resolveUIBinary resolves the forge-ui-tui binary path from user config or PATH.
// Extracted for testability since syscall.Exec cannot be tested in unit tests.
func resolveUIBinary(userConfig cmdutil.UserConfig) (string, error) {
	binary, _, err := cmdutil.ResolveToolBinary(userConfig.Tools.UI, "forge-ui-tui", "", "")
	if err != nil {
		return "", fmt.Errorf("forge-ui-tui binary not found: install it or set tools.ui in ~/.config/forge/config.yaml")
	}
	return binary, nil
}

// runUI launches the forge-ui-tui binary, replacing the current process.
func runUI(args []string) error {
	userConfig, err := cmdutil.LoadUserConfig()
	if err != nil {
		return fmt.Errorf("failed to load user config: %w", err)
	}

	binary, err := resolveUIBinary(userConfig)
	if err != nil {
		return err
	}

	err = syscall.Exec(binary, append([]string{binary}, args...), os.Environ())
	// syscall.Exec only returns on error
	return err
}
