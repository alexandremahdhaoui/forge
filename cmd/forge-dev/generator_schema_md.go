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
)

// SchemaMDTemplateData contains the data passed to the schema.md.tmpl template.
type SchemaMDTemplateData struct {
	// EngineName is the name of the engine.
	EngineName string
	// Title is the title for the schema documentation.
	Title string
	// Description is the engine description.
	Description string
	// Properties contains the documented properties.
	Properties []PropertyDoc
	// SpecURL is the relative path to the OpenAPI spec file.
	SpecURL string
}

// PropertyDoc represents a property for documentation purposes.
type PropertyDoc struct {
	// Name is the property name.
	Name string
	// Type is the human-readable type string.
	Type string
	// Required indicates if this property is required.
	Required bool
	// Description is the property description.
	Description string
	// Default is the default value as a string (if any).
	Default string
}

// GenerateSchemaMD generates the docs/schema.md file content.
// It uses the schema.md.tmpl template to generate markdown documentation
// describing the engine's configuration options.
func GenerateSchemaMD(schema *SpecSchema, config *Config) ([]byte, error) {
	// Transform OpenAPI properties to PropertyDoc slice
	properties := make([]PropertyDoc, 0, len(schema.Properties))
	for _, prop := range schema.Properties {
		properties = append(properties, PropertyDoc{
			Name:        prop.Name,
			Type:        propertyTypeToString(prop),
			Required:    prop.Required,
			Description: prop.Description,
			Default:     defaultValueToString(prop.Default),
		})
	}

	// SpecURL is always relative from docs/schema.md to spec.openapi.yaml
	// which is one level up from the docs directory
	specURL := "../spec.openapi.yaml"

	// Prepare template data
	data := SchemaMDTemplateData{
		EngineName:  config.Name,
		Title:       fmt.Sprintf("%s Configuration", config.Name),
		Description: config.Description,
		Properties:  properties,
		SpecURL:     specURL,
	}

	// Parse and execute template
	tmpl, err := parseTemplate("schema.md.tmpl")
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	// No gofmt needed - it's markdown
	return buf.Bytes(), nil
}

// propertyTypeToString converts a PropertySchema to a human-readable type string.
func propertyTypeToString(prop PropertySchema) string {
	switch prop.Type {
	case "string":
		if len(prop.Enum) > 0 {
			return "string (enum)"
		}
		return "string"
	case "boolean":
		return "boolean"
	case "integer":
		return "integer"
	case "number":
		return "number"
	case "array":
		if prop.Items != nil {
			return fmt.Sprintf("array of %s", propertyTypeToString(*prop.Items))
		}
		return "array"
	case "object":
		if prop.AdditionalProperties != nil {
			return fmt.Sprintf("map[string]%s", propertyTypeToString(*prop.AdditionalProperties))
		}
		return "object"
	default:
		return prop.Type
	}
}

// defaultValueToString converts a default value to a string representation.
func defaultValueToString(value interface{}) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}
