//go:build e2e || unit

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

package testrunner

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// RenderTemplate renders a Go template string with the given TemplateData.
// It supports the built-in index function for nested map access, e.g.:
//
//	{{ index .Steps "step-id" "field" }}
func RenderTemplate(tmpl string, data *TemplateData) (string, error) {
	// Fast path: if the string contains no template delimiters, return as-is.
	if !strings.Contains(tmpl, "{{") {
		return tmpl, nil
	}

	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parsing template %q: %w", tmpl, err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template %q: %w", tmpl, err)
	}

	return buf.String(), nil
}

// RenderMapValues recursively walks a map and renders any string values as Go
// templates. Non-string values are left unchanged.
func RenderMapValues(m map[string]interface{}, data *TemplateData) (map[string]interface{}, error) {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		rendered, err := renderValue(v, data)
		if err != nil {
			return nil, fmt.Errorf("rendering field %q: %w", k, err)
		}
		result[k] = rendered
	}
	return result, nil
}

// renderValue renders a single value: strings are treated as templates,
// maps are recursed, slices are iterated, and everything else passes through.
func renderValue(v interface{}, data *TemplateData) (interface{}, error) {
	switch val := v.(type) {
	case string:
		return RenderTemplate(val, data)
	case map[string]interface{}:
		return RenderMapValues(val, data)
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			rendered, err := renderValue(item, data)
			if err != nil {
				return nil, fmt.Errorf("index %d: %w", i, err)
			}
			result[i] = rendered
		}
		return result, nil
	default:
		return v, nil
	}
}
