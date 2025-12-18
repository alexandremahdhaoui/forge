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

// ValidateTemplateData contains the data passed to the validate.go.tmpl template.
type ValidateTemplateData struct {
	// PackageName is the Go package name for the generated file.
	PackageName string
	// ChecksumHeader is the checksum header line.
	ChecksumHeader string
	// EngineName is the name of the engine.
	EngineName string
	// Properties contains all Spec properties.
	Properties []PropertySchema
	// Types contains all type definitions for the new kin-openapi-based generation path.
	// When Types is set, templates iterate over Types to generate validation functions.
	// This is nil for the backwards-compatible path.
	Types []ForgeTypeDefinition
	// MainType is the name of the main entry point type (always "Spec").
	MainType string
	// NeedsFmtImport indicates if the generated code needs the fmt import.
	NeedsFmtImport bool
}

// GenerateValidateFile generates the zz_generated.validate.go file content.
// It uses the validate.go.tmpl template to generate Go code with:
// - Validate function to validate a Spec struct
// - ValidateMap function to validate a map[string]interface{}
func GenerateValidateFile(schema *SpecSchema, config *Config, checksum string) ([]byte, error) {
	// Prepare template data
	data := ValidateTemplateData{
		PackageName:    config.Generate.PackageName,
		ChecksumHeader: ChecksumHeader(checksum),
		EngineName:     config.Name,
		Properties:     schema.Properties,
	}

	// Parse and execute template
	tmpl, err := parseTemplate("validate.go.tmpl")
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
