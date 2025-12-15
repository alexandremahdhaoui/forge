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
