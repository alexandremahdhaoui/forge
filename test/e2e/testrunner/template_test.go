//go:build unit

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
	"testing"
)

func TestRenderTemplate_NoDelimiters(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain string",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "string with braces but no template",
			input: "json: {key: value}",
			want:  "json: {key: value}",
		},
		{
			name:  "path-like string",
			input: "/usr/local/bin/forge",
			want:  "/usr/local/bin/forge",
		},
	}

	data := &TemplateData{Binary: "/bin/forge"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderTemplate(tt.input, data)
			if err != nil {
				t.Fatalf("RenderTemplate() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("RenderTemplate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderTemplate_SimpleField(t *testing.T) {
	data := &TemplateData{
		Binary:    "/usr/local/bin/forge",
		Workspace: "/tmp/test-workspace",
		ForgeYAML: "/tmp/test-workspace/forge.yaml",
		CWD:       "/tmp/test-workspace",
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Binary substitution",
			input: "{{.Binary}}",
			want:  "/usr/local/bin/forge",
		},
		{
			name:  "Workspace substitution",
			input: "{{.Workspace}}",
			want:  "/tmp/test-workspace",
		},
		{
			name:  "ForgeYAML substitution",
			input: "{{.ForgeYAML}}",
			want:  "/tmp/test-workspace/forge.yaml",
		},
		{
			name:  "CWD substitution",
			input: "{{.CWD}}",
			want:  "/tmp/test-workspace",
		},
		{
			name:  "mixed text and template",
			input: "binary is at {{.Binary}} in {{.Workspace}}",
			want:  "binary is at /usr/local/bin/forge in /tmp/test-workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderTemplate(tt.input, data)
			if err != nil {
				t.Fatalf("RenderTemplate() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("RenderTemplate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderTemplate_IndexFunction(t *testing.T) {
	data := &TemplateData{
		Steps: map[string]map[string]interface{}{
			"create": {
				"stdout": "test-e2e-stub-20240101-abc123",
				"code":   float64(0),
			},
		},
	}

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "index into Steps",
			input:   `{{ index .Steps "create" "stdout" }}`,
			want:    "test-e2e-stub-20240101-abc123",
			wantErr: false,
		},
		{
			name:    "index missing step single level",
			input:   `{{ index .Steps "nonexistent" }}`,
			want:    "map[]",
			wantErr: false,
		},
		{
			name:    "index missing step double level",
			input:   `{{ index .Steps "nonexistent" "field" }}`,
			want:    "<no value>",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderTemplate(tt.input, data)
			if tt.wantErr {
				if err == nil {
					t.Errorf("RenderTemplate() expected error, got nil (result: %q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("RenderTemplate() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("RenderTemplate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderMapValues_StringValues(t *testing.T) {
	data := &TemplateData{
		Binary:    "/bin/forge",
		Workspace: "/tmp/ws",
	}

	input := map[string]interface{}{
		"path":    "{{.Workspace}}/output",
		"binary":  "{{.Binary}}",
		"literal": "no-template-here",
	}

	got, err := RenderMapValues(input, data)
	if err != nil {
		t.Fatalf("RenderMapValues() unexpected error: %v", err)
	}

	expected := map[string]interface{}{
		"path":    "/tmp/ws/output",
		"binary":  "/bin/forge",
		"literal": "no-template-here",
	}

	for k, want := range expected {
		if got[k] != want {
			t.Errorf("RenderMapValues()[%q] = %v, want %v", k, got[k], want)
		}
	}
}

func TestRenderMapValues_NestedMap(t *testing.T) {
	data := &TemplateData{
		Workspace: "/tmp/ws",
	}

	input := map[string]interface{}{
		"outer": map[string]interface{}{
			"inner": "{{.Workspace}}/nested",
		},
	}

	got, err := RenderMapValues(input, data)
	if err != nil {
		t.Fatalf("RenderMapValues() unexpected error: %v", err)
	}

	outer, ok := got["outer"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected outer to be map[string]interface{}, got %T", got["outer"])
	}

	if inner := outer["inner"]; inner != "/tmp/ws/nested" {
		t.Errorf("nested value = %v, want %v", inner, "/tmp/ws/nested")
	}
}

func TestRenderMapValues_NonStringPassthrough(t *testing.T) {
	data := &TemplateData{Binary: "/bin/forge"}

	input := map[string]interface{}{
		"count":   42,
		"enabled": true,
		"ratio":   3.14,
		"nothing": nil,
		"items":   []interface{}{"a", "b"},
	}

	got, err := RenderMapValues(input, data)
	if err != nil {
		t.Fatalf("RenderMapValues() unexpected error: %v", err)
	}

	if got["count"] != 42 {
		t.Errorf("count = %v, want 42", got["count"])
	}
	if got["enabled"] != true {
		t.Errorf("enabled = %v, want true", got["enabled"])
	}
	if got["ratio"] != 3.14 {
		t.Errorf("ratio = %v, want 3.14", got["ratio"])
	}
	if got["nothing"] != nil {
		t.Errorf("nothing = %v, want nil", got["nothing"])
	}
	items, ok := got["items"].([]interface{})
	if !ok {
		t.Fatalf("items = %T, want []interface{}", got["items"])
	}
	if len(items) != 2 || items[0] != "a" || items[1] != "b" {
		t.Errorf("items = %v, want [a b]", items)
	}
}
