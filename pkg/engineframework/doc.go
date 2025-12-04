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

// Package engineframework provides MCP tool registration utilities for forge engines.
//
// This package EXTENDS existing infrastructure:
//   - Engines use internal/cli.Bootstrap for lifecycle management (main, version, --mcp flags)
//   - Engines use engineframework for MCP tool registration (build, run, create, delete tools)
//
// DO NOT use this package to replace cli.Bootstrap. Use it to simplify MCP tool registration.
//
// # Architecture
//
// Forge engines have a two-layer architecture:
//
//  1. Lifecycle Layer (internal/cli.Bootstrap):
//     - Handles main() function
//     - Parses command-line flags (version, help, --mcp)
//     - Manages MCP server lifecycle
//     - Provides version information
//
//  2. MCP Registration Layer (pkg/engineframework):
//     - Registers MCP tools (build, run, create, delete)
//     - Handles batch operations automatically
//     - Provides input validation and result formatting
//     - Eliminates duplicate MCP handler code
//
// # Framework Types
//
// The package provides three specialized frameworks:
//
//  1. Builder Framework (builder.go):
//     - For engines that build artifacts (binaries, containers, generated code)
//     - Registers "build" and "buildBatch" MCP tools
//     - Used by: go-build, container-build, generic-builder, go-gen-openapi, go-gen-mocks, go-format, go-lint
//
//  2. TestRunner Framework (testrunner.go):
//     - For engines that execute tests and generate reports
//     - Registers "run" MCP tool
//     - Used by: go-test, go-lint-tags, generic-test-runner, forge-e2e
//
//  3. TestEnv Subengine Framework (testenvsubengine.go):
//     - For engines that create and delete test environments
//     - Registers "create" and "delete" MCP tools
//     - Used by: testenv-kind, testenv-lcr, testenv-helm-install
//     - NOTE: testenv orchestrator does NOT use this (different pattern)
//
// # Function Type Approach
//
// This framework uses function types (not interface embedding):
//
//	type BuilderFunc func(ctx context.Context, input mcptypes.BuildInput) (*forge.Artifact, error)
//	type TestRunnerFunc func(ctx context.Context, input mcptypes.RunInput) (*forge.TestReport, error)
//	type CreateFunc func(ctx context.Context, input CreateInput) (*TestEnvArtifact, error)
//
// This approach:
//   - Compiles cleanly in Go (no impossible interface constraints)
//   - Matches existing engine handler patterns (standalone functions)
//   - Simplifies implementation (no struct requirements)
//   - Maintains type safety
//
// # Usage Examples
//
// # Builder Framework Example
//
// Using cli.Bootstrap + BuilderFramework together:
//
//	package main
//
//	import (
//	    "context"
//	    "fmt"
//	    "github.com/alexandremahdhaoui/forge/internal/cli"
//	    "github.com/alexandremahdhaoui/forge/internal/mcpserver"
//	    "github.com/alexandremahdhaoui/forge/pkg/engineframework"
//	    "github.com/alexandremahdhaoui/forge/pkg/forge"
//	    "github.com/alexandremahdhaoui/forge/pkg/mcptypes"
//	)
//
//	var versionInfo cli.VersionInfo
//
//	func main() {
//	    // Use cli.Bootstrap for lifecycle (main, version, --mcp)
//	    cli.Bootstrap(runMCPServer, &versionInfo)
//	}
//
//	func runMCPServer() error {
//	    v, _, _ := versionInfo.Get()
//	    server := mcpserver.New("my-builder", v)
//
//	    // Use engineframework for MCP registration
//	    config := engineframework.BuilderConfig{
//	        Name:      "my-builder",
//	        Version:   v,
//	        BuildFunc: buildFunc,
//	    }
//
//	    if err := engineframework.RegisterBuilderTools(server, config); err != nil {
//	        return err
//	    }
//
//	    return server.RunDefault()
//	}
//
//	func buildFunc(ctx context.Context, input mcptypes.BuildInput) (*forge.Artifact, error) {
//	    // Extract spec configuration
//	    outputDir := engineframework.ExtractStringWithDefault(input.Spec, "outputDir", "./build")
//
//	    // Perform build
//	    if err := runBuild(input.Name, outputDir); err != nil {
//	        return nil, fmt.Errorf("build failed: %w", err)
//	    }
//
//	    // Return versioned artifact
//	    return engineframework.CreateVersionedArtifact(input.Name, "binary", outputDir+"/"+input.Name)
//	}
//
// # TestRunner Framework Example
//
// For test runners that execute tests and return reports:
//
//	func runMCPServer() error {
//	    v, _, _ := versionInfo.Get()
//	    server := mcpserver.New("my-test-runner", v)
//
//	    config := engineframework.TestRunnerConfig{
//	        Name:        "my-test-runner",
//	        Version:     v,
//	        RunTestFunc: runTests,
//	    }
//
//	    if err := engineframework.RegisterTestRunnerTools(server, config); err != nil {
//	        return err
//	    }
//
//	    return server.RunDefault()
//	}
//
//	func runTests(ctx context.Context, input mcptypes.RunInput) (*forge.TestReport, error) {
//	    // Extract spec configuration
//	    testPattern := engineframework.ExtractStringWithDefault(input.Spec, "pattern", "./...")
//
//	    // Run tests
//	    output, err := executeTests(input.Stage, testPattern)
//	    if err != nil {
//	        // Execution error - couldn't run tests
//	        return nil, fmt.Errorf("failed to execute tests: %w", err)
//	    }
//
//	    // Parse test results
//	    report := parseTestOutput(output)
//
//	    // CRITICAL: Return report even if tests failed
//	    // Framework will use ErrorResultWithArtifact for failed tests
//	    return report, nil
//	}
//
// # TestEnv Subengine Framework Example
//
// For test environment provisioners (clusters, registries, databases):
//
//	func runMCPServer() error {
//	    v, _, _ := versionInfo.Get()
//	    server := mcpserver.New("my-testenv", v)
//
//	    config := engineframework.TestEnvSubengineConfig{
//	        Name:       "my-testenv",
//	        Version:    v,
//	        CreateFunc: createResource,
//	        DeleteFunc: deleteResource,
//	    }
//
//	    if err := engineframework.RegisterTestEnvSubengineTools(server, config); err != nil {
//	        return err
//	    }
//
//	    return server.RunDefault()
//	}
//
//	func createResource(ctx context.Context, input engineframework.CreateInput) (*engineframework.TestEnvArtifact, error) {
//	    // Extract spec configuration
//	    version := engineframework.ExtractStringWithDefault(input.Spec, "version", "latest")
//
//	    // Create resource
//	    resourceName := fmt.Sprintf("myapp-%s", input.TestID)
//	    if err := provisionResource(resourceName, version); err != nil {
//	        return nil, fmt.Errorf("failed to create resource: %w", err)
//	    }
//
//	    // Return artifact with files, metadata, and managed resources
//	    return &engineframework.TestEnvArtifact{
//	        TestID: input.TestID,
//	        Files: map[string]string{
//	            "my-testenv.config": "config.yaml",
//	        },
//	        Metadata: map[string]string{
//	            "my-testenv.resourceName": resourceName,
//	            "my-testenv.version":      version,
//	        },
//	        ManagedResources: []string{input.TmpDir + "/config.yaml"},
//	    }, nil
//	}
//
//	func deleteResource(ctx context.Context, input engineframework.DeleteInput) error {
//	    // Best-effort cleanup
//	    resourceName := input.Metadata["my-testenv.resourceName"]
//	    if err := cleanupResource(resourceName); err != nil {
//	        log.Printf("Warning: failed to cleanup: %v", err)
//	        return nil // Don't fail on cleanup errors
//	    }
//	    return nil
//	}
//
// # Utilities
//
// The package also provides common utilities:
//
//   - Spec Extraction (spec.go): Extract typed values from map[string]any Spec fields
//   - Git Versioning (version.go): Standardize artifact versioning with git commit hashes
//
// See individual framework files (builder.go, testrunner.go, testenvsubengine.go) for detailed usage.
package engineframework
