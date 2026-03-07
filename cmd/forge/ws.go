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

// wsToolName maps a CLI subcommand to the corresponding MCP tool name.
func wsToolName(subcmd string) (string, error) {
	switch subcmd {
	case "list":
		return "list-workspaces", nil
	case "create":
		return "create-workspace", nil
	case "get":
		return "get-workspace", nil
	case "delete":
		return "delete-workspace", nil
	case "suspend":
		return "suspend-workspace", nil
	case "resume":
		return "resume-workspace", nil
	default:
		return "", fmt.Errorf("unknown ws subcommand: %s (available: list, create, get, delete, suspend, resume)", subcmd)
	}
}

// parseWSFlags parses CLI flags from args into a map of MCP tool parameters.
func parseWSFlags(args []string) map[string]any {
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

// runWS handles the ws command.
func runWS(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("ws subcommand required (list, create, get, delete, suspend, resume)")
	}

	subcmd := args[0]

	toolName, err := wsToolName(subcmd)
	if err != nil {
		return err
	}

	// Load user config for binary resolution.
	userConfig, err := cmdutil.LoadUserConfig()
	if err != nil {
		return fmt.Errorf("failed to load user config: %w", err)
	}

	// Resolve forge-ws binary.
	binary, binaryArgs, err := cmdutil.ResolveToolBinary(
		userConfig.Tools.WS,
		"forge-ws",
		"github.com/alexandremahdhaoui/forge-workspace",
		getVersion(),
	)
	if err != nil {
		return fmt.Errorf("failed to resolve forge-ws binary: %w", err)
	}

	// Build MCP tool params from CLI flags.
	params := parseWSFlags(args[1:])

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
