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

// Package engineresolver provides shared engine URI parsing functionality.
// This package handles the parsing of engine URIs (forge://, alias://) and
// returns the engine type, command, and args for execution.
//
// IMPORTANT: This package ONLY parses URIs. For alias:// URIs, it returns
// EngineTypeAlias - the caller must handle alias resolution separately.
package engineresolver

import (
	"fmt"
	"strings"

	"github.com/alexandremahdhaoui/forge/internal/forgepath"
)

const (
	// EngineTypeMCP indicates a forge:// URI that should be executed as an MCP server.
	EngineTypeMCP = "mcp"
	// EngineTypeAlias indicates an alias:// URI that requires resolution by the caller.
	EngineTypeAlias = "alias"
)

// GoSchemeError is the message a go:// URI fails with. The scheme is removed,
// not aliased, so every stale file fails loudly and identically.
const GoSchemeError = "the go:// scheme is removed; use forge:// " +
	"(forge://<name> for forge's own engines, " +
	"forge://<module-path>[@rev] resolves a factory member through its register)"

// ParseEngineURI parses an engine URI and returns the engine type, command, and args.
// Supports forge:// and alias:// protocols:
//   - forge://go-build -> forge's own engine at the running forge version,
//     executed via `go run github.com/alexandremahdhaoui/forge/cmd/go-build@{forgeVersion}`
//   - forge://github.com/x/repo/cmd/tool -> a factory member, materialized and
//     executed via `forge-factory run github.com/x/repo/cmd/tool -- --mcp`;
//     the version comes from the member's register unless an @rev is given
//   - alias://my-engine -> returns EngineTypeAlias with aliasName - caller must resolve
//
// Returns:
//   - engineType: EngineTypeMCP for forge:// URIs, EngineTypeAlias for alias:// URIs
//   - command: "go" or "forge-factory" for forge:// URIs, aliasName for alias:// URIs
//   - args: the command's arguments, nil for alias:// URIs
//   - err: error if parsing fails
func ParseEngineURI(engineURI, forgeVersion string) (engineType string, command string, args []string, err error) {
	// Check for alias:// protocol - return marker for caller to handle
	if strings.HasPrefix(engineURI, "alias://") {
		aliasName := strings.TrimPrefix(engineURI, "alias://")
		if aliasName == "" {
			return "", "", nil, fmt.Errorf("empty alias name after alias://")
		}
		// Return special marker - caller will handle resolution
		return EngineTypeAlias, aliasName, nil, nil
	}

	if strings.HasPrefix(engineURI, "go://") {
		return "", "", nil, fmt.Errorf("%s: %s", engineURI, GoSchemeError)
	}

	if !strings.HasPrefix(engineURI, "forge://") {
		return "", "", nil, fmt.Errorf("unsupported engine protocol: %s (must start with forge:// or alias://)", engineURI)
	}

	path := strings.TrimPrefix(engineURI, "forge://")
	if path == "" {
		return "", "", nil, fmt.Errorf("empty engine path after forge://")
	}

	// A member form names a repo by URL-ish module path. The enclosing Go
	// workspace wins when it carries the module - the engine twin of run's
	// rule 2 - and go run uses the local copy. Otherwise forge-factory owns
	// materializing it: clone, resolve its factory and register, sync,
	// build; the run cache makes warm starts a plain exec. Forge's own
	// module path stays a built-in: forge resolves itself with no factory.
	bare := strings.SplitN(path, "@", 2)[0]
	if forgepath.IsExternalModule(bare) && !forgepath.IsForgeModulePath(bare) {
		if forgepath.IsWorkspaceModule(bare) {
			return EngineTypeMCP, "go", []string{"run", bare}, nil
		}

		return EngineTypeMCP, "forge-factory", []string{"run", "--quiet", path, "--", "--mcp"}, nil
	}

	// A short form is one of forge's own engines at the running forge version.
	// Embedded versions are IGNORED for internal engines to ensure consistency.
	packageName := path
	if idx := strings.Index(path, "@"); idx != -1 {
		packageName = path[:idx]
	}

	if strings.Contains(packageName, "/") {
		// Sub-path like "cmd/tool" - extract last component
		parts := strings.Split(packageName, "/")
		packageName = parts[len(parts)-1]
	}

	if packageName == "" {
		return "", "", nil, fmt.Errorf("could not extract package name from engine URI: %s", engineURI)
	}

	runArgs, err := forgepath.BuildGoRunCommand(packageName, forgeVersion)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to build go run command for %s: %w", packageName, err)
	}

	return EngineTypeMCP, "go", runArgs, nil
}
