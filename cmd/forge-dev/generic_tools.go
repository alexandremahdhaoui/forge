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
	"strings"
	"unicode"
)

// GenericTool is one declared tool resolved into everything the template needs.
type GenericTool struct {
	// Name is the MCP tool name as callers see it.
	Name string
	// GoName is Name in CamelCase, used to build identifiers.
	GoName string
	// FuncType is the generated handler type name, GoName plus Func.
	FuncType string
	// Description is what the tool does.
	Description string
	// InputType is the Go type of the tool input, qualified when spec types
	// live in another package.
	InputType string
	// OutputType is the Go type of the tool output, qualified. Empty when the
	// handler returns only an error.
	OutputType string
	// UseSpec makes the generated wrapper parse and validate Spec before
	// calling the handler.
	UseSpec bool
}

// BuildGenericTools resolves the declared tools into template data. It assumes
// ValidateGenericTools has already passed.
func BuildGenericTools(config *Config, specTypesCtx *SpecTypesContext) []GenericTool {
	prefix := ""
	if specTypesCtx != nil {
		prefix = specTypesCtx.Prefix
	}

	out := make([]GenericTool, 0, len(config.tools()))

	for _, t := range config.tools() {
		goName := camel(t.Name)

		tool := GenericTool{
			Name:        t.Name,
			GoName:      goName,
			FuncType:    goName + "Func",
			Description: t.Description,
			InputType:   prefix + t.Input,
			UseSpec:     t.UseSpec,
		}

		if t.Output != "" {
			tool.OutputType = prefix + t.Output
		}

		out = append(out, tool)
	}

	return out
}

// ValidateGenericTools cross references each declared tool against the schemas
// the OpenAPI spec actually produced. ValidateConfig cannot do this, because it
// runs before the spec is parsed.
func ValidateGenericTools(config *Config, types []ForgeTypeDefinition) []ValidationError {
	if config.engineType() != EngineTypeGeneric {
		return nil
	}

	known := make(map[string]ForgeTypeDefinition, len(types))
	for _, t := range types {
		known[t.Name] = t
	}

	var errors []ValidationError

	generated := map[string]string{}

	for i, t := range config.tools() {
		field := fmt.Sprintf("surface.tools[%d]", i)

		errors = append(errors, requireObjectSchema(field+".input", t.Input, known)...)

		if t.Output != "" {
			errors = append(errors, requireObjectSchema(field+".output", t.Output, known)...)
		}

		funcType := camel(t.Name) + "Func"

		if _, clash := known[funcType]; clash {
			errors = append(errors, ValidationError{
				Field: field + ".name",
				Message: fmt.Sprintf(
					"tool name %q generates type %q, which collides with a schema of that name",
					t.Name, funcType),
			})
		}

		if other, clash := generated[funcType]; clash {
			errors = append(errors, ValidationError{
				Field: field + ".name",
				Message: fmt.Sprintf("tool names %q and %q both generate type %q",
					other, t.Name, funcType),
			})
		}

		generated[funcType] = t.Name
	}

	if _, clash := known["Handlers"]; clash {
		errors = append(errors, ValidationError{
			Field:   "surface.tools",
			Message: `a schema named "Handlers" collides with the generated Handlers struct`,
		})
	}

	return errors
}

func requireObjectSchema(field, name string, known map[string]ForgeTypeDefinition) []ValidationError {
	def, ok := known[name]
	if !ok {
		return []ValidationError{{
			Field:   field,
			Message: fmt.Sprintf("schema %q not found in components.schemas", name),
		}}
	}

	if def.IsEnum || def.IsUnion {
		return []ValidationError{{
			Field: field,
			Message: fmt.Sprintf(
				"schema %q is an enum or a union. Tool inputs and outputs must be object schemas.",
				name),
		}}
	}

	return nil
}

// camel turns a tool name into an exported Go identifier. Hyphens and
// underscores separate words, so state-get becomes StateGet.
func camel(name string) string {
	var b strings.Builder

	upper := true

	for _, r := range name {
		if r == '-' || r == '_' {
			upper = true

			continue
		}

		if upper {
			b.WriteRune(unicode.ToUpper(r))
			upper = false

			continue
		}

		b.WriteRune(r)
	}

	return b.String()
}
