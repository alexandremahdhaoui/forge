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

// SpecTemplateData contains the data passed to the spec.go.tmpl template.
type SpecTemplateData struct {
	// PackageName is the Go package name for the generated file.
	PackageName string
	// ChecksumHeader is the checksum header line (e.g., "SourceChecksum: sha256:...")
	ChecksumHeader string
	// EngineName is the name of the engine.
	EngineName string
	// Properties contains all Spec properties.
	Properties []PropertySchema
	// HasArrayOrMap indicates if any property is an array or map (needs fmt import).
	HasArrayOrMap bool
}

// GenerateSpecFile generates the zz_generated.spec.go file content.
// It uses the spec.go.tmpl template to generate Go code with:
// - Spec struct with JSON tags
// - FromMap function to parse map[string]interface{} to Spec
// - ToMap function to convert Spec to map[string]interface{}
func GenerateSpecFile(schema *SpecSchema, config *Config, checksum string) ([]byte, error) {
	// Prepare template data
	data := SpecTemplateData{
		PackageName:    config.Generate.PackageName,
		ChecksumHeader: ChecksumHeader(checksum),
		EngineName:     config.Name,
		Properties:     schema.Properties,
		HasArrayOrMap:  hasArrayOrMapType(schema.Properties),
	}

	// Parse and execute template
	tmpl, err := parseTemplate("spec.go.tmpl")
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

// hasArrayOrMapType checks if any property is an array or map type.
// This is used to determine if the fmt import is needed.
func hasArrayOrMapType(properties []PropertySchema) bool {
	for _, p := range properties {
		goType := p.GoType()
		if len(goType) > 2 && goType[:2] == "[]" {
			return true
		}
		if len(goType) > 3 && goType[:3] == "map" {
			return true
		}
	}
	return false
}
