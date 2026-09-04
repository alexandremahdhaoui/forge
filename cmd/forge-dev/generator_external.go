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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexandremahdhaoui/forge/internal/mcpcaller"
)

// GeneratorModel is the normalized document an external generator receives.
// It is the whole contract: identity, the kind and its surface, the raw
// OpenAPI spec, the runtime inputs, and the checksum the emitted files
// should carry. forge-dev core keeps parsing, validation, freshness, the
// runnable manifest and the shared docs; the generator only answers files.
type GeneratorModel struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description,omitempty"`
	Kind        string                 `json:"kind"`
	Language    string                 `json:"language,omitempty"`
	PackageName string                 `json:"packageName"`
	Surface     map[string]interface{} `json:"surface,omitempty"`
	Runtime     *RuntimeConfig         `json:"runtime,omitempty"`
	OpenAPISpec string                 `json:"openapiSpec"`
	ProtoSpec   string                 `json:"protoSpec,omitempty"`
	Checksum    string                 `json:"checksum"`
	SrcDir      string                 `json:"srcDir"`
}

// GeneratorOutput is what the generate tool answers: files relative to the
// engine source directory.
type GeneratorOutput struct {
	Files []GeneratedFile `json:"files"`
}

// GeneratedFile is one emitted file.
type GeneratedFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// surfaceMap flattens the surface block for the wire: the typed lists plus
// whatever a custom kind carries.
func surfaceMap(c *Config) map[string]interface{} {
	if c.Surface == nil {
		return nil
	}

	out := map[string]interface{}{}

	for k, v := range c.Surface.Extra {
		out[k] = v
	}

	if len(c.Surface.Tools) > 0 {
		tools := []interface{}{}
		for _, t := range c.Surface.Tools {
			tools = append(tools, map[string]interface{}{
				"name": t.Name, "description": t.Description,
				"input": t.Input, "output": t.Output, "useSpec": t.UseSpec,
			})
		}

		out["tools"] = tools
	}

	if len(c.Surface.Commands) > 0 {
		commands := []interface{}{}
		for _, cmd := range c.Surface.Commands {
			commands = append(commands, map[string]interface{}{
				"name": cmd.Name, "description": cmd.Description,
			})
		}

		out["commands"] = commands
	}

	return out
}

// generatorCaller is the slice of the MCP stack the external dispatch
// needs. mcpcaller.Caller satisfies it; tests inject a stub.
type generatorCaller interface {
	ResolveEngine(engineURI string) (string, []string, error)
	CallMCP(command string, args []string, toolName string, params interface{}) (interface{}, error)
}

// newGeneratorCaller is swapped by tests.
var newGeneratorCaller = func() generatorCaller {
	return mcpcaller.NewCaller(Version)
}

// generateExternal hands the normalized model to generatorURI and writes
// the files it answers. Every path must stay inside the engine source
// directory: a generator emits code, never escapes. The same call serves a
// generator: owning the whole cell and a configGenerator: filling only the
// config surface of a builtin cell.
func generateExternal(srcDir string, config *Config, generatorURI, checksum string) ([]string, error) {
	openAPISpec, err := config.openAPISpecText(srcDir)
	if err != nil {
		return nil, err
	}

	protoSpec, err := config.protoSpecText(srcDir)
	if err != nil {
		return nil, err
	}

	model := GeneratorModel{
		Name:        config.Name,
		Version:     config.Version,
		Description: config.Description,
		Kind:        config.Kind,
		Language:    config.Language,
		PackageName: config.Generate.PackageName,
		Surface:     surfaceMap(config),
		Runtime:     config.Runtime,
		OpenAPISpec: openAPISpec,
		ProtoSpec:   protoSpec,
		Checksum:    checksum,
		SrcDir:      srcDir,
	}

	caller := newGeneratorCaller()

	command, args, err := caller.ResolveEngine(generatorURI)
	if err != nil {
		return nil, fmt.Errorf("resolving generator %s: %w", generatorURI, err)
	}

	raw, err := caller.CallMCP(command, args, "generate", model)
	if err != nil {
		return nil, fmt.Errorf("calling generator %s: %w", generatorURI, err)
	}

	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("encoding the generator answer: %w", err)
	}

	var output GeneratorOutput
	if err := json.Unmarshal(encoded, &output); err != nil {
		return nil, fmt.Errorf("decoding the generator answer: %w", err)
	}

	if len(output.Files) == 0 {
		return nil, fmt.Errorf("generator %s answered no files", generatorURI)
	}

	written := []string{}

	for _, file := range output.Files {
		clean := filepath.Clean(file.Path)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("generator %s answered a path outside the engine directory: %s", generatorURI, file.Path)
		}

		full := filepath.Join(srcDir, clean)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return nil, fmt.Errorf("creating the directory of %s: %w", clean, err)
		}

		if err := os.WriteFile(full, []byte(file.Content), 0o644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", clean, err)
		}

		written = append(written, clean)
	}

	return written, nil
}
