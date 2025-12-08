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
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// runMCPServer starts the go-gen-bpf MCP server with stdio transport.
func runMCPServer() error {
	server := mcpserver.New(Name, Version)

	config := engineframework.BuilderConfig{
		Name:      Name,
		Version:   Version,
		BuildFunc: build,
	}

	if err := engineframework.RegisterBuilderTools(server, config); err != nil {
		return err
	}

	if err := enginedocs.RegisterDocsTools(server, *docsConfig); err != nil {
		return err
	}

	return server.RunDefault()
}

// Bpf2goOptions holds configuration options extracted from BuildInput.Spec
type Bpf2goOptions struct {
	// Required
	Ident string // Go identifier for generated types (required)

	// Optional with defaults
	Bpf2goVersion string   // Version of bpf2go tool (default: "latest")
	GoPackage     string   // Go package name (default: basename of dest)
	OutputStem    string   // Filename prefix (default: "zz_generated")
	Tags          []string // Build tags (default: ["linux"])

	// Optional, empty means not used
	Types  []string // Specific types to generate (default: all)
	CFlags []string // C compiler flags (default: none)
	CC     string   // C compiler binary (default: bpf2go default)
}

// extractBpf2goOptions extracts bpf2go options from the BuildInput.Spec map.
// Returns a Bpf2goOptions struct with defaults applied for missing values.
// The goPackage default is set to the dest directory basename if not specified.
func extractBpf2goOptions(spec map[string]any, dest string) (*Bpf2goOptions, error) {
	// Required: ident
	ident, err := engineframework.RequireString(spec, "ident")
	if err != nil {
		return nil, err
	}

	// Optional with defaults
	bpf2goVersion := engineframework.ExtractStringWithDefault(spec, "bpf2goVersion", "latest")

	// goPackage defaults to basename of dest directory
	defaultGoPackage := filepath.Base(dest)
	goPackage := engineframework.ExtractStringWithDefault(spec, "goPackage", defaultGoPackage)

	outputStem := engineframework.ExtractStringWithDefault(spec, "outputStem", "zz_generated")
	tags := engineframework.ExtractStringSliceWithDefault(spec, "tags", []string{"linux"})

	// Optional, empty means not used
	types, _ := engineframework.ExtractStringSlice(spec, "types")
	cflags, _ := engineframework.ExtractStringSlice(spec, "cflags")
	cc := engineframework.ExtractStringWithDefault(spec, "cc", "")

	return &Bpf2goOptions{
		Ident:         ident,
		Bpf2goVersion: bpf2goVersion,
		GoPackage:     goPackage,
		OutputStem:    outputStem,
		Tags:          tags,
		Types:         types,
		CFlags:        cflags,
		CC:            cc,
	}, nil
}

// buildBpf2goArgs constructs the bpf2go command line arguments.
// Arguments order:
//  1. --go-package=<pkg>
//  2. --output-dir=<dest>
//  3. --output-stem=<stem>
//  4. --tags=<t1>,<t2>
//  5. --type=<t1> --type=<t2> (for each type)
//  6. <ident>
//  7. <src>
//  8. -- <cflags...> (if any)
func buildBpf2goArgs(src, dest string, opts *Bpf2goOptions) []string {
	args := make([]string, 0)

	// 1. Go package
	args = append(args, "--go-package", opts.GoPackage)

	// 2. Output directory
	args = append(args, "--output-dir", dest)

	// 3. Output stem
	args = append(args, "--output-stem", opts.OutputStem)

	// 4. Tags (comma-joined)
	if len(opts.Tags) > 0 {
		args = append(args, "--tags", strings.Join(opts.Tags, ","))
	}

	// 5. Types (one --type flag per type)
	for _, t := range opts.Types {
		args = append(args, "--type", t)
	}

	// 6. Ident
	args = append(args, opts.Ident)

	// 7. Source file
	args = append(args, src)

	// 8. C flags after "--" separator
	if len(opts.CFlags) > 0 {
		args = append(args, "--")
		args = append(args, opts.CFlags...)
	}

	return args
}

// build implements the BuilderFunc for generating Go code from BPF C source files
func build(ctx context.Context, input mcptypes.BuildInput) (*forge.Artifact, error) {
	// 1. Log start
	log.Printf("Generating BPF code for: %s", input.Name)

	// 2. Validate required inputs
	if input.Src == "" {
		return nil, fmt.Errorf("src is required")
	}
	if input.Dest == "" {
		return nil, fmt.Errorf("dest is required")
	}

	// 3. Validate source file exists and is a file (not directory)
	srcInfo, err := os.Stat(input.Src)
	if err != nil {
		return nil, fmt.Errorf("source file not found: %s", input.Src)
	}
	if srcInfo.IsDir() {
		return nil, fmt.Errorf("src must be a file, not directory")
	}
	log.Printf("Source file: %s", input.Src)

	// 4. Extract bpf2go options from spec
	opts, err := extractBpf2goOptions(input.Spec, input.Dest)
	if err != nil {
		return nil, err
	}

	// 5. Create destination directory
	if err := os.MkdirAll(input.Dest, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create dest directory: %w", err)
	}

	// 6. Build bpf2go command arguments
	bpf2goArgs := buildBpf2goArgs(input.Src, input.Dest, opts)

	// 7. Execute bpf2go via "go run" (following go-gen-mocks pattern)
	bpf2goTool := fmt.Sprintf("github.com/cilium/ebpf/cmd/bpf2go@%s", opts.Bpf2goVersion)
	args := []string{"run", bpf2goTool}
	args = append(args, bpf2goArgs...)

	log.Printf("Executing: go %s", strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Stdout = os.Stderr // MCP mode: redirect to stderr
	cmd.Stderr = os.Stderr

	// Set CC environment variable if specified (preserve existing environment)
	if opts.CC != "" {
		cmd.Env = append(os.Environ(), "BPF2GO_CC="+opts.CC)
	}

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("bpf2go failed: %w", err)
	}

	// 8. Build dependencies
	deps, err := buildDependencies(input.Src, srcInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to build dependencies: %w", err)
	}

	// 9. Create and return artifact
	artifact := &forge.Artifact{
		Name:                     input.Name,
		Type:                     "bpf",
		Location:                 input.Dest,
		Timestamp:                time.Now().UTC().Format(time.RFC3339),
		Dependencies:             deps,
		DependencyDetectorEngine: "go://go-gen-bpf",
	}

	log.Printf("Successfully generated BPF code for %s", input.Name)

	return artifact, nil
}

// buildDependencies creates ArtifactDependency entries for the source file.
// Returns dependencies with absolute paths and RFC3339 timestamps.
func buildDependencies(srcPath string, srcInfo os.FileInfo) ([]forge.ArtifactDependency, error) {
	// Resolve to absolute path
	absPath, err := filepath.Abs(srcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for %s: %w", srcPath, err)
	}

	// Format timestamp as RFC3339 UTC
	timestamp := srcInfo.ModTime().UTC().Format(time.RFC3339)

	deps := []forge.ArtifactDependency{
		{
			Type:      forge.DependencyTypeFile,
			FilePath:  absPath,
			Timestamp: timestamp,
		},
	}

	return deps, nil
}
