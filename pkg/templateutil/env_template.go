package templateutil

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"
)

// ExpandTemplates recursively walks through a spec map and expands environment variable templates.
//
// Template syntax: {{.Env.VARIABLE_NAME}}
//
// Behavior:
//   - Walks spec map recursively (handles nested maps and arrays)
//   - Finds ALL string values containing templates
//   - Expands templates using provided environment map
//   - Returns error if template references undefined variable
//
// Parameters:
//   - spec: The specification map to expand (may contain nested maps and arrays)
//   - env: Environment variables available for template expansion
//
// Returns:
//   - Expanded spec map (new map, does not modify input)
//   - Error if template expansion fails or references undefined variable
//
// Error handling:
//   - If a template references an undefined variable, returns detailed error with:
//   - Template string
//   - Undefined variable name
//   - List of available environment variables
//   - Example: "template expansion failed: variable 'UNDEFINED_VAR' not found in environment for template '{{.Env.UNDEFINED_VAR}}'. Available: [KUBECONFIG, TESTENV_LCR_FQDN]"
func ExpandTemplates(spec map[string]interface{}, env map[string]string) (map[string]interface{}, error) {
	// Create a copy of the spec to avoid modifying the input
	result := make(map[string]interface{})

	// Recursively expand templates in the spec
	for key, value := range spec {
		expanded, err := expandValue(value, env)
		if err != nil {
			return nil, err
		}
		result[key] = expanded
	}

	return result, nil
}

// expandValue recursively expands templates in a value (string, map, array, or other)
func expandValue(value interface{}, env map[string]string) (interface{}, error) {
	switch v := value.(type) {
	case string:
		// Expand templates in string value
		return expandString(v, env)

	case map[string]interface{}:
		// Recursively expand nested map
		result := make(map[string]interface{})
		for k, val := range v {
			expanded, err := expandValue(val, env)
			if err != nil {
				return nil, err
			}
			result[k] = expanded
		}
		return result, nil

	case []interface{}:
		// Recursively expand array elements
		result := make([]interface{}, len(v))
		for i, val := range v {
			expanded, err := expandValue(val, env)
			if err != nil {
				return nil, err
			}
			result[i] = expanded
		}
		return result, nil

	default:
		// Non-string, non-map, non-array values pass through unchanged
		return value, nil
	}
}

// expandString expands templates in a string value using Go text/template
func expandString(str string, env map[string]string) (string, error) {
	// If no template markers, return unchanged
	if !strings.Contains(str, "{{") {
		return str, nil
	}

	// Create template with custom error handling for missing keys
	tmpl, err := template.New("spec").Option("missingkey=error").Parse(str)
	if err != nil {
		return "", fmt.Errorf("template parsing failed for '%s': %w", str, err)
	}

	// Prepare template data
	data := struct {
		Env map[string]string
	}{
		Env: env,
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		// Extract missing variable name from error if possible
		errMsg := err.Error()

		// Try to extract the variable name from the error message
		// Go template error format: "template: spec:1:X: executing \"spec\" at <.Env.VAR>: map has no entry for key \"VAR\""
		var missingVar string
		if strings.Contains(errMsg, "map has no entry for key") {
			// Extract variable name from error message
			parts := strings.Split(errMsg, "map has no entry for key \"")
			if len(parts) >= 2 {
				varParts := strings.Split(parts[1], "\"")
				if len(varParts) > 0 {
					missingVar = varParts[0]
				}
			}
		}

		// Build list of available environment variables
		availableVars := make([]string, 0, len(env))
		for k := range env {
			availableVars = append(availableVars, k)
		}
		sort.Strings(availableVars)

		// Build detailed error message
		if missingVar != "" {
			return "", fmt.Errorf("template expansion failed: variable '%s' not found in environment for template '%s'. Available: %v",
				missingVar, str, availableVars)
		}

		// Fallback error message if we couldn't extract the variable name
		return "", fmt.Errorf("template expansion failed for '%s': %w. Available environment variables: %v",
			str, err, availableVars)
	}

	return buf.String(), nil
}
