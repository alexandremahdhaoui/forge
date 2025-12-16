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
	"fmt"
	"go/format"
)

// DocsTemplateData contains the data passed to the docs.go.tmpl template.
type DocsTemplateData struct {
	// PackageName is the Go package name for the generated file.
	PackageName string
	// ChecksumHeader is the checksum header line.
	ChecksumHeader string
	// EngineName is the name of the engine.
	EngineName string
	// LocalDir is the local directory for docs (e.g., "cmd/<engine-name>/docs").
	LocalDir string
	// BaseURL is the base URL for remote docs.
	BaseURL string
	// RequiredDocs lists the required documentation files.
	RequiredDocs []string
}

// DocsBaseURL is the base URL for engine documentation.
const DocsBaseURL = "https://raw.githubusercontent.com/alexandremahdhaoui/forge/refs/heads/main"

// GenerateDocsFile generates the zz_generated.docs.go file content.
// It uses the docs.go.tmpl template to generate Go code with:
// - docsConfig variable for documentation configuration
// - RegisterDocsMCPTools function to register docs-list and docs-get MCP tools
func GenerateDocsFile(config *Config, checksum string) ([]byte, error) {
	// LocalDir is computed as cmd/<engine-name>/docs
	localDir := fmt.Sprintf("cmd/%s/docs", config.Name)

	// Prepare template data
	data := DocsTemplateData{
		PackageName:    config.Generate.PackageName,
		ChecksumHeader: ChecksumHeader(checksum),
		EngineName:     config.Name,
		LocalDir:       localDir,
		BaseURL:        DocsBaseURL,
		RequiredDocs:   []string{"usage", "schema"},
	}

	// Parse and execute template
	tmpl, err := parseTemplate("docs.go.tmpl")
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Return unformatted code for debugging
		return buf.Bytes(), err
	}

	return formatted, nil
}
