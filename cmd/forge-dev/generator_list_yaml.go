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
	"bytes"
)

// ListYAMLTemplateData contains the data passed to the list.yaml.tmpl template.
type ListYAMLTemplateData struct {
	// ChecksumHeader is the checksum header line (as YAML comment).
	ChecksumHeader string
	// EngineName is the name of the engine.
	EngineName string
	// BaseURL is the base URL for remote docs.
	BaseURL string
}

// GenerateListYAML generates the docs/list.yaml file content.
// It uses the list.yaml.tmpl template to generate the docs registry YAML
// that lists available documentation files (usage and schema).
func GenerateListYAML(config *Config, checksum string) ([]byte, error) {
	// Prepare template data
	// ChecksumHeader for YAML uses # comment prefix instead of //
	data := ListYAMLTemplateData{
		ChecksumHeader: "# SourceChecksum: " + checksum,
		EngineName:     config.Name,
		BaseURL:        DocsBaseURL,
	}

	// Parse and execute template
	tmpl, err := parseTemplate("list.yaml.tmpl")
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	// No gofmt needed - it's YAML
	return buf.Bytes(), nil
}
