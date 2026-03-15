//go:build e2e || unit

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

package testrunner

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// writeForgeYAML writes a forge.yaml file to the workspace directory.
//
// Input fields:
//   - name: project name (string)
//   - artifactStorePath: path for artifact store (string, optional)
//   - build: list of build specs (optional)
//   - test: list of test specs (optional)
//   - engines: list of engine aliases (optional)
//   - _raw: verbatim content to write instead of marshaling (string, optional)
func writeForgeYAML(data *TemplateData, input map[string]interface{}) error {
	forgeYAMLPath := filepath.Join(data.Workspace, "forge.yaml")

	// Ensure .envrc exists in the workspace (forge defaults envFile to .envrc).
	envrcPath := filepath.Join(data.Workspace, ".envrc")
	if _, err := os.Stat(envrcPath); os.IsNotExist(err) {
		if err := os.WriteFile(envrcPath, []byte(""), 0o644); err != nil {
			return fmt.Errorf("write-forge-yaml: creating .envrc: %w", err)
		}
	}

	// If _raw is set, write verbatim content.
	if raw, ok := input["_raw"]; ok {
		rawStr, ok := raw.(string)
		if !ok {
			return fmt.Errorf("write-forge-yaml: '_raw' must be a string, got %T", raw)
		}
		if err := os.WriteFile(forgeYAMLPath, []byte(rawStr), 0o644); err != nil {
			return fmt.Errorf("write-forge-yaml: writing %q: %w", forgeYAMLPath, err)
		}
		data.ForgeYAML = forgeYAMLPath
		data.CWD = data.Workspace
		return nil
	}

	// Build forge.yaml content from input fields.
	config := make(map[string]interface{})
	for key, val := range input {
		config[key] = val
	}

	content, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("write-forge-yaml: marshaling config: %w", err)
	}

	if err := os.WriteFile(forgeYAMLPath, content, 0o644); err != nil {
		return fmt.Errorf("write-forge-yaml: writing %q: %w", forgeYAMLPath, err)
	}

	data.ForgeYAML = forgeYAMLPath
	data.CWD = data.Workspace
	return nil
}

// writeFile writes content to a file at the given path. Both path and content
// are rendered as templates.
//
// Input fields:
//   - path: file path (template)
//   - content: file content (template)
func writeFile(data *TemplateData, input map[string]interface{}) error {
	pathRaw, ok := input["path"]
	if !ok {
		return fmt.Errorf("write-file: missing 'path' in input")
	}
	pathStr, ok := pathRaw.(string)
	if !ok {
		return fmt.Errorf("write-file: 'path' must be a string")
	}

	contentRaw, ok := input["content"]
	if !ok {
		return fmt.Errorf("write-file: missing 'content' in input")
	}
	contentStr, ok := contentRaw.(string)
	if !ok {
		return fmt.Errorf("write-file: 'content' must be a string")
	}

	renderedPath, err := RenderTemplate(pathStr, data)
	if err != nil {
		return fmt.Errorf("write-file: rendering path: %w", err)
	}

	renderedContent, err := RenderTemplate(contentStr, data)
	if err != nil {
		return fmt.Errorf("write-file: rendering content: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(renderedPath), 0o755); err != nil {
		return fmt.Errorf("write-file: creating directory: %w", err)
	}

	if err := os.WriteFile(renderedPath, []byte(renderedContent), 0o644); err != nil {
		return fmt.Errorf("write-file: writing %q: %w", renderedPath, err)
	}

	return nil
}

// checkFileExists verifies that a file exists at the given path.
//
// Input fields:
//   - path: file path (template)
//
// Returns: {"exists": true, "size": <int>}
func checkFileExists(data *TemplateData, input map[string]interface{}) (map[string]interface{}, error) {
	pathRaw, ok := input["path"]
	if !ok {
		return nil, fmt.Errorf("check-file-exists: missing 'path' in input")
	}
	pathStr, ok := pathRaw.(string)
	if !ok {
		return nil, fmt.Errorf("check-file-exists: 'path' must be a string")
	}

	renderedPath, err := RenderTemplate(pathStr, data)
	if err != nil {
		return nil, fmt.Errorf("check-file-exists: rendering path: %w", err)
	}

	info, err := os.Stat(renderedPath)
	if err != nil {
		return nil, fmt.Errorf("check-file-exists: file %q does not exist: %w", renderedPath, err)
	}

	return map[string]interface{}{
		"exists": true,
		"size":   float64(info.Size()),
	}, nil
}

// checkFileAbsent verifies that a file does NOT exist at the given path.
//
// Input fields:
//   - path: file path (template)
func checkFileAbsent(data *TemplateData, input map[string]interface{}) error {
	pathRaw, ok := input["path"]
	if !ok {
		return fmt.Errorf("check-file-absent: missing 'path' in input")
	}
	pathStr, ok := pathRaw.(string)
	if !ok {
		return fmt.Errorf("check-file-absent: 'path' must be a string")
	}

	renderedPath, err := RenderTemplate(pathStr, data)
	if err != nil {
		return fmt.Errorf("check-file-absent: rendering path: %w", err)
	}

	if _, err := os.Stat(renderedPath); err == nil {
		return fmt.Errorf("check-file-absent: file %q exists but should be absent", renderedPath)
	}

	return nil
}

// setEnv sets environment variables in both data.Env and the process environment.
//
// Input fields: key-value pairs to set.
func setEnv(data *TemplateData, input map[string]interface{}) error {
	if data.Env == nil {
		data.Env = make(map[string]string)
	}

	for key, val := range input {
		valStr := fmt.Sprintf("%v", val)
		data.Env[key] = valStr
		if err := os.Setenv(key, valStr); err != nil {
			return fmt.Errorf("set-env: setting %q: %w", key, err)
		}
	}

	return nil
}
