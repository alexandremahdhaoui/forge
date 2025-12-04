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
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// runMCPServer starts the go-gen-protobuf MCP server with stdio transport.
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

	return server.RunDefault()
}

// build implements the BuilderFunc for compiling Protocol Buffer files to Go code
func build(ctx context.Context, input mcptypes.BuildInput) (*forge.Artifact, error) {
	// 1. Log start
	log.Printf("Generating protobuf code for: %s", input.Name)

	// 2. Validate required inputs
	if input.Src == "" {
		return nil, fmt.Errorf("src is required")
	}
	if input.Dest == "" {
		return nil, fmt.Errorf("dest is required")
	}

	// 3. Discover proto files
	protoFiles, err := discoverProtoFiles(input.Src)
	if err != nil {
		return nil, fmt.Errorf("failed to discover proto files: %w", err)
	}
	if len(protoFiles) == 0 {
		return nil, fmt.Errorf("no .proto files found in %s", input.Src)
	}
	log.Printf("Found %d proto files", len(protoFiles))

	// 4. Build dependencies list
	deps, err := buildDependencies(input.Src, protoFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to build dependencies: %w", err)
	}

	// 5. Extract protoc options
	opts := extractProtocOptions(input.Spec)

	// 6. Build absolute proto file paths for protoc
	absProtoFiles := make([]string, len(protoFiles))
	for i, pf := range protoFiles {
		absProtoFiles[i] = filepath.Join(input.Src, pf)
	}

	// 7. Create destination directory
	if err := os.MkdirAll(input.Dest, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// 8. Build and execute protoc command
	cmd := buildProtocCommand(input.Src, input.Dest, absProtoFiles, opts)
	cmd.Stdout = os.Stderr // MCP mode: redirect to stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("protoc failed: %w", err)
	}

	// 9. Create artifact (DIRECTLY, not using engineframework.CreateArtifact)
	artifact := &forge.Artifact{
		Name:                     input.Name,
		Type:                     "protobuf",
		Location:                 input.Dest,
		Timestamp:                time.Now().UTC().Format(time.RFC3339),
		Dependencies:             deps,
		DependencyDetectorEngine: "go://go-gen-protobuf",
	}

	log.Printf("Successfully generated protobuf code for %s", input.Name)

	return artifact, nil
}

// discoverProtoFiles recursively discovers all .proto files in the given directory.
// Returns a slice of relative paths (relative to srcDir).
// Skips hidden directories (starting with '.') and the 'vendor' directory.
func discoverProtoFiles(srcDir string) ([]string, error) {
	// Check if directory exists
	info, err := os.Stat(srcDir)
	if err != nil {
		return nil, fmt.Errorf("failed to access directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", srcDir)
	}

	var protoFiles []string

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// Skip vendor directory
		if info.IsDir() && info.Name() == "vendor" {
			return filepath.SkipDir
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only include .proto files
		if strings.HasSuffix(info.Name(), ".proto") {
			// Get relative path
			relPath, err := filepath.Rel(srcDir, path)
			if err != nil {
				return fmt.Errorf("failed to get relative path: %w", err)
			}
			protoFiles = append(protoFiles, relPath)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return protoFiles, nil
}

// buildDependencies creates ArtifactDependency entries for each proto file.
// The protoFiles should be relative paths (relative to srcDir).
// Returns dependencies with absolute paths and RFC3339 timestamps.
func buildDependencies(srcDir string, protoFiles []string) ([]forge.ArtifactDependency, error) {
	deps := make([]forge.ArtifactDependency, 0, len(protoFiles))

	for _, protoFile := range protoFiles {
		// Resolve to absolute path
		absPath, err := filepath.Abs(filepath.Join(srcDir, protoFile))
		if err != nil {
			return nil, fmt.Errorf("failed to resolve absolute path for %s: %w", protoFile, err)
		}

		// Get file modification time
		info, err := os.Stat(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to stat file %s: %w", absPath, err)
		}

		// Format timestamp as RFC3339 UTC
		timestamp := info.ModTime().UTC().Format(time.RFC3339)

		deps = append(deps, forge.ArtifactDependency{
			Type:      forge.DependencyTypeFile,
			FilePath:  absPath,
			Timestamp: timestamp,
		})
	}

	return deps, nil
}

// ProtocOptions holds configuration options extracted from BuildInput.Spec
type ProtocOptions struct {
	GoOpt     string   // --go_opt value (default: "paths=source_relative")
	GoGrpcOpt string   // --go-grpc_opt value (default: "paths=source_relative")
	ProtoPath []string // --proto_path values (optional)
	Plugin    []string // --plugin values (optional)
	ExtraArgs []string // additional raw protoc arguments (optional)
}

// extractProtocOptions extracts protoc options from the BuildInput.Spec map.
// Returns a ProtocOptions struct with defaults applied for missing values.
func extractProtocOptions(spec map[string]interface{}) *ProtocOptions {
	opts := &ProtocOptions{
		GoOpt:     "paths=source_relative",
		GoGrpcOpt: "paths=source_relative",
		ProtoPath: []string{},
		Plugin:    []string{},
		ExtraArgs: []string{},
	}

	if spec == nil {
		return opts
	}

	// Extract go_opt
	if val, ok := spec["go_opt"]; ok {
		if str, ok := val.(string); ok {
			opts.GoOpt = str
		}
	}

	// Extract go-grpc_opt
	if val, ok := spec["go-grpc_opt"]; ok {
		if str, ok := val.(string); ok {
			opts.GoGrpcOpt = str
		}
	}

	// Extract proto_path (can be string or []interface{})
	if val, ok := spec["proto_path"]; ok {
		switch v := val.(type) {
		case string:
			if v != "" {
				opts.ProtoPath = []string{v}
			}
		case []interface{}:
			for _, item := range v {
				if str, ok := item.(string); ok {
					opts.ProtoPath = append(opts.ProtoPath, str)
				}
			}
		case []string:
			opts.ProtoPath = v
		}
	}

	// Extract plugin ([]interface{} or []string)
	if val, ok := spec["plugin"]; ok {
		switch v := val.(type) {
		case []interface{}:
			for _, item := range v {
				if str, ok := item.(string); ok {
					opts.Plugin = append(opts.Plugin, str)
				}
			}
		case []string:
			opts.Plugin = v
		}
	}

	// Extract extra_args ([]interface{} or []string)
	if val, ok := spec["extra_args"]; ok {
		switch v := val.(type) {
		case []interface{}:
			for _, item := range v {
				if str, ok := item.(string); ok {
					opts.ExtraArgs = append(opts.ExtraArgs, str)
				}
			}
		case []string:
			opts.ExtraArgs = v
		}
	}

	return opts
}

// buildProtocCommand constructs the protoc command with all arguments.
// Arguments order:
//  1. --proto_path={srcDir} (FIRST - required for import resolution)
//  2. --go_out={dest}
//  3. --go_opt={opts.GoOpt}
//  4. --go-grpc_out={dest}
//  5. --go-grpc_opt={opts.GoGrpcOpt}
//  6. User proto_paths: --proto_path={path}
//  7. Plugins: --plugin={plugin}
//  8. Extra args
//  9. Proto files (at the end)
func buildProtocCommand(srcDir string, dest string, protoFiles []string, opts *ProtocOptions) *exec.Cmd {
	args := make([]string, 0)

	// 1. Source directory proto_path FIRST (required for import resolution)
	args = append(args, "--proto_path="+srcDir)

	// 2. Go output directory
	args = append(args, "--go_out="+dest)

	// 3. Go options
	args = append(args, "--go_opt="+opts.GoOpt)

	// 4. gRPC Go output directory
	args = append(args, "--go-grpc_out="+dest)

	// 5. gRPC Go options
	args = append(args, "--go-grpc_opt="+opts.GoGrpcOpt)

	// 6. User-specified proto_paths
	for _, protoPath := range opts.ProtoPath {
		args = append(args, "--proto_path="+protoPath)
	}

	// 7. Plugins
	for _, plugin := range opts.Plugin {
		args = append(args, "--plugin="+plugin)
	}

	// 8. Extra args
	args = append(args, opts.ExtraArgs...)

	// 9. Proto files at the end
	args = append(args, protoFiles...)

	return exec.Command("protoc", args...)
}
