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

	"github.com/alexandremahdhaoui/forge/internal/forgepath"
)

// parseEngine parses an engine URI and returns the engine type, command, and args for execution.
// Supports go:// and alias:// protocols:
//   - go://go-build -> executes via `go run github.com/alexandremahdhaoui/forge/cmd/go-build@{forgeVersion}`
//   - go://testenv-kind -> executes via `go run github.com/alexandremahdhaoui/forge/cmd/testenv-kind@{forgeVersion}`
//   - alias://my-engine -> resolves alias from forge.yaml engines section
//
// Returns:
//   - engineType: "mcp" for go:// URIs, "alias" for alias:// URIs
//   - command: "go" for go:// URIs, aliasName for alias:// URIs
//   - args: ["run", "package/path@version"] for go:// URIs, nil for alias:// URIs
//   - err: error if parsing fails
func parseEngine(engineURI, forgeVersion string) (engineType string, command string, args []string, err error) {
	// Check for alias:// protocol - return special marker
	if strings.HasPrefix(engineURI, "alias://") {
		aliasName := strings.TrimPrefix(engineURI, "alias://")
		if aliasName == "" {
			return "", "", nil, fmt.Errorf("empty alias name after alias://")
		}
		// Return special marker - caller will handle resolution
		return "alias", aliasName, nil, nil
	}

	if !strings.HasPrefix(engineURI, "go://") {
		return "", "", nil, fmt.Errorf("unsupported engine protocol: %s (must start with go:// or alias://)", engineURI)
	}

	// Remove go:// prefix
	path := strings.TrimPrefix(engineURI, "go://")
	if path == "" {
		return "", "", nil, fmt.Errorf("empty engine path after go://")
	}

	// Extract package name (ignore version specifiers for go run)
	packageName := path
	if idx := strings.Index(path, "@"); idx != -1 {
		packageName = path[:idx]
	}

	// Expand short names to just the binary name
	// If path doesn't contain slashes, it's a short name like "testenv-kind"
	if !strings.Contains(packageName, "/") {
		// Just the package name, will be expanded by BuildGoRunCommand
	} else {
		// Full path like "github.com/user/repo/cmd/tool" - extract last component
		parts := strings.Split(packageName, "/")
		packageName = parts[len(parts)-1]
	}

	if packageName == "" {
		return "", "", nil, fmt.Errorf("could not extract package name from engine URI: %s", engineURI)
	}

	// Use forgepath to build the go run command
	runArgs, err := forgepath.BuildGoRunCommand(packageName, forgeVersion)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to build go run command for %s: %w", packageName, err)
	}

	// Return command and args for go run
	return "mcp", "go", runArgs, nil
}
