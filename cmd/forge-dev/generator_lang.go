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

// GeneratedRunnableFile is the runnable contract every generation emits.
const GeneratedRunnableFile = "zz_generated.runnable.yaml"

// LangMainFiles maps a non-go language to the server file it generates.
var LangMainFiles = map[string]string{
	"python":     "zz_generated_main.py",
	"typescript": "zzGeneratedMain.ts",
	"rust":       "zz_generated_server.rs",
}

var langTemplates = map[string]string{
	"python":     "mcp_generic.py.tmpl",
	"typescript": "mcp_generic.ts.tmpl",
	"rust":       "mcp_generic.rs.tmpl",
}

// LangTemplateData feeds the runnable manifest and the non-go server
// templates. Everything in it derives from forge-dev.yaml and spec.yaml.
type LangTemplateData struct {
	Checksum         string
	Name             string
	Version          string
	Description      string
	Tools            []ToolConfig
	RequiredSpecKeys []string
	Env              []string
	Files            []string
}

func langTemplateData(config *Config, schema *SpecSchema, checksum string) LangTemplateData {
	data := LangTemplateData{
		Checksum:    checksum,
		Name:        config.Name,
		Version:     config.Version,
		Description: config.Description,
		Tools:       config.Generate.Tools,
	}

	if schema != nil {
		for _, prop := range schema.Properties {
			if prop.Required {
				data.RequiredSpecKeys = append(data.RequiredSpecKeys, prop.Name)
			}
		}
	}

	if config.Runtime != nil {
		data.Env = config.Runtime.Env
		data.Files = config.Runtime.Files
	}

	return data
}

// GenerateRunnableYAML emits the runnable contract: the inputs a run of this
// engine needs, derived from runtime: and the spec's required keys. It is
// yaml, so it serves every language.
func GenerateRunnableYAML(config *Config, schema *SpecSchema, checksum string) ([]byte, error) {
	return renderLangTemplate("runnable.yaml.tmpl", config, schema, checksum)
}

// GenerateLangMain emits the MCP server for a non-go language.
func GenerateLangMain(config *Config, schema *SpecSchema, checksum string) (string, []byte, error) {
	name, ok := LangMainFiles[config.Language]
	if !ok {
		return "", nil, fmt.Errorf("no template set for language %q", config.Language)
	}

	content, err := renderLangTemplate(langTemplates[config.Language], config, schema, checksum)
	if err != nil {
		return "", nil, err
	}

	return name, content, nil
}

func renderLangTemplate(name string, config *Config, schema *SpecSchema, checksum string) ([]byte, error) {
	tmpl, err := parseTemplate(name)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, langTemplateData(config, schema, checksum)); err != nil {
		return nil, fmt.Errorf("rendering %s: %w", name, err)
	}

	return buf.Bytes(), nil
}
