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
	"go/format"
)

// MainTemplateData contains the data passed to the main.go.tmpl template.
type MainTemplateData struct {
	// PackageName is the Go package name for the generated file.
	PackageName string
	// ChecksumHeader is the checksum header line.
	ChecksumHeader string
	// EngineName is the name of the engine.
	EngineName string
	// EngineType is the type of engine (builder, test-runner, testenv-subengine, dependency-detector).
	EngineType string
	// Version is the engine version.
	Version string
	// Description is the engine description.
	Description string
	// BuildFunc is the build function name for builder engines.
	BuildFunc string
	// RunFunc is the run function name for test-runner engines.
	RunFunc string
	// CreateFunc is the create function name for testenv-subengine engines.
	CreateFunc string
	// DeleteFunc is the delete function name for testenv-subengine engines.
	DeleteFunc string
}

// GenerateMainFile generates the zz_generated.main.go file content.
// It uses the main.go.tmpl template to generate Go code with:
// - main() function calling enginecli.Bootstrap
// - runMCPServer() function calling SetupMCPServer
// - Version information variables
func GenerateMainFile(config *Config, checksum string) ([]byte, error) {
	// Prepare template data
	data := MainTemplateData{
		PackageName:    config.Generate.PackageName,
		ChecksumHeader: ChecksumHeader(checksum),
		EngineName:     config.Name,
		EngineType:     string(config.Type),
		Version:        config.Version,
		Description:    config.Description,
		BuildFunc:      config.GetBuildFunc(),
		RunFunc:        config.GetRunFunc(),
		CreateFunc:     config.GetCreateFunc(),
		DeleteFunc:     config.GetDeleteFunc(),
	}

	// Parse and execute template
	tmpl, err := parseTemplate("main.go.tmpl")
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
