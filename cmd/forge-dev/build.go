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
	specFilePath := filepath.Join(srcDir, GeneratedSpecFile)
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

	// Step 5: Parse OpenAPI spec
	schema, err := ParseOpenAPISpec(specPath)
	if err != nil {
		return nil, fmt.Errorf("parsing OpenAPI spec: %w", err)
	}

	// Step 6: Generate all three files using templates
	generatedFiles := []string{}

	// Generate zz_generated.spec.go
	specContent, err := GenerateSpecFile(schema, config, checksum)
	if err != nil {
		return nil, fmt.Errorf("generating spec file: %w", err)
	}
	if err := os.WriteFile(specFilePath, specContent, 0o644); err != nil {
		return nil, fmt.Errorf("writing spec file: %w", err)
	}
	generatedFiles = append(generatedFiles, GeneratedSpecFile)
	log.Printf("forge-dev: generated %s", specFilePath)

	// Generate zz_generated.validate.go
	validateFilePath := filepath.Join(srcDir, GeneratedValidateFile)
	validateContent, err := GenerateValidateFile(schema, config, checksum)
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
	mcpContent, err := GenerateMCPFile(config, checksum)
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
	mainContent, err := GenerateMainFile(config, checksum)
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

	// Ensure docs/ directory exists for schema.md and list.yaml
	docsDir := filepath.Join(srcDir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating docs directory: %w", err)
	}

	// Generate docs/schema.md
	schemaMDPath := filepath.Join(docsDir, "schema.md")
	schemaMDContent, err := GenerateSchemaMD(schema, config)
	if err != nil {
		return nil, fmt.Errorf("generating schema.md: %w", err)
	}
	if err := os.WriteFile(schemaMDPath, schemaMDContent, 0o644); err != nil {
		return nil, fmt.Errorf("writing schema.md: %w", err)
	}
	generatedFiles = append(generatedFiles, "docs/schema.md")
	log.Printf("forge-dev: generated %s", schemaMDPath)

	// Generate docs/list.yaml
	listYAMLPath := filepath.Join(docsDir, "list.yaml")
	listYAMLContent, err := GenerateListYAML(config, checksum)
	if err != nil {
		return nil, fmt.Errorf("generating list.yaml: %w", err)
	}
	if err := os.WriteFile(listYAMLPath, listYAMLContent, 0o644); err != nil {
		return nil, fmt.Errorf("writing list.yaml: %w", err)
	}
	generatedFiles = append(generatedFiles, "docs/list.yaml")
	log.Printf("forge-dev: generated %s", listYAMLPath)

	log.Printf("forge-dev: successfully generated %d files for %s", len(generatedFiles), config.Name)

	// Step 7: Return Artifact with list of generated files
	// Note: The Artifact.Location contains the directory where files were generated
	return &forge.Artifact{
		Name:      config.Name,
		Type:      "generated",
		Location:  srcDir,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   checksum,
	}, nil
}
