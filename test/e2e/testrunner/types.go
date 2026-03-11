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

// TestFile represents a parsed YAML test case file.
type TestFile struct {
	TestCases []TestCase `yaml:"testCases" json:"testCases"`
}

// TestCase represents a single test case.
type TestCase struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Tags        []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Setup       []Step   `yaml:"setup,omitempty" json:"setup,omitempty"`
	Steps       []Step   `yaml:"steps" json:"steps"`
	Teardown    []Step   `yaml:"teardown,omitempty" json:"teardown,omitempty"`
}

// Step represents a single action in a test case.
type Step struct {
	ID          string                 `yaml:"id,omitempty" json:"id,omitempty"`
	Description string                 `yaml:"description,omitempty" json:"description,omitempty"`
	Mode        string                 `yaml:"mode,omitempty" json:"mode,omitempty"` // "" (harness), "cli", "mcp"
	Tool        string                 `yaml:"tool,omitempty" json:"tool,omitempty"` // MCP tool name (when mode=mcp)
	Command     string                 `yaml:"command" json:"command"`               // CLI subcommand or harness command
	Input       map[string]interface{} `yaml:"input,omitempty" json:"input,omitempty"`
	Expected    map[string]interface{} `yaml:"expected,omitempty" json:"expected,omitempty"`
	Capture     map[string]string      `yaml:"capture,omitempty" json:"capture,omitempty"`
}

// TemplateData holds data available for Go template rendering.
type TemplateData struct {
	Binary    string                            // path to forge binary
	Workspace string                            // temp workspace root
	ForgeYAML string                            // path to forge.yaml (if written)
	CWD       string                            // working directory for forge commands
	Env       map[string]string                 // test env vars
	Steps     map[string]map[string]interface{} // captured step results
}
