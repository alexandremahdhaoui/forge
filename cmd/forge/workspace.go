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

func resolveWorkspace() error {
	if skipWorkspaceResolution {
		return nil
	}

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

	for _, absUseDir := range absUseDirs {
		if isInsideDir(cwd, absUseDir) {
			return nil
		}
	}

	for _, absUseDir := range absUseDirs {
		if !forgepath.IsForgeRepo(absUseDir) {
			continue
		}

		if err := os.Chdir(absUseDir); err != nil {
			return fmt.Errorf("cannot change to forge repo member %q: %w", absUseDir, err)
		}

		fmt.Fprintf(os.Stderr, "forge: workspace detected, changed to %s\n", absUseDir)

		return nil
	}

	return nil
}

func isInsideDir(path, dir string) bool {
	path = filepath.Clean(path)
	dir = filepath.Clean(dir)

	if path == dir {
		return true
	}

	return strings.HasPrefix(path, dir+string(filepath.Separator))
}
