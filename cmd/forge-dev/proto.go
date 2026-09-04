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
	"os"
	"path/filepath"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/getkin/kin-openapi/openapi3"
)

type ProtoConfig struct {
	SpecPath string `yaml:"specPath"`
}

func (c *Config) declaresOpenAPI() bool {
	return c.OpenAPI.SpecPath != ""
}

func (c *Config) declaresProto() bool {
	return c.Proto.SpecPath != ""
}

func (c *Config) openAPISpecPath(srcDir string) string {
	return filepath.Join(srcDir, c.OpenAPI.SpecPath)
}

func (c *Config) protoSpecPath(srcDir string) string {
	return filepath.Join(srcDir, c.Proto.SpecPath)
}

func (c *Config) specPaths(srcDir string) []string {
	paths := []string{}
	if c.declaresOpenAPI() {
		paths = append(paths, c.openAPISpecPath(srcDir))
	}
	if c.declaresProto() {
		paths = append(paths, c.protoSpecPath(srcDir))
	}
	if c.declaresWiring() {
		paths = append(paths, c.wiringSpecPath(srcDir))
	}

	return paths
}

func (c *Config) validateSpecSources() []ValidationError {
	if c.declaresOpenAPI() {
		return nil
	}

	if !c.declaresProto() && !c.declaresWiring() {
		return []ValidationError{{
			Field:   "openapi.specPath",
			Message: "required field is missing",
		}}
	}

	if c.Generator == "" {
		return []ValidationError{{
			Field:   "generator",
			Message: "a cell that declares proto.specPath or wiring.specPath and no openapi.specPath needs a generator that reads it",
		}}
	}

	return nil
}

func validateCellWithNoOpenAPI(config *Config, configPath string, warnings []mcptypes.ValidationWarning) *mcptypes.ConfigValidateOutput {
	warnings = warnMissingProto(config, configPath, warnings)
	warnings = warnMissingWiring(config, configPath, warnings)

	return &mcptypes.ConfigValidateOutput{Valid: true, Warnings: warnings}
}

func warnMissingProto(config *Config, configPath string, warnings []mcptypes.ValidationWarning) []mcptypes.ValidationWarning {
	if !config.declaresProto() {
		return warnings
	}

	if _, statErr := os.Stat(config.protoSpecPath(configPath)); !os.IsNotExist(statErr) {
		return warnings
	}

	return append(warnings, mcptypes.ValidationWarning{
		Field: "proto.specPath",
		Message: fmt.Sprintf(
			"%s does not exist yet; the build's spec resolution materializes it, so its content is validated at build time",
			config.Proto.SpecPath),
	})
}

func loadCellTypes(config *Config, srcDir string) (*openapi3.T, []ForgeTypeDefinition, error) {
	if !config.declaresOpenAPI() {
		return nil, nil, nil
	}

	spec, err := LoadOpenAPISpec(config.openAPISpecPath(srcDir))
	if err != nil {
		return nil, nil, fmt.Errorf("loading OpenAPI spec: %w", err)
	}

	types, err := GenerateForgeTypes(spec, config.Generate.PackageName)
	if err != nil {
		return nil, nil, fmt.Errorf("generating types: %w", err)
	}

	return spec, types, nil
}

func readSpecText(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading spec %s for the generator: %w", path, err)
	}

	return string(raw), nil
}

func (c *Config) openAPISpecText(srcDir string) (string, error) {
	if !c.declaresOpenAPI() {
		return "", nil
	}

	return readSpecText(c.openAPISpecPath(srcDir))
}

func (c *Config) protoSpecText(srcDir string) (string, error) {
	if !c.declaresProto() {
		return "", nil
	}

	return readSpecText(c.protoSpecPath(srcDir))
}
