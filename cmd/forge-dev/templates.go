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
	"embed"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// templateFuncs contains helper functions available in templates.
var templateFuncs = template.FuncMap{
	"title":    toTitle,
	"camel":    toCamelCase,
	"goType":   goType,
	"jsonTag":  jsonTag,
	"zeroVal":  zeroValue,
	"isSimple": isSimpleType,

	// Reference handling functions (for PropertySchema - backwards compatibility)
	"isRef":           isRef,
	"refType":         refType,
	"isArrayRef":      isArrayRef,
	"arrayRefType":    arrayRefType,
	"usePointer":      usePointer,
	"itemsUsePointer": itemsUsePointer,

	// Template functions for ForgeProperty (new adapter types)
	"forgeGoType":        forgeGoType,
	"forgeIsRef":         forgeIsRef,
	"forgeRefType":       forgeRefType,
	"forgeIsArrayRef":    forgeIsArrayRef,
	"forgeArrayRefType":  forgeArrayRefType,
	"forgeUsePointer":    forgeUsePointer,
	"forgeIsArray":       forgeIsArray,
	"forgeArrayItemType": forgeArrayItemType,
	"forgeIsMap":         forgeIsMap,
	"forgeMapValueType":  forgeMapValueType,
	"forgeIsEnum":        forgeIsEnum,
	"forgeEnumValues":    forgeEnumValues,

	// Template functions for ForgeTypeDefinition
	"isUnion":        isUnion,
	"discField":      discField,
	"discMapping":    discMapping,
	"isTypeEnum":     isTypeEnum,
	"unionVariants":  unionVariants,
	"typeEnumValues": typeEnumValues,

	// Helper functions
	"commentify": commentify,
}

// toTitle converts the first character to uppercase.
func toTitle(s string) string {
	if s == "" {
		return ""
	}
	// Only convert if the first character is lowercase
	if s[0] >= 'a' && s[0] <= 'z' {
		return string(s[0]-'a'+'A') + s[1:]
	}
	return s
}

// toCamelCase converts a snake_case or kebab-case string to CamelCase.
func toCamelCase(s string) string {
	if s == "" {
		return ""
	}
	result := ""
	capitalize := true
	for _, c := range s {
		if c == '_' || c == '-' {
			capitalize = true
			continue
		}
		if capitalize {
			if c >= 'a' && c <= 'z' {
				result += string(c - 'a' + 'A')
			} else {
				result += string(c)
			}
			capitalize = false
		} else {
			result += string(c)
		}
	}
	return result
}

// goType returns the Go type for a property.
func goType(p PropertySchema) string {
	return p.GoType()
}

// jsonTag returns the JSON struct tag for a property.
func jsonTag(name string, required bool) string {
	if required {
		return "`json:\"" + name + "\"`"
	}
	return "`json:\"" + name + ",omitempty\"`"
}

// zeroValue returns the zero value expression for a Go type.
func zeroValue(goType string) string {
	switch goType {
	case "string":
		return `""`
	case "bool":
		return "false"
	case "int":
		return "0"
	case "float64":
		return "0.0"
	default:
		if len(goType) > 2 && goType[:2] == "[]" {
			return "nil"
		}
		if len(goType) > 3 && goType[:3] == "map" {
			return "nil"
		}
		return "nil"
	}
}

// isSimpleType returns true if the type is a simple primitive type.
func isSimpleType(goType string) bool {
	switch goType {
	case "string", "bool", "int", "float64":
		return true
	default:
		return false
	}
}

// parseTemplate parses a template from the embedded filesystem.
func parseTemplate(name string) (*template.Template, error) {
	return template.New(name).Funcs(templateFuncs).ParseFS(templatesFS, "templates/"+name)
}

// isRef returns true if the property is a $ref to another schema.
func isRef(p PropertySchema) bool {
	return p.Ref != ""
}

// refType returns the name of the referenced schema.
// Returns an empty string if RefResolved is nil.
func refType(p PropertySchema) string {
	if p.RefResolved != nil {
		return p.RefResolved.Name
	}
	return ""
}

// isArrayRef returns true if the property is an array with items that use $ref.
func isArrayRef(p PropertySchema) bool {
	return p.Type == "array" && p.Items != nil && p.Items.Ref != ""
}

// arrayRefType returns the name of the schema referenced by array items.
// Returns an empty string if Items or Items.RefResolved is nil.
func arrayRefType(p PropertySchema) string {
	if p.Items != nil && p.Items.RefResolved != nil {
		return p.Items.RefResolved.Name
	}
	return ""
}

// usePointer returns true if the property should use a pointer type.
// This is set to true for properties that are part of a circular reference.
func usePointer(p PropertySchema) bool {
	return p.UsePointer
}

// itemsUsePointer returns true if array items should use a pointer type.
// This is set to true when array items are part of a circular reference.
func itemsUsePointer(p PropertySchema) bool {
	return p.Items != nil && p.Items.UsePointer
}

// -----------------------------------------------------------------------------
// Template functions for ForgeProperty (new adapter types)
// These functions work with the new ForgeProperty type from oapi_adapter.go.
// They have a "forge" prefix to avoid naming conflicts with existing functions.
// -----------------------------------------------------------------------------

// forgeGoType returns the Go type for a ForgeProperty.
func forgeGoType(p ForgeProperty) string {
	return p.GoType
}

// forgeIsRef returns true if the property is a $ref to another schema.
func forgeIsRef(p ForgeProperty) bool {
	return p.IsRef
}

// forgeRefType returns the name of the referenced schema.
func forgeRefType(p ForgeProperty) string {
	return p.RefType
}

// forgeIsArrayRef returns true if the property is an array with items that use $ref.
func forgeIsArrayRef(p ForgeProperty) bool {
	return p.IsArrayOfRef
}

// forgeArrayRefType returns the name of the schema referenced by array items.
func forgeArrayRefType(p ForgeProperty) string {
	return p.ArrayItemType
}

// forgeUsePointer returns true if the property should use a pointer type.
func forgeUsePointer(p ForgeProperty) bool {
	return p.IsPointer
}

// forgeIsArray returns true if the property is an array.
func forgeIsArray(p ForgeProperty) bool {
	return p.IsArray
}

// forgeArrayItemType returns the array item type.
func forgeArrayItemType(p ForgeProperty) string {
	return p.ArrayItemType
}

// forgeIsMap returns true if the property is a map[string]T.
func forgeIsMap(p ForgeProperty) bool {
	return p.IsMap
}

// forgeMapValueType returns the map value type.
func forgeMapValueType(p ForgeProperty) string {
	return p.MapValueType
}

// forgeIsEnum returns true if the property has enum values.
func forgeIsEnum(p ForgeProperty) bool {
	return p.IsEnum
}

// forgeEnumValues returns the valid enum values for the property.
func forgeEnumValues(p ForgeProperty) []string {
	return p.EnumValues
}

// -----------------------------------------------------------------------------
// Template functions for ForgeTypeDefinition
// These functions work with ForgeTypeDefinition from oapi_adapter.go.
// -----------------------------------------------------------------------------

// isUnion returns true if the type is a union type (oneOf/anyOf).
func isUnion(t ForgeTypeDefinition) bool {
	return t.IsUnion
}

// discField returns the discriminator field name for union types.
func discField(t ForgeTypeDefinition) string {
	return t.DiscriminatorField
}

// discMapping returns the discriminator value to type mapping.
func discMapping(t ForgeTypeDefinition) map[string]string {
	return t.DiscriminatorMapping
}

// isTypeEnum returns true if the type definition is an enum type.
func isTypeEnum(t ForgeTypeDefinition) bool {
	return t.IsEnum
}

// unionVariants returns the variant type names for union types.
func unionVariants(t ForgeTypeDefinition) []string {
	return t.UnionVariants
}

// typeEnumValues returns the enum values for enum type definitions.
func typeEnumValues(t ForgeTypeDefinition) []string {
	return t.EnumValues
}

// commentify converts a string (potentially multi-line) to a Go comment block.
// Each line is prefixed with "// ".
func commentify(s string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	var result []string
	for _, line := range lines {
		// Trim trailing whitespace but keep leading whitespace for indentation
		line = strings.TrimRight(line, " \t\r")
		if line == "" {
			result = append(result, "//")
		} else {
			result = append(result, "// "+line)
		}
	}
	return strings.Join(result, "\n")
}
