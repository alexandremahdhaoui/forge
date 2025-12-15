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
	"sort"

	"gopkg.in/yaml.v3"
)

// SpecSchema represents the extracted Spec schema from an OpenAPI specification.
type SpecSchema struct {
	// Properties contains all property definitions.
	Properties []PropertySchema
	// Required lists the names of required properties.
	Required []string
}

// PropertySchema represents a single property in the Spec schema.
type PropertySchema struct {
	// Name is the property name.
	Name string
	// Type is the OpenAPI type (string, boolean, integer, number, array, object).
	Type string
	// Description is the property description from OpenAPI.
	Description string
	// Required indicates if this property is required.
	Required bool
	// Default is the default value if specified.
	Default interface{}
	// Items contains the schema for array items (only for type=array).
	Items *PropertySchema
	// AdditionalProperties contains the schema for map values (only for type=object with additionalProperties).
	AdditionalProperties *PropertySchema
	// Properties contains nested object properties (only for type=object with properties).
	Properties []PropertySchema
	// Enum lists allowed values for enum fields (only for type=string with enum).
	Enum []string
}

// openAPIDocument represents the structure of an OpenAPI document.
type openAPIDocument struct {
	OpenAPI    string            `yaml:"openapi"`
	Info       openAPIInfo       `yaml:"info"`
	Components openAPIComponents `yaml:"components"`
}

type openAPIInfo struct {
	Title   string `yaml:"title"`
	Version string `yaml:"version"`
}

type openAPIComponents struct {
	Schemas map[string]openAPISchema `yaml:"schemas"`
}

type openAPISchema struct {
	Type                 string                   `yaml:"type"`
	Description          string                   `yaml:"description,omitempty"`
	Properties           map[string]openAPISchema `yaml:"properties,omitempty"`
	Required             []string                 `yaml:"required,omitempty"`
	Items                *openAPISchema           `yaml:"items,omitempty"`
	AdditionalProperties *openAPISchema           `yaml:"additionalProperties,omitempty"`
	Enum                 []string                 `yaml:"enum,omitempty"`
	Default              interface{}              `yaml:"default,omitempty"`
	Ref                  string                   `yaml:"$ref,omitempty"`
	OneOf                []openAPISchema          `yaml:"oneOf,omitempty"`
	AnyOf                []openAPISchema          `yaml:"anyOf,omitempty"`
	AllOf                []openAPISchema          `yaml:"allOf,omitempty"`
}

// ParseOpenAPISpec reads and parses an OpenAPI spec file, extracting the Spec schema.
func ParseOpenAPISpec(path string) (*SpecSchema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading OpenAPI spec %s: %w", path, err)
	}

	var doc openAPIDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing OpenAPI spec %s: %w", path, err)
	}

	// Validate OpenAPI version
	if doc.OpenAPI == "" {
		return nil, fmt.Errorf("missing 'openapi' field in %s", path)
	}

	// Get Spec schema
	specSchema, ok := doc.Components.Schemas["Spec"]
	if !ok {
		return nil, fmt.Errorf("missing 'components.schemas.Spec' in %s", path)
	}

	// Validate Spec is an object
	if specSchema.Type != "object" {
		return nil, fmt.Errorf("'components.schemas.Spec' must be type: object, got %q in %s", specSchema.Type, path)
	}

	// Convert to SpecSchema
	schema, err := convertSchema(&specSchema, path)
	if err != nil {
		return nil, err
	}

	// Mark required properties
	requiredSet := make(map[string]bool)
	for _, name := range schema.Required {
		requiredSet[name] = true
	}
	for i := range schema.Properties {
		schema.Properties[i].Required = requiredSet[schema.Properties[i].Name]
	}

	return schema, nil
}

// convertSchema converts an openAPISchema to a SpecSchema.
func convertSchema(schema *openAPISchema, path string) (*SpecSchema, error) {
	// Check for unsupported features
	if schema.Ref != "" {
		return nil, fmt.Errorf("$ref is not supported in %s", path)
	}
	if len(schema.OneOf) > 0 {
		return nil, fmt.Errorf("oneOf is not supported in %s", path)
	}
	if len(schema.AnyOf) > 0 {
		return nil, fmt.Errorf("anyOf is not supported in %s", path)
	}
	if len(schema.AllOf) > 0 {
		return nil, fmt.Errorf("allOf is not supported in %s", path)
	}

	result := &SpecSchema{
		Properties: make([]PropertySchema, 0, len(schema.Properties)),
		Required:   schema.Required,
	}

	// Convert properties in sorted order for deterministic output
	names := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		prop := schema.Properties[name]
		converted, err := convertProperty(name, &prop, path)
		if err != nil {
			return nil, err
		}
		result.Properties = append(result.Properties, *converted)
	}

	return result, nil
}

// convertProperty converts an openAPISchema property to a PropertySchema.
func convertProperty(name string, schema *openAPISchema, path string) (*PropertySchema, error) {
	// Check for unsupported features
	if schema.Ref != "" {
		return nil, fmt.Errorf("$ref in property %q is not supported in %s", name, path)
	}
	if len(schema.OneOf) > 0 {
		return nil, fmt.Errorf("oneOf in property %q is not supported in %s", name, path)
	}
	if len(schema.AnyOf) > 0 {
		return nil, fmt.Errorf("anyOf in property %q is not supported in %s", name, path)
	}
	if len(schema.AllOf) > 0 {
		return nil, fmt.Errorf("allOf in property %q is not supported in %s", name, path)
	}

	prop := &PropertySchema{
		Name:        name,
		Type:        schema.Type,
		Description: schema.Description,
		Default:     schema.Default,
		Enum:        schema.Enum,
	}

	switch schema.Type {
	case "string", "boolean", "integer", "number":
		// Simple types - no additional conversion needed
	case "array":
		if schema.Items == nil {
			return nil, fmt.Errorf("array property %q missing 'items' in %s", name, path)
		}
		// Check for unsupported array of objects
		if schema.Items.Type == "object" && len(schema.Items.Properties) > 0 {
			return nil, fmt.Errorf("array of objects in property %q is not supported in %s", name, path)
		}
		items, err := convertProperty(name+".items", schema.Items, path)
		if err != nil {
			return nil, err
		}
		prop.Items = items
	case "object":
		// Object with additionalProperties (map)
		if schema.AdditionalProperties != nil {
			addProps, err := convertProperty(name+".additionalProperties", schema.AdditionalProperties, path)
			if err != nil {
				return nil, err
			}
			prop.AdditionalProperties = addProps
		}
		// Object with properties (nested object)
		if len(schema.Properties) > 0 {
			nestedSchema, err := convertSchema(schema, path)
			if err != nil {
				return nil, err
			}
			prop.Properties = nestedSchema.Properties
			// Mark nested required properties
			requiredSet := make(map[string]bool)
			for _, reqName := range schema.Required {
				requiredSet[reqName] = true
			}
			for i := range prop.Properties {
				prop.Properties[i].Required = requiredSet[prop.Properties[i].Name]
			}
		}
	default:
		if schema.Type == "" {
			return nil, fmt.Errorf("property %q missing 'type' in %s", name, path)
		}
		return nil, fmt.Errorf("unsupported type %q for property %q in %s", schema.Type, name, path)
	}

	return prop, nil
}

// GoType returns the Go type for this property.
func (p *PropertySchema) GoType() string {
	switch p.Type {
	case "string":
		return "string"
	case "boolean":
		return "bool"
	case "integer":
		return "int"
	case "number":
		return "float64"
	case "array":
		if p.Items != nil {
			return "[]" + p.Items.GoType()
		}
		return "[]interface{}"
	case "object":
		if p.AdditionalProperties != nil {
			return "map[string]" + p.AdditionalProperties.GoType()
		}
		// Nested object - will need a struct name
		return "interface{}"
	default:
		return "interface{}"
	}
}

// IsRequired returns true if the property is required.
func (s *SpecSchema) IsRequired(name string) bool {
	for _, req := range s.Required {
		if req == name {
			return true
		}
	}
	return false
}
