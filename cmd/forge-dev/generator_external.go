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
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexandremahdhaoui/forge/internal/mcpcaller"
	"gopkg.in/yaml.v3"
)

type GeneratorModel struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description,omitempty"`
	Kind        string                 `json:"kind"`
	Language    string                 `json:"language,omitempty"`
	PackageName string                 `json:"packageName"`
	Layout      map[string]interface{} `json:"layout,omitempty"`
	Runtime     *RuntimeConfig         `json:"runtime,omitempty"`
	OpenAPISpec string                 `json:"openapiSpec,omitempty"`
	ProtoSpec   string                 `json:"protoSpec,omitempty"`
	WiringSpec  string                 `json:"wiringSpec,omitempty"`
	Checksum    string                 `json:"checksum"`
	SrcDir      string                 `json:"srcDir"`
}

type GeneratorOutput struct {
	Files    []GeneratedFile `json:"files"`
	Manifest bool            `json:"manifest,omitempty"`
}

type GeneratedFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func layoutMap(c *Config) map[string]interface{} {
	if c.Layout == nil {
		return nil
	}

	out := map[string]interface{}{}

	for k, v := range c.Layout.Extra {
		out[k] = v
	}

	if len(c.Layout.Tools) > 0 {
		tools := []interface{}{}
		for _, t := range c.Layout.Tools {
			tools = append(tools, map[string]interface{}{
				"name": t.Name, "description": t.Description,
				"input": t.Input, "output": t.Output, "useSpec": t.UseSpec,
			})
		}

		out["tools"] = tools
	}

	if len(c.Layout.Commands) > 0 {
		commands := []interface{}{}
		for _, cmd := range c.Layout.Commands {
			commands = append(commands, map[string]interface{}{
				"name": cmd.Name, "description": cmd.Description,
			})
		}

		out["commands"] = commands
	}

	return out
}

type generatorCaller interface {
	ResolveEngine(engineURI string) (string, []string, error)
	CallMCP(command string, args []string, toolName string, params interface{}) (interface{}, error)
}

var newGeneratorCaller = func() generatorCaller {
	return mcpcaller.NewCaller(Version)
}

type externalCall struct {
	srcDir      string
	config      *Config
	generator   string
	checksum    string
	openAPISpec string
	outputDir   string
	configOnly  bool
}

func generateExternal(call externalCall) ([]string, error) {
	srcDir, config, generatorURI := call.srcDir, call.config, call.generator

	model := GeneratorModel{
		Name:        config.Name,
		Version:     config.Version,
		Description: config.Description,
		Kind:        config.Kind,
		Language:    config.Language,
		PackageName: config.Generate.PackageName,
		Runtime:     config.Runtime,
		OpenAPISpec: call.openAPISpec,
		Checksum:    call.checksum,
		SrcDir:      srcDir,
	}

	if !call.configOnly {
		protoSpec, err := config.protoSpecText(srcDir)
		if err != nil {
			return nil, err
		}

		wiringSpec, err := config.wiringSpecText(srcDir)
		if err != nil {
			return nil, err
		}

		model.Layout = layoutMap(config)
		model.ProtoSpec = protoSpec
		model.WiringSpec = wiringSpec
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

	placed := make([]GeneratedFile, 0, len(output.Files))

	for _, file := range output.Files {
		clean, err := placeAnsweredPath(generatorURI, call.outputDir, file.Path)
		if err != nil {
			return nil, err
		}

		placed = append(placed, GeneratedFile{Path: clean, Content: file.Content})
	}

	if err := checkGeneratedPaths(generatorURI, placed); err != nil {
		return nil, err
	}

	written := []string{}

	for _, file := range placed {
		full := filepath.Join(srcDir, file.Path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return nil, fmt.Errorf("creating the directory of %s: %w", file.Path, err)
		}

		if err := os.WriteFile(full, []byte(file.Content), 0o644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", file.Path, err)
		}

		written = append(written, file.Path)
	}

	if err := checkGeneratedAnswer(srcDir, generatorURI, output, written); err != nil {
		return nil, err
	}

	return written, nil
}

func placeAnsweredPath(generatorURI, outputDir, answered string) (string, error) {
	clean := filepath.Clean(answered)
	if escapesEngineDir(clean) {
		return "", fmt.Errorf("generator %s answered a path outside the engine directory: %s", generatorURI, answered)
	}

	if outputDir == "" {
		return clean, nil
	}

	placed := filepath.Clean(filepath.Join(outputDir, clean))
	if escapesEngineDir(placed) {
		return "", fmt.Errorf("generator %s answered a path outside the engine directory: %s", generatorURI, placed)
	}

	return placed, nil
}

func escapesEngineDir(clean string) bool {
	return filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator))
}

const GeneratedCellManifestFile = "zz_generated_cell.yaml"

func checkGeneratedAnswer(srcDir, generatorURI string, output GeneratorOutput, written []string) error {
	if err := checkCellManifest(generatorURI, output, written); err != nil {
		return err
	}

	runnablePath := filepath.Join(srcDir, GeneratedRunnableFile)

	recorded, err := ReadGeneratedFiles(runnablePath)
	if err != nil {
		return err
	}

	return recordGeneratedFiles(runnablePath, append(append([]string{}, recorded...), written...))
}

func sweepStaleGeneratedFiles(srcDir string, previous []string) error {
	kept, err := ReadGeneratedFiles(filepath.Join(srcDir, GeneratedRunnableFile))
	if err != nil {
		return err
	}

	return removeStaleGeneratedFiles(srcDir, previous, kept)
}

var generatedRootNames = map[string]bool{
	"lib.rs":             true,
	"main.rs":            true,
	"mod.rs":             true,
	"__init__.py":        true,
	"index.ts":           true,
	"Chart.yaml":         true,
	"values.yaml":        true,
	"values.schema.json": true,
}

func checkGeneratedPaths(generatorURI string, placed []GeneratedFile) error {
	for _, file := range placed {
		base := filepath.Base(file.Path)
		if strings.HasPrefix(base, "zz_generated") {
			continue
		}

		if !generatedRootNames[base] {
			return fmt.Errorf(
				"checking the answer of generator %s: %s is not named zz_generated",
				generatorURI, file.Path)
		}

		if !HasGeneratedHeader(file.Content) {
			return fmt.Errorf(
				"checking the answer of generator %s: the generated root %s is missing the generated header on line one",
				generatorURI, file.Path)
		}
	}

	return nil
}

func checkCellManifest(generatorURI string, output GeneratorOutput, written []string) error {
	if !output.Manifest {
		return nil
	}

	for _, path := range written {
		if filepath.Base(path) == GeneratedCellManifestFile {
			return nil
		}
	}

	return fmt.Errorf(
		"checking the answer of generator %s: manifest is true and no %s was answered",
		generatorURI, GeneratedCellManifestFile)
}

func removeStaleGeneratedFiles(srcDir string, previous, written []string) error {
	kept := map[string]bool{}
	for _, path := range written {
		kept[path] = true
	}

	for _, path := range previous {
		if kept[path] {
			continue
		}

		if !removableGeneratedPath(srcDir, path) {
			log.Printf("forge-dev: skipped the recorded entry %s, it is not removable, it must be named zz_generated or be a generated root carrying the generated header, and stay inside the engine directory", path)

			continue
		}

		if err := os.Remove(filepath.Join(srcDir, path)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing the stale generated file %s: %w", path, err)
		}

		log.Printf("forge-dev: removed the stale generated file %s", path)
	}

	return nil
}

func removableGeneratedPath(srcDir, path string) bool {
	clean := filepath.Clean(path)
	if escapesEngineDir(clean) {
		return false
	}

	base := filepath.Base(clean)
	if strings.HasPrefix(base, "zz_generated") {
		return true
	}

	if !generatedRootNames[base] {
		return false
	}

	raw, err := os.ReadFile(filepath.Join(srcDir, clean))
	if os.IsNotExist(err) {
		return true
	}

	return err == nil && HasGeneratedHeader(string(raw))
}

func recordGeneratedFiles(runnablePath string, written []string) error {
	raw, err := os.ReadFile(runnablePath)
	if err != nil {
		return fmt.Errorf("reading the runnable manifest %s: %w", runnablePath, err)
	}

	body := strings.TrimRight(stripFilesBlock(string(raw)), "\n") + "\nfiles:\n"
	for _, path := range written {
		body += "  - " + path + "\n"
	}

	if err := os.WriteFile(runnablePath, []byte(body), 0o644); err != nil {
		return fmt.Errorf("writing the runnable manifest %s: %w", runnablePath, err)
	}

	return nil
}

func stripFilesBlock(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line == "files:" {
			return strings.Join(lines[:i], "\n")
		}
	}

	return text
}

func ReadGeneratedFiles(runnablePath string) ([]string, error) {
	raw, err := os.ReadFile(runnablePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("reading the runnable manifest %s: %w", runnablePath, err)
	}

	var recorded struct {
		Files []string `yaml:"files"`
	}

	if err := yaml.Unmarshal(raw, &recorded); err != nil {
		return nil, fmt.Errorf("parsing the runnable manifest %s: %w", runnablePath, err)
	}

	return recorded.Files, nil
}
