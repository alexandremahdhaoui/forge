// Copyright 2025 Alexandre Mahdhaoui
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
	"sort"

	"github.com/getkin/kin-openapi/openapi3"
)

// ForgeTypeDefinition represents a type to be generated from an OpenAPI schema.
// This is our adapter layer between kin-openapi and forge templates.
type ForgeTypeDefinition struct {
	Name                 string            // Go type name (CamelCase)
	JsonName             string            // Original JSON name from spec
	Description          string            // Schema description
	Properties           []ForgeProperty   // Flattened properties
	IsUnion              bool              // true for oneOf/anyOf types
	UnionVariants        []string          // Variant type names
	DiscriminatorField   string            // Field for discrimination
	DiscriminatorMapping map[string]string // value -> type mapping
	IsEnum               bool              // true for enum types
	EnumValues           []string          // Enum constant names (sorted)
}

// ForgeProperty represents a property in a generated struct.
type ForgeProperty struct {
	Name          string   // Go field name (CamelCase)
	JsonName      string   // JSON field name
	GoType        string   // Full Go type string
	Description   string   // Property description
	Required      bool     // Is required
	Nullable      bool     // Is nullable
	IsRef         bool     // Is a $ref type
	RefType       string   // Referenced type name
	IsArray       bool     // Is an array
	ArrayItemType string   // Array item type
	IsArrayOfRef  bool     // Array of $ref
	IsMap         bool     // Is map[string]T
	MapValueType  string   // Map value type
	IsEnum        bool     // Has enum values
	EnumValues    []string // Valid enum values
	IsPointer     bool     // Type is pointer
}

// LoadOpenAPISpec loads an OpenAPI specification from a file using kin-openapi.
// This replaces the custom YAML parsing in openapi.go.
func LoadOpenAPISpec(path string) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false // Only support internal refs

	spec, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading OpenAPI spec from %s: %w", path, err)
	}

	// Validate OpenAPI version field is present
	if spec.OpenAPI == "" {
		return nil, fmt.Errorf("missing 'openapi' field in %s", path)
	}

	return spec, nil
}

// schemaToGoType converts an OpenAPI schema type and format to a Go type string.
func schemaToGoType(schema *openapi3.Schema) string {
	if schema == nil {
		return "interface{}"
	}

	// Handle array type
	if schema.Type.Is("array") {
		if schema.Items != nil && schema.Items.Value != nil {
			itemType := schemaToGoType(schema.Items.Value)
			return "[]" + itemType
		}
		return "[]interface{}"
	}

	// Handle object type with additionalProperties (map)
	if schema.Type.Is("object") {
		// Check if additionalProperties has a schema (e.g., additionalProperties: type: string)
		if schema.AdditionalProperties.Schema != nil && schema.AdditionalProperties.Schema.Value != nil {
			valueType := schemaToGoType(schema.AdditionalProperties.Schema.Value)
			return "map[string]" + valueType
		}
		// Check if additionalProperties is set to true (allows any additional properties)
		if schema.AdditionalProperties.Has != nil && *schema.AdditionalProperties.Has {
			return "map[string]interface{}"
		}
		// Object with properties - this should be a struct, return interface{} as placeholder
		return "interface{}"
	}

	// Handle primitive types
	switch {
	case schema.Type.Is("string"):
		return "string"
	case schema.Type.Is("integer"):
		switch schema.Format {
		case "int32":
			return "int32"
		case "int64":
			return "int64"
		default:
			return "int"
		}
	case schema.Type.Is("number"):
		switch schema.Format {
		case "float":
			return "float32"
		default:
			return "float64"
		}
	case schema.Type.Is("boolean"):
		return "bool"
	default:
		return "interface{}"
	}
}

// ConvertSchemaProperty converts an OpenAPI SchemaRef to a ForgeProperty.
func ConvertSchemaProperty(name string, schemaRef *openapi3.SchemaRef, required bool) ForgeProperty {
	prop := ForgeProperty{
		Name:     toCamelCase(name),
		JsonName: name,
		Required: required,
	}

	// Handle $ref - the schemaRef.Ref contains the reference path
	if schemaRef.Ref != "" {
		prop.IsRef = true
		prop.RefType = extractSchemaName(schemaRef.Ref)
		prop.GoType = prop.RefType
		return prop
	}

	schema := schemaRef.Value
	if schema == nil {
		prop.GoType = "interface{}"
		return prop
	}

	prop.Description = schema.Description
	prop.Nullable = schema.Nullable

	// Check for enum values
	if len(schema.Enum) > 0 {
		prop.IsEnum = true
		for _, v := range schema.Enum {
			if s, ok := v.(string); ok {
				prop.EnumValues = append(prop.EnumValues, s)
			}
		}
		sort.Strings(prop.EnumValues)
	}

	// Handle array type
	if schema.Type.Is("array") {
		prop.IsArray = true
		if schema.Items != nil {
			// Check if array items are a $ref
			if schema.Items.Ref != "" {
				prop.IsArrayOfRef = true
				prop.ArrayItemType = extractSchemaName(schema.Items.Ref)
				prop.GoType = "[]" + prop.ArrayItemType
			} else if schema.Items.Value != nil {
				prop.ArrayItemType = schemaToGoType(schema.Items.Value)
				prop.GoType = "[]" + prop.ArrayItemType
			} else {
				prop.ArrayItemType = "interface{}"
				prop.GoType = "[]interface{}"
			}
		} else {
			prop.ArrayItemType = "interface{}"
			prop.GoType = "[]interface{}"
		}
		return prop
	}

	// Handle object type with additionalProperties (map)
	// Check if additionalProperties has a schema OR is set to true
	if schema.Type.Is("object") {
		hasAdditionalPropsSchema := schema.AdditionalProperties.Schema != nil
		hasAdditionalPropsTrue := schema.AdditionalProperties.Has != nil && *schema.AdditionalProperties.Has

		if hasAdditionalPropsSchema || hasAdditionalPropsTrue {
			prop.IsMap = true
			if schema.AdditionalProperties.Schema != nil {
				if schema.AdditionalProperties.Schema.Ref != "" {
					prop.MapValueType = extractSchemaName(schema.AdditionalProperties.Schema.Ref)
				} else if schema.AdditionalProperties.Schema.Value != nil {
					prop.MapValueType = schemaToGoType(schema.AdditionalProperties.Schema.Value)
				} else {
					prop.MapValueType = "interface{}"
				}
			} else {
				prop.MapValueType = "interface{}"
			}
			prop.GoType = "map[string]" + prop.MapValueType
			return prop
		}
	}

	// Handle primitive types
	prop.GoType = schemaToGoType(schema)

	// Determine if pointer is needed (nullable and not required)
	if prop.Nullable && !prop.Required {
		prop.IsPointer = true
		prop.GoType = "*" + prop.GoType
	}

	return prop
}

// ConvertSchemaToTypeDefinition converts an OpenAPI schema to a ForgeTypeDefinition.
func ConvertSchemaToTypeDefinition(name string, schemaRef *openapi3.SchemaRef) ForgeTypeDefinition {
	td := ForgeTypeDefinition{
		Name:     name,
		JsonName: name,
	}

	if schemaRef == nil || schemaRef.Value == nil {
		return td
	}

	schema := schemaRef.Value
	td.Description = schema.Description

	// Handle union types (oneOf or anyOf)
	if len(schema.OneOf) > 0 || len(schema.AnyOf) > 0 {
		td.IsUnion = true
		variants := schema.OneOf
		if len(variants) == 0 {
			variants = schema.AnyOf
		}
		for _, variant := range variants {
			if variant.Ref != "" {
				td.UnionVariants = append(td.UnionVariants, extractSchemaName(variant.Ref))
			}
		}
		sort.Strings(td.UnionVariants)

		// Extract discriminator info
		if schema.Discriminator != nil {
			td.DiscriminatorField = schema.Discriminator.PropertyName
			if len(schema.Discriminator.Mapping) > 0 {
				td.DiscriminatorMapping = make(map[string]string)
				for value, ref := range schema.Discriminator.Mapping {
					td.DiscriminatorMapping[value] = extractSchemaName(ref)
				}
			}
		}
		return td
	}

	// Handle enum types
	if len(schema.Enum) > 0 {
		td.IsEnum = true
		for _, v := range schema.Enum {
			if s, ok := v.(string); ok {
				td.EnumValues = append(td.EnumValues, s)
			}
		}
		sort.Strings(td.EnumValues)
		return td
	}

	// Build required set for properties
	requiredSet := make(map[string]bool)
	for _, req := range schema.Required {
		requiredSet[req] = true
	}

	// Get property names in sorted order for deterministic output
	propNames := make([]string, 0, len(schema.Properties))
	for propName := range schema.Properties {
		propNames = append(propNames, propName)
	}
	sort.Strings(propNames)

	// Convert properties
	for _, propName := range propNames {
		propSchemaRef := schema.Properties[propName]
		prop := ConvertSchemaProperty(propName, propSchemaRef, requiredSet[propName])
		td.Properties = append(td.Properties, prop)
	}

	return td
}

// GenerateForgeTypes converts an OpenAPI spec to Forge type definitions.
// Types are returned in topological order (dependencies before dependents).
func GenerateForgeTypes(spec *openapi3.T, pkgName string) ([]ForgeTypeDefinition, error) {
	if spec.Components == nil || spec.Components.Schemas == nil {
		return nil, fmt.Errorf("spec has no components/schemas")
	}

	schemas := spec.Components.Schemas

	// Check for Spec schema
	if _, ok := schemas["Spec"]; !ok {
		return nil, fmt.Errorf("spec must have a 'Spec' schema as the main entry point")
	}

	// Create a SchemaRegistry for topological sorting
	// We reuse the existing SchemaRegistry and its Tarjan's SCC implementation
	registry := NewSchemaRegistry()

	// Get schema names in sorted order for deterministic processing
	schemaNames := make([]string, 0, len(schemas))
	for name := range schemas {
		schemaNames = append(schemaNames, name)
	}
	sort.Strings(schemaNames)

	// Convert all schemas to ForgeTypeDefinition and register for topological sort
	typeMap := make(map[string]ForgeTypeDefinition)
	for _, name := range schemaNames {
		schemaRef := schemas[name]
		td := ConvertSchemaToTypeDefinition(name, schemaRef)
		typeMap[name] = td

		// Create a NamedSchema for the registry (for topological sorting)
		namedSchema := &NamedSchema{
			Name:       name,
			Properties: convertForgePropertiesToPropertySchemas(td.Properties),
		}
		registry.Register(name, namedSchema)
	}

	// Compute topological order using existing Tarjan's SCC algorithm
	if err := registry.ComputeOrder(); err != nil {
		return nil, fmt.Errorf("computing generation order: %w", err)
	}

	// Build result in topological order
	order := registry.GetGenerationOrder()
	types := make([]ForgeTypeDefinition, 0, len(order))
	for _, name := range order {
		if td, ok := typeMap[name]; ok {
			// Update pointer flags based on registry's cycle detection
			if namedSchema := registry.Get(name); namedSchema != nil {
				updatePointerFlags(&td, namedSchema)
			}
			types = append(types, td)
		}
	}

	return types, nil
}

// convertForgePropertiesToPropertySchemas converts ForgeProperty slice to PropertySchema slice
// for use with the existing SchemaRegistry's dependency tracking.
func convertForgePropertiesToPropertySchemas(props []ForgeProperty) []PropertySchema {
	result := make([]PropertySchema, 0, len(props))
	for _, fp := range props {
		ps := PropertySchema{
			Name:     fp.JsonName,
			Type:     "string", // Placeholder, not used for dependency tracking
			Required: fp.Required,
		}
		if fp.IsRef {
			ps.Ref = "#/components/schemas/" + fp.RefType
		}
		if fp.IsArrayOfRef {
			ps.Type = "array"
			ps.Items = &PropertySchema{
				Ref: "#/components/schemas/" + fp.ArrayItemType,
			}
		}
		result = append(result, ps)
	}
	return result
}

// updatePointerFlags updates ForgeTypeDefinition properties with pointer flags
// detected by the SchemaRegistry's cycle detection.
func updatePointerFlags(td *ForgeTypeDefinition, ns *NamedSchema) {
	// Create a map of property names to their UsePointer status
	pointerMap := make(map[string]bool)
	itemsPointerMap := make(map[string]bool)
	for _, ps := range ns.Properties {
		if ps.UsePointer {
			pointerMap[ps.Name] = true
		}
		if ps.Items != nil && ps.Items.UsePointer {
			itemsPointerMap[ps.Name] = true
		}
	}

	// Update ForgeProperty flags
	for i := range td.Properties {
		prop := &td.Properties[i]
		if pointerMap[prop.JsonName] && prop.IsRef {
			prop.IsPointer = true
			prop.GoType = "*" + prop.RefType
		}
		if itemsPointerMap[prop.JsonName] && prop.IsArrayOfRef {
			// Array items should use pointer
			prop.GoType = "[]*" + prop.ArrayItemType
		}
	}
}

// ForgeTypesToSpecSchema converts the "Spec" ForgeTypeDefinition to a SpecSchema
// for backwards compatibility with GenerateSchemaMD.
func ForgeTypesToSpecSchema(types []ForgeTypeDefinition) *SpecSchema {
	// Find the Spec type
	for _, t := range types {
		if t.Name == "Spec" {
			return &SpecSchema{
				Properties: forgePropertiesToPropertySchemas(t.Properties),
			}
		}
	}
	return &SpecSchema{}
}

// forgePropertiesToPropertySchemas converts ForgeProperty slice to PropertySchema slice.
func forgePropertiesToPropertySchemas(props []ForgeProperty) []PropertySchema {
	result := make([]PropertySchema, 0, len(props))
	for _, fp := range props {
		ps := PropertySchema{
			Name:        fp.JsonName,
			Description: fp.Description,
			Required:    fp.Required,
		}

		// Determine type from GoType
		switch {
		case fp.IsRef:
			ps.Ref = "#/components/schemas/" + fp.RefType
		case fp.IsArray:
			ps.Type = "array"
			if fp.IsArrayOfRef {
				ps.Items = &PropertySchema{
					Ref: "#/components/schemas/" + fp.ArrayItemType,
				}
			} else {
				ps.Items = &PropertySchema{
					Type: fp.ArrayItemType,
				}
			}
		case fp.IsMap:
			ps.Type = "object"
			ps.AdditionalProperties = &PropertySchema{
				Type: fp.MapValueType,
			}
		case fp.IsEnum:
			ps.Type = "string"
			ps.Enum = fp.EnumValues
		default:
			// Map Go types back to OpenAPI types
			switch fp.GoType {
			case "string", "*string":
				ps.Type = "string"
			case "int", "int32", "int64", "*int", "*int32", "*int64":
				ps.Type = "integer"
			case "float32", "float64", "*float32", "*float64":
				ps.Type = "number"
			case "bool", "*bool":
				ps.Type = "boolean"
			default:
				ps.Type = "string" // fallback
			}
		}

		result = append(result, ps)
	}
	return result
}
