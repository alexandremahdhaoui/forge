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
	"strings"

	"github.com/alexandremahdhaoui/forge/internal/cmdutil"
)

// cuToolName maps a CLI subcommand to the corresponding MCP tool name.
func cuToolName(subcmd string) (string, error) {
	switch subcmd {
	case "status", "commit", "checkout", "list-branches", "go-get":
		return "cu-" + subcmd, nil
	default:
		return "", fmt.Errorf("unknown cu subcommand: %s (available: status, commit, checkout, list-branches, go-get)", subcmd)
	}
}

// parseCUFlags parses CLI flags from args into a map of MCP tool parameters.
func parseCUFlags(args []string) map[string]any {
	params := make(map[string]any)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		key := strings.TrimPrefix(arg, "--")
		if i+1 >= len(args) {
			break
		}
		i++
		params[key] = args[i]
	}
	return params
}

// runCU handles the cu command.
func runCU(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("cu subcommand required (status, commit, checkout, list-branches, go-get)")
	}

	subcmd := args[0]

	toolName, err := cuToolName(subcmd)
	if err != nil {
		return err
	}

	// Load user config for binary resolution.
	userConfig, err := cmdutil.LoadUserConfig()
	if err != nil {
		return fmt.Errorf("failed to load user config: %w", err)
	}

	// Load forge.yaml for cu configuration.
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load forge config: %w", err)
	}

	// Resolve forge-cu binary.
	binary, binaryArgs, err := cmdutil.ResolveToolBinary(
		userConfig.Tools.CU,
		"forge-cu",
		"github.com/alexandremahdhaoui/forge-cu",
		getVersion(),
	)
	if err != nil {
		return fmt.Errorf("failed to resolve forge-cu binary: %w", err)
	}

	// Build MCP tool params from CLI flags.
	params := parseCUFlags(args[1:])

	// Inject cu-repo-path from forge.yaml if not provided by CLI flags.
	if _, ok := params["cu-repo-path"]; !ok {
		if config.CU != nil && config.CU.CompoURL != "" {
			params["cu-repo-path"] = config.CU.CompoURL
		}
	}

	// Inject managedFiles from forge.yaml if available.
	if config.CU != nil && len(config.CU.ManagedFiles) > 0 {
		if _, ok := params["managedFiles"]; !ok {
			params["managedFiles"] = config.CU.ManagedFiles
		}
	}

	// Call MCP engine.
	result, err := callMCPEngine(binary, binaryArgs, toolName, params)
	if err != nil {
		return err
	}

	if result != nil {
		fmt.Println(result)
	}

	return nil
}
