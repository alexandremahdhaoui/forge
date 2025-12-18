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
	"sort"
	"strings"
)

// SpecSchema represents the extracted Spec schema from an OpenAPI specification.
type SpecSchema struct {
	// Properties contains all property definitions.
	Properties []PropertySchema
	// Required lists the names of required properties.
	Required []string
}

// SchemaRegistry holds all named schemas from an OpenAPI specification.
// It provides topologically sorted access for code generation.
type SchemaRegistry struct {
	schemas map[string]*NamedSchema
	order   []string // topologically sorted schema names
}

// NewSchemaRegistry creates a new empty SchemaRegistry.
func NewSchemaRegistry() *SchemaRegistry {
	return &SchemaRegistry{
		schemas: make(map[string]*NamedSchema),
		order:   nil,
	}
}

// Register adds a named schema to the registry.
func (r *SchemaRegistry) Register(name string, schema *NamedSchema) {
	r.schemas[name] = schema
}

// Get returns the schema with the given name, or nil if not found.
func (r *SchemaRegistry) Get(name string) *NamedSchema {
	return r.schemas[name]
}

// GetSpec returns the schema named "Spec", or nil if not found.
func (r *SchemaRegistry) GetSpec() *NamedSchema {
	return r.schemas["Spec"]
}

// GetGenerationOrder returns the topologically sorted list of schema names.
// Returns nil if ComputeOrder has not been called.
func (r *SchemaRegistry) GetGenerationOrder() []string {
	return r.order
}

// ComputeOrder performs a topological sort on the schemas to determine generation order.
// Dependencies (referenced schemas) will be ordered before dependents.
// Cycles are detected using Tarjan's SCC algorithm, and back-edges in cycles are
// marked with UsePointer = true to break the value cycle in generated Go code.
func (r *SchemaRegistry) ComputeOrder() error {
	// Build dependency graph
	// dependencies[A] = [B, C] means A depends on B and C
	dependencies := make(map[string][]string)

	// Collect all schema names first
	names := make([]string, 0, len(r.schemas))
	for name := range r.schemas {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build the dependency graph
	for _, name := range names {
		schema := r.schemas[name]
		deps := r.collectDependencies(schema)
		dependencies[name] = deps
	}

	// Find Strongly Connected Components using Tarjan's algorithm
	sccs := tarjanSCC(names, dependencies)

	// Mark back-edges in cycles with UsePointer = true
	r.markCyclicReferences(sccs, dependencies)

	// Topological sort of SCCs (each SCC is treated as a single unit)
	order := topologicalSortSCCs(sccs, dependencies)

	r.order = order
	return nil
}

// collectDependencies extracts all schema names that the given schema depends on.
func (r *SchemaRegistry) collectDependencies(schema *NamedSchema) []string {
	seen := make(map[string]bool)
	var deps []string

	for _, prop := range schema.Properties {
		// Direct $ref dependency
		if prop.Ref != "" {
			depName := extractSchemaName(prop.Ref)
			if depName != "" && !seen[depName] {
				seen[depName] = true
				deps = append(deps, depName)
			}
		}
		// Array item $ref dependency
		if prop.Items != nil && prop.Items.Ref != "" {
			depName := extractSchemaName(prop.Items.Ref)
			if depName != "" && !seen[depName] {
				seen[depName] = true
				deps = append(deps, depName)
			}
		}
	}

	return deps
}

// extractSchemaName extracts the schema name from a $ref string.
// Example: "#/components/schemas/VMResource" -> "VMResource"
func extractSchemaName(ref string) string {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return ""
	}
	return strings.TrimPrefix(ref, prefix)
}

// tarjanSCC implements Tarjan's algorithm to find Strongly Connected Components.
// Returns a list of SCCs, where each SCC is a list of schema names.
// SCCs are returned in reverse topological order (dependencies come later in the list).
func tarjanSCC(nodes []string, deps map[string][]string) [][]string {
	var (
		index   = 0
		stack   []string
		onStack = make(map[string]bool)
		indices = make(map[string]int)
		lowlink = make(map[string]int)
		sccs    [][]string
	)

	var strongconnect func(v string)
	strongconnect = func(v string) {
		// Set the depth index for v to the smallest unused index
		indices[v] = index
		lowlink[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true

		// Consider successors of v
		for _, w := range deps[v] {
			// Only process nodes that are in our set
			if _, exists := indices[w]; !exists {
				// Successor w has not yet been visited; recurse on it
				// First check if w is in our node set
				found := false
				for _, n := range nodes {
					if n == w {
						found = true
						break
					}
				}
				if !found {
					continue
				}
				strongconnect(w)
				if lowlink[w] < lowlink[v] {
					lowlink[v] = lowlink[w]
				}
			} else if onStack[w] {
				// Successor w is in stack and hence in the current SCC
				if indices[w] < lowlink[v] {
					lowlink[v] = indices[w]
				}
			}
		}

		// If v is a root node, pop the stack and generate an SCC
		if lowlink[v] == indices[v] {
			var scc []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			sccs = append(sccs, scc)
		}
	}

	// Process all nodes in sorted order for deterministic output
	for _, v := range nodes {
		if _, visited := indices[v]; !visited {
			strongconnect(v)
		}
	}

	return sccs
}

// markCyclicReferences marks properties in cyclic dependencies with UsePointer = true.
// An SCC with size > 1 indicates a cycle, and all back-edges within the cycle are marked.
func (r *SchemaRegistry) markCyclicReferences(sccs [][]string, deps map[string][]string) {
	for _, scc := range sccs {
		if len(scc) == 1 {
			// Single-node SCC - check for self-reference
			schemaName := scc[0]
			schema := r.schemas[schemaName]
			for i := range schema.Properties {
				prop := &schema.Properties[i]
				// Check direct $ref self-reference
				if prop.Ref != "" && extractSchemaName(prop.Ref) == schemaName {
					prop.UsePointer = true
				}
				// Check array item $ref self-reference
				if prop.Items != nil && prop.Items.Ref != "" && extractSchemaName(prop.Items.Ref) == schemaName {
					prop.Items.UsePointer = true
				}
			}
			continue
		}

		// Multi-node SCC - cycle detected
		// Create a set of schemas in this SCC for quick lookup
		sccSet := make(map[string]bool)
		for _, name := range scc {
			sccSet[name] = true
		}

		// Mark all edges within the SCC as requiring pointers
		// In a cycle, we need to break at least one edge with a pointer
		// For simplicity, we mark ALL references within the SCC as pointers
		for _, schemaName := range scc {
			schema := r.schemas[schemaName]
			for i := range schema.Properties {
				prop := &schema.Properties[i]
				// Check direct $ref
				if prop.Ref != "" {
					refTarget := extractSchemaName(prop.Ref)
					if sccSet[refTarget] {
						prop.UsePointer = true
					}
				}
				// Check array item $ref
				if prop.Items != nil && prop.Items.Ref != "" {
					refTarget := extractSchemaName(prop.Items.Ref)
					if sccSet[refTarget] {
						prop.Items.UsePointer = true
					}
				}
			}
		}
	}
}

// topologicalSortSCCs performs a topological sort on the SCCs based on inter-SCC dependencies.
// Each SCC is treated as a single unit, and schemas within an SCC are sorted alphabetically.
// Returns a flat list of schema names in generation order (dependencies before dependents).
func topologicalSortSCCs(sccs [][]string, deps map[string][]string) []string {
	if len(sccs) == 0 {
		return nil
	}

	// Create a map from node to its SCC index
	nodeToSCC := make(map[string]int)
	for i, scc := range sccs {
		for _, node := range scc {
			nodeToSCC[node] = i
		}
	}

	// Build the SCC dependency graph
	// sccDeps[i] contains the indices of SCCs that SCC i depends on
	sccDeps := make(map[int]map[int]bool)
	for i := range sccs {
		sccDeps[i] = make(map[int]bool)
	}

	for i, scc := range sccs {
		for _, node := range scc {
			for _, dep := range deps[node] {
				depSCC, exists := nodeToSCC[dep]
				if exists && depSCC != i {
					// SCC i depends on SCC depSCC
					sccDeps[i][depSCC] = true
				}
			}
		}
	}

	// Topological sort of SCCs using Kahn's algorithm
	inDegree := make(map[int]int)
	for i := range sccs {
		inDegree[i] = len(sccDeps[i])
	}

	// Build reverse graph (which SCCs depend on me)
	dependents := make(map[int][]int)
	for i, deps := range sccDeps {
		for dep := range deps {
			dependents[dep] = append(dependents[dep], i)
		}
	}

	// Start with SCCs that have no dependencies
	var queue []int
	for i := range sccs {
		if inDegree[i] == 0 {
			queue = append(queue, i)
		}
	}
	// Sort for deterministic output
	sort.Ints(queue)

	var orderedSCCs [][]string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Sort schemas within the SCC alphabetically for deterministic output
		scc := make([]string, len(sccs[current]))
		copy(scc, sccs[current])
		sort.Strings(scc)
		orderedSCCs = append(orderedSCCs, scc)

		// Update dependents
		var newZero []int
		for _, dependent := range dependents[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				newZero = append(newZero, dependent)
			}
		}
		sort.Ints(newZero)
		queue = append(queue, newZero...)
	}

	// Flatten the result
	var result []string
	for _, scc := range orderedSCCs {
		result = append(result, scc...)
	}

	return result
}

// NamedSchema represents a named schema from components.schemas.
// It extends the concept of SpecSchema with composition and union type support.
type NamedSchema struct {
	// Name is the schema name (e.g., "VMResource").
	Name string
	// Type is the schema type (typically "object").
	Type string
	// Properties contains all property definitions.
	Properties []PropertySchema
	// Required lists the names of required properties.
	Required []string
	// AllOf contains schema references for composition (Phase 2).
	AllOf []SchemaRef
	// OneOf contains schema references for union types (Phase 3).
	OneOf []SchemaRef
	// AnyOf contains schema references for flexible unions (Phase 3).
	AnyOf []SchemaRef
	// Discriminator specifies the discriminator for oneOf unions (Phase 3).
	Discriminator *Discriminator
}

// SchemaRef represents a reference to another schema.
// It can be either a $ref string or an inline schema definition.
type SchemaRef struct {
	// Ref is the reference path (e.g., "#/components/schemas/VMResource").
	Ref string
	// Inline is the inline schema definition (used for allOf inline objects).
	Inline *NamedSchema
}

// Discriminator specifies how to determine which variant of a oneOf union is used.
type Discriminator struct {
	// PropertyName is the field used for discrimination.
	PropertyName string
	// Mapping maps discriminator values to schema names.
	Mapping map[string]string
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
	// Ref is the $ref string if this property references another schema.
	Ref string
	// RefResolved is the resolved schema when Ref is set (populated during resolution).
	RefResolved *NamedSchema
	// UsePointer is true if this ref is part of a cycle and needs a pointer type.
	UsePointer bool
}

// GoType returns the Go type for this property.
func (p *PropertySchema) GoType() string {
	// Handle resolved $ref
	if p.RefResolved != nil {
		if p.UsePointer {
			return "*" + p.RefResolved.Name
		}
		return p.RefResolved.Name
	}

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
	case "ref":
		// Unresolved ref - should have been resolved by resolveAllRefs
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
