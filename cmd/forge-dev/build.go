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
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// Generated file names for forge-dev output.
const (
	GeneratedSpecFile     = "zz_generated.spec.go"
	GeneratedValidateFile = "zz_generated.validate.go"
	GeneratedMCPFile      = "zz_generated.mcp.go"
	GeneratedMainFile     = "zz_generated.main.go"
	GeneratedDocsFile     = "zz_generated.docs.go"
)

// generate is the main code generation function for forge-dev.
// It reads forge-dev.yaml and spec.openapi.yaml from the input.Src directory,
// computes checksums to check if regeneration is needed, and generates
// all three zz_generated files using the embedded templates.
func generate(ctx context.Context, input mcptypes.BuildInput) (*forge.Artifact, error) {
	srcDir := input.Src
	if srcDir == "" {
		return nil, fmt.Errorf("src directory is required")
	}

	// Make srcDir absolute if it isn't already
	if !filepath.IsAbs(srcDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting current directory: %w", err)
		}
		srcDir = filepath.Join(cwd, srcDir)
	}

	log.Printf("forge-dev: generating code for %s", srcDir)

	// Step 1: Read forge-dev.yaml from input.Src directory
	config, err := ReadConfig(srcDir)
	if err != nil {
		return nil, fmt.Errorf("reading forge-dev.yaml: %w", err)
	}

	// Validate configuration
	if errs := ValidateConfig(config); len(errs) > 0 {
		return nil, fmt.Errorf("invalid forge-dev.yaml: %v", errs[0])
	}

	// Resolve SpecTypesContext if specTypes is configured
	var specTypesCtx *SpecTypesContext
	if config.Generate.SpecTypes != nil && config.Generate.SpecTypes.Enabled {
		var err error
		specTypesCtx, err = ResolveSpecTypesContext(srcDir, config.Generate.SpecTypes)
		if err != nil {
			return nil, fmt.Errorf("resolving spec types context: %w", err)
		}
	}

	// Validate docs/usage.md exists (required, not generated)
	usagePath := filepath.Join(srcDir, "docs", "usage.md")
	if _, err := os.Stat(usagePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("docs/usage.md is required but not found at %s", usagePath)
	}

	// Step 2: Resolve spec.openapi.yaml path (relative to forge-dev.yaml location)
	specPath := filepath.Join(srcDir, config.OpenAPI.SpecPath)
	configPath := filepath.Join(srcDir, ConfigFileName)

	// Step 3: Compute source checksum
	checksum, err := ComputeSourceChecksum(configPath, specPath)
	if err != nil {
		return nil, fmt.Errorf("computing source checksum: %w", err)
	}

	// Step 4: Check if regeneration is needed (compare checksums from existing generated files)
	// Skip checksum comparison if force flag is set
	// Determine where spec file is/will be located. A non-go language
	// carries its checksum in its generated server file instead.
	specFilePath := filepath.Join(srcDir, GeneratedSpecFile)
	if specTypesCtx != nil {
		specFilePath = filepath.Join(specTypesCtx.OutputDir, GeneratedSpecFile)
	}
	if langFile, ok := LangMainFiles[config.Language]; ok {
		specFilePath = filepath.Join(srcDir, langFile)
	}
	if config.Kind == KindCLI {
		specFilePath = filepath.Join(srcDir, GeneratedCLIFile)
	}
	if config.Kind == KindBinary || config.Generator != "" {
		specFilePath = filepath.Join(srcDir, GeneratedRunnableFile)
	}
	if !input.Force {
		existingChecksum, err := ReadChecksumFromFile(specFilePath)
		if err != nil {
			return nil, fmt.Errorf("reading existing checksum: %w", err)
		}

		if ChecksumMatches(checksum, existingChecksum) {
			log.Printf("forge-dev: checksums match, skipping regeneration for %s", config.Name)
			// Return artifact with existing files
			return &forge.Artifact{
				Name:      config.Name,
				Type:      "generated",
				Location:  srcDir,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Version:   checksum,
			}, nil
		}
	} else {
		log.Printf("forge-dev: force flag set, regenerating %s", config.Name)
	}

	// Step 5: Load and parse OpenAPI spec using kin-openapi
	spec, err := LoadOpenAPISpec(specPath)
	if err != nil {
		return nil, fmt.Errorf("loading OpenAPI spec: %w", err)
	}

	// Generate types using new adapter
	types, err := GenerateForgeTypes(spec, config.Generate.PackageName)
	if err != nil {
		return nil, fmt.Errorf("generating types: %w", err)
	}

	// A generic mcp-server names its inputs and outputs by schema. Cross
	// reference them now, before anything is written, because a missing
	// schema would otherwise surface as a compile error in generated code.
	if config.Kind == KindMCPServer && config.Generator == "" {
		if errs := ValidateGenericTools(config, types); len(errs) > 0 {
			return nil, fmt.Errorf("invalid forge-dev.yaml: %s: %s", errs[0].Field, errs[0].Message)
		}
	}

	// Step 6: Generate the files using templates
	generatedFiles := []string{}

	// The runnable contract is emitted for every language: the inputs a run
	// needs, derived from runtime: and the spec's required keys.
	runnableSchema := ForgeTypesToSpecSchema(types)

	runnablePath := filepath.Join(srcDir, GeneratedRunnableFile)
	runnableContent, err := GenerateRunnableYAML(config, runnableSchema, checksum)
	if err != nil {
		return nil, fmt.Errorf("generating runnable manifest: %w", err)
	}
	if err := os.WriteFile(runnablePath, runnableContent, 0o644); err != nil {
		return nil, fmt.Errorf("writing runnable manifest: %w", err)
	}
	generatedFiles = append(generatedFiles, GeneratedRunnableFile)
	log.Printf("forge-dev: generated %s", runnablePath)

	finish := func() (*forge.Artifact, error) {
		if err := generateSharedDocs(srcDir, config, types, checksum, &generatedFiles); err != nil {
			return nil, err
		}

		log.Printf("forge-dev: successfully generated %d files for %s", len(generatedFiles), config.Name)

		return &forge.Artifact{
			Name:      config.Name,
			Type:      "generated",
			Location:  srcDir,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   checksum,
		}, nil
	}

	// An external generator owns the whole cell: it answers the files and
	// core writes them. The manifest above and the docs below stay core's.
	if config.Generator != "" {
		files, err := generateExternal(srcDir, config, checksum, specPath)
		if err != nil {
			return nil, err
		}

		generatedFiles = append(generatedFiles, files...)

		return finish()
	}

	// The binary kind is an entrypoint the author owns: the manifest and the
	// docs are the whole generated output.
	if config.Kind == KindBinary {
		return finish()
	}

	// The cli kind generates the dispatcher and the main; the author writes
	// the handlers next to it.
	if config.Kind == KindCLI {
		cliContent, err := GenerateCLIFile(config, checksum)
		if err != nil {
			return nil, fmt.Errorf("generating cli dispatcher: %w", err)
		}

		cliPath := filepath.Join(srcDir, GeneratedCLIFile)
		if err := os.WriteFile(cliPath, cliContent, 0o644); err != nil {
			return nil, fmt.Errorf("writing cli dispatcher: %w", err)
		}

		generatedFiles = append(generatedFiles, GeneratedCLIFile)
		log.Printf("forge-dev: generated %s", cliPath)

		return finish()
	}

	// A non-go language generates one MCP server file from its template set
	// plus the shared docs; the Go code paths do not apply.
	if config.Language != "" && config.Language != "go" {
		langName, langContent, err := GenerateLangMain(config, runnableSchema, checksum)
		if err != nil {
			return nil, fmt.Errorf("generating %s server: %w", config.Language, err)
		}
		langPath := filepath.Join(srcDir, langName)
		if err := os.WriteFile(langPath, langContent, 0o644); err != nil {
			return nil, fmt.Errorf("writing %s server: %w", config.Language, err)
		}
		generatedFiles = append(generatedFiles, langName)
		log.Printf("forge-dev: generated %s", langPath)

		return finish()
	}

	// Generate zz_generated.spec.go
	specContent, err := GenerateSpecFileFromTypes(types, config, checksum, specTypesCtx)
	if err != nil {
		return nil, fmt.Errorf("generating spec file: %w", err)
	}
	// Ensure output directory exists (for external spec types)
	if specTypesCtx != nil {
		if err := os.MkdirAll(specTypesCtx.OutputDir, 0o755); err != nil {
			return nil, fmt.Errorf("creating spec types directory: %w", err)
		}
	}
	if err := os.WriteFile(specFilePath, specContent, 0o644); err != nil {
		return nil, fmt.Errorf("writing spec file: %w", err)
	}
	generatedFiles = append(generatedFiles, GeneratedSpecFile)
	log.Printf("forge-dev: generated %s", specFilePath)

	// Generate zz_generated.validate.go
	validateFilePath := filepath.Join(srcDir, GeneratedValidateFile)
	validateContent, err := GenerateValidateFileFromTypes(types, config, checksum, specTypesCtx)
	if err != nil {
		return nil, fmt.Errorf("generating validate file: %w", err)
	}
	if err := os.WriteFile(validateFilePath, validateContent, 0o644); err != nil {
		return nil, fmt.Errorf("writing validate file: %w", err)
	}
	generatedFiles = append(generatedFiles, GeneratedValidateFile)
	log.Printf("forge-dev: generated %s", validateFilePath)

	// Generate zz_generated.mcp.go
	mcpFilePath := filepath.Join(srcDir, GeneratedMCPFile)
	mcpContent, err := GenerateMCPFile(config, checksum, specTypesCtx)
	if err != nil {
		return nil, fmt.Errorf("generating mcp file: %w", err)
	}
	if err := os.WriteFile(mcpFilePath, mcpContent, 0o644); err != nil {
		return nil, fmt.Errorf("writing mcp file: %w", err)
	}
	generatedFiles = append(generatedFiles, GeneratedMCPFile)
	log.Printf("forge-dev: generated %s", mcpFilePath)

	// Generate zz_generated.main.go
	mainFilePath := filepath.Join(srcDir, GeneratedMainFile)
	mainContent, err := GenerateMainFile(config, checksum, specTypesCtx)
	if err != nil {
		return nil, fmt.Errorf("generating main file: %w", err)
	}
	if err := os.WriteFile(mainFilePath, mainContent, 0o644); err != nil {
		return nil, fmt.Errorf("writing main file: %w", err)
	}
	generatedFiles = append(generatedFiles, GeneratedMainFile)
	log.Printf("forge-dev: generated %s", mainFilePath)

	// Generate zz_generated.docs.go
	docsFilePath := filepath.Join(srcDir, GeneratedDocsFile)
	docsContent, err := GenerateDocsFile(config, checksum)
	if err != nil {
		return nil, fmt.Errorf("generating docs file: %w", err)
	}
	if err := os.WriteFile(docsFilePath, docsContent, 0o644); err != nil {
		return nil, fmt.Errorf("writing docs file: %w", err)
	}
	generatedFiles = append(generatedFiles, GeneratedDocsFile)
	log.Printf("forge-dev: generated %s", docsFilePath)

	return finish()
}

// generateSharedDocs emits the language-agnostic docs: docs/schema.md and
// docs/list.yaml. Both the Go path and the non-go language path use it.
func generateSharedDocs(srcDir string, config *Config, types []ForgeTypeDefinition, checksum string, generatedFiles *[]string) error {
	docsDir := filepath.Join(srcDir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		return fmt.Errorf("creating docs directory: %w", err)
	}

	schemaMDPath := filepath.Join(docsDir, "schema.md")
	schema := ForgeTypesToSpecSchema(types)
	schemaMDContent, err := GenerateSchemaMD(schema, config)
	if err != nil {
		return fmt.Errorf("generating schema.md: %w", err)
	}
	if err := os.WriteFile(schemaMDPath, schemaMDContent, 0o644); err != nil {
		return fmt.Errorf("writing schema.md: %w", err)
	}
	*generatedFiles = append(*generatedFiles, "docs/schema.md")
	log.Printf("forge-dev: generated %s", schemaMDPath)

	listYAMLPath := filepath.Join(docsDir, "list.yaml")
	listYAMLContent, err := GenerateListYAML(config, checksum)
	if err != nil {
		return fmt.Errorf("generating list.yaml: %w", err)
	}
	if err := os.WriteFile(listYAMLPath, listYAMLContent, 0o644); err != nil {
		return fmt.Errorf("writing list.yaml: %w", err)
	}
	*generatedFiles = append(*generatedFiles, "docs/list.yaml")
	log.Printf("forge-dev: generated %s", listYAMLPath)

	return nil
}
