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

// GeneratorModel is the normalized document an external generator receives.
// It is the whole contract: identity, the kind and its layout, the raw
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
	Layout      map[string]interface{} `json:"layout,omitempty"`
	Runtime     *RuntimeConfig         `json:"runtime,omitempty"`
	OpenAPISpec string                 `json:"openapiSpec,omitempty"`
	ProtoSpec   string                 `json:"protoSpec,omitempty"`
	WiringSpec  string                 `json:"wiringSpec,omitempty"`
	Checksum    string                 `json:"checksum"`
	SrcDir      string                 `json:"srcDir"`
}

// GeneratorOutput is what the generate tool answers: files relative to the
// engine source directory, and whether the answer carries a cell manifest.
type GeneratorOutput struct {
	Files    []GeneratedFile `json:"files"`
	Manifest bool            `json:"manifest,omitempty"`
}

// GeneratedFile is one emitted file.
type GeneratedFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// layoutMap flattens the layout block for the wire: the typed lists plus
// whatever a custom kind carries.
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

// externalCall is one dispatch to a generator: which engine, what spec it
// reads, and where its answer lands.
type externalCall struct {
	srcDir      string
	config      *Config
	generator   string
	checksum    string
	openAPISpec string
	outputDir   string
}

// generateExternal hands the normalized model to the generator and writes
// the files it answers. Every path must stay inside the engine source
// directory: a generator emits code, never escapes. The same call serves a
// generator: owning the whole cell and a configGenerator: filling only the
// config keys.
func generateExternal(call externalCall) ([]string, error) {
	srcDir, config, generatorURI := call.srcDir, call.config, call.generator

	protoSpec, err := config.protoSpecText(srcDir)
	if err != nil {
		return nil, err
	}

	wiringSpec, err := config.wiringSpecText(srcDir)
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
		Layout:      layoutMap(config),
		Runtime:     config.Runtime,
		OpenAPISpec: call.openAPISpec,
		ProtoSpec:   protoSpec,
		WiringSpec:  wiringSpec,
		Checksum:    call.checksum,
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

	placed := make([]string, 0, len(output.Files))

	for _, file := range output.Files {
		clean, err := placeAnsweredPath(generatorURI, call.outputDir, file.Path)
		if err != nil {
			return nil, err
		}

		placed = append(placed, clean)
	}

	if err := checkGeneratedPaths(generatorURI, placed); err != nil {
		return nil, err
	}

	written := []string{}

	for i, file := range output.Files {
		clean := placed[i]

		full := filepath.Join(srcDir, clean)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return nil, fmt.Errorf("creating the directory of %s: %w", clean, err)
		}

		if err := os.WriteFile(full, []byte(file.Content), 0o644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", clean, err)
		}

		written = append(written, clean)
	}

	if err := checkGeneratedAnswer(srcDir, generatorURI, output, written); err != nil {
		return nil, err
	}

	return written, nil
}

// placeAnsweredPath moves one answered path under outputDir and refuses
// anything that leaves the engine directory, before or after the move.
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

// GeneratedCellManifestFile is the cell manifest a generator declares with
// manifest: true and must then answer.
const GeneratedCellManifestFile = "zz_generated_cell.yaml"

// checkGeneratedAnswer holds the rules that need the written answer: a
// declared manifest is present and the answer joins the recorded list. The
// path rules run before anything is written. The stale sweep runs once,
// after every generator of the cell has answered.
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

// sweepStaleGeneratedFiles removes what the previous run recorded and no
// generator of this run answered. The recorded list is the union of every
// generator's answer, so a cell with a main generator and a config
// generator keeps both sets.
func sweepStaleGeneratedFiles(srcDir string, previous []string) error {
	kept, err := ReadGeneratedFiles(filepath.Join(srcDir, GeneratedRunnableFile))
	if err != nil {
		return err
	}

	return removeStaleGeneratedFiles(srcDir, previous, kept)
}

func checkGeneratedPaths(generatorURI string, written []string) error {
	for _, path := range written {
		if !strings.HasPrefix(filepath.Base(path), "zz_generated") {
			return fmt.Errorf(
				"checking the answer of generator %s: %s is not named zz_generated",
				generatorURI, path)
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

		if !removableGeneratedPath(path) {
			log.Printf("forge-dev: skipped the recorded entry %s, it is not removable, it must be named zz_generated and stay inside the engine directory", path)

			continue
		}

		if err := os.Remove(filepath.Join(srcDir, path)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing the stale generated file %s: %w", path, err)
		}

		log.Printf("forge-dev: removed the stale generated file %s", path)
	}

	return nil
}

func removableGeneratedPath(path string) bool {
	clean := filepath.Clean(path)
	if escapesEngineDir(clean) {
		return false
	}

	return strings.HasPrefix(filepath.Base(clean), "zz_generated")
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

// ReadGeneratedFiles answers the paths the previous generator answer held,
// as recorded in the runnable manifest.
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
