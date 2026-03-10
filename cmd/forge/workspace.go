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
	"path/filepath"
	"strings"

	"github.com/alexandremahdhaoui/forge/internal/forgepath"
)

// resolveWorkspace auto-detects Go workspaces and sets environment variables
// for local development mode. If the CWD is the workspace root (not inside
// a member module), it changes CWD to the forge repo member.
//
// Environment variables set when go.work is found:
//   - FORGE_RUN_LOCAL_ENABLED=true
//   - FORGE_RUN_LOCAL_BASEDIR=<forge repo directory>
func resolveWorkspace() error {
	if skipWorkspaceResolution {
		return nil
	}

	// Walk up from CWD looking for go.work
	wsRoot := forgepath.FindGoWork()
	if wsRoot == "" {
		return nil
	}

	goWorkPath := filepath.Join(wsRoot, "go.work")
	content, err := os.ReadFile(goWorkPath)
	if err != nil {
		return fmt.Errorf("cannot read go.work at %q: %w", goWorkPath, err)
	}

	useDirs := forgepath.ParseGoWorkUseDirs(string(content))
	if len(useDirs) == 0 {
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot get working directory: %w", err)
	}

	// Resolve all use directories to absolute paths
	absUseDirs := make([]string, 0, len(useDirs))
	for _, useDir := range useDirs {
		var absUseDir string
		if filepath.IsAbs(useDir) {
			absUseDir = useDir
		} else {
			absUseDir = filepath.Join(wsRoot, useDir)
		}
		absUseDir, err = filepath.EvalSymlinks(absUseDir)
		if err != nil {
			continue
		}
		absUseDirs = append(absUseDirs, absUseDir)
	}

	// Find the forge repo member (needed for BASEDIR in all cases)
	var forgeRepoDir string
	for _, absUseDir := range absUseDirs {
		if forgepath.IsForgeRepo(absUseDir) {
			forgeRepoDir = absUseDir
			break
		}
	}

	// Check if CWD is inside a use directory
	for _, absUseDir := range absUseDirs {
		if isInsideDir(cwd, absUseDir) {
			// CWD is already inside a workspace member; set env vars and return
			if forgeRepoDir != "" {
				setWorkspaceEnv(forgeRepoDir)
			}
			return nil
		}
	}

	// CWD is not inside a use directory (workspace root or elsewhere in tree).
	// Find the forge repo member and chdir to it.
	if forgeRepoDir != "" {
		if err := os.Chdir(forgeRepoDir); err != nil {
			return fmt.Errorf("cannot change to forge repo member %q: %w", forgeRepoDir, err)
		}
		fmt.Fprintf(os.Stderr, "forge: workspace detected, changed to %s\n", forgeRepoDir)
		setWorkspaceEnv(forgeRepoDir)
		return nil
	}

	// go.work found but no forge repo member
	return nil
}

// setWorkspaceEnv sets the environment variables that enable local development
// mode for engine resolution. forgeRepo is the forge repository directory
// containing cmd/ — used by engine resolution to find engine binaries.
func setWorkspaceEnv(forgeRepo string) {
	_ = os.Setenv("FORGE_RUN_LOCAL_ENABLED", "true")
	_ = os.Setenv("FORGE_RUN_LOCAL_BASEDIR", forgeRepo)
}

// isInsideDir reports whether path is inside (or equal to) dir.
func isInsideDir(path, dir string) bool {
	// Normalize both paths
	path = filepath.Clean(path)
	dir = filepath.Clean(dir)

	if path == dir {
		return true
	}

	return strings.HasPrefix(path, dir+string(filepath.Separator))
}
