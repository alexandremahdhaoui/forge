# Engine Framework

The `engineframework` package provides a standardized framework for building MCP (Model Context Protocol) engines in Forge. It eliminates code duplication by abstracting common patterns for builders, test runners, and test environment subengines.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [When to Use Each Framework](#when-to-use-each-framework)
- [Quick Start Guides](#quick-start-guides)
  - [Creating a Builder](#creating-a-builder)
  - [Creating a Test Runner](#creating-a-test-runner)
  - [Creating a TestEnv Subengine](#creating-a-testenv-subengine)
- [Migration Guide](#migration-guide)
- [Detailed Documentation](#detailed-documentation)
- [Troubleshooting](#troubleshooting)

## Overview

The engine framework provides:

- **Builder Framework** (`builder.go`) - For build engines that create artifacts
- **TestRunner Framework** (`testrunner.go`) - For test execution engines
- **TestEnv Subengine Framework** (`testenvsubengine.go`) - For test environment provisioning
- **Spec Utilities** (`spec.go`) - Type-safe extraction from `map[string]any` specs
- **Version Utilities** (`version.go`) - Git versioning for artifacts

**Key Benefits:**

- Automatic MCP tool registration (build, buildBatch, run, create, delete)
- Standardized input validation
- Consistent error handling and response formatting
- Reduced code duplication (typical savings: ~60% fewer lines)
- Type-safe spec extraction with sensible defaults

## Architecture

The framework follows a two-layer architecture:

```
┌──────────────────────────────────────┐
│   internal/cli.Bootstrap             │  ← Engine Lifecycle
│   - CLI flag parsing                 │
│   - --mcp mode detection             │
│   - Version info management          │
│   - main() orchestration             │
└──────────────────────────────────────┘
                  ↓
┌──────────────────────────────────────┐
│   pkg/engineframework                │  ← MCP Tool Registration
│   - RegisterBuilderTools()           │
│   - RegisterTestRunnerTools()        │
│   - RegisterTestEnvSubengineTools()  │
│   - Input validation                 │
│   - Error conversion                 │
│   - Response formatting              │
└──────────────────────────────────────┘
                  ↓
┌──────────────────────────────────────┐
│   Your Engine Implementation         │  ← Business Logic
│   - BuilderFunc / TestRunnerFunc     │
│   - CreateFunc / DeleteFunc          │
│   - Spec extraction                  │
│   - Resource management              │
└──────────────────────────────────────┘
```

**CRITICAL: Use Both Layers**

- **`internal/cli.Bootstrap`** handles engine lifecycle (CLI parsing, --mcp mode)
- **`pkg/engineframework`** handles MCP tool registration and validation
- **Never replace cli.Bootstrap** - the framework extends it, not replaces it

**Typical main.go structure:**

```go
package main

import (
    "github.com/alexandremahdhaoui/forge/internal/cli"
    "github.com/alexandremahdhaoui/forge/pkg/engineframework"
)

func main() {
    cli.Bootstrap(runMCPServer, &versionInfo)
}

func runMCPServer() error {
    v, _, _ := versionInfo.Get()
    server := mcpserver.New("my-engine", v)

    config := engineframework.BuilderConfig{
        Name:      "my-engine",
        Version:   v,
        BuildFunc: myBuildFunc,
    }

    if err := engineframework.RegisterBuilderTools(server, config); err != nil {
        return err
    }

    return server.RunDefault()
}
```

## When to Use Each Framework

### Builder Framework

Use `RegisterBuilderTools()` when your engine:

- Builds artifacts (binaries, containers, generated code)
- Takes a `BuildInput` with Name, Engine, Spec
- Returns an `Artifact` with Name, Type, Location, Version, Timestamp
- Needs both `build` and `buildBatch` MCP tools

**Examples:** go-build, container-build, generic-builder, go-gen-openapi

### TestRunner Framework

Use `RegisterTestRunnerTools()` when your engine:

- Executes tests and collects results
- Takes a `RunInput` with Stage, Name, Spec
- Returns a `TestReport` with Status, TestStats, ErrorMessage
- Needs a `run` MCP tool
- Must distinguish between test failures (report with Status="failed") and execution errors

**Examples:** go-test, generic-test-runner

### TestEnv Subengine Framework

Use `RegisterTestEnvSubengineTools()` when your engine:

- Provisions test environment resources (clusters, registries, databases)
- Takes `CreateInput` with TestID, Stage, TmpDir, Metadata, Spec
- Takes `DeleteInput` with TestID, Metadata
- Returns `TestEnvArtifact` with Files, Metadata, ManagedResources
- Needs both `create` and `delete` MCP tools
- Is called by the testenv orchestrator

**Examples:** testenv-kind, testenv-lcr, testenv-helm-install

## Quick Start Guides

### Creating a Builder

**Step 1: Define your build function**

```go
func myBuildFunc(ctx context.Context, input mcptypes.BuildInput) (*forge.Artifact, error) {
    // Extract spec values
    outputDir := engineframework.ExtractStringWithDefault(input.Spec, "outputDir", "./build")

    // Perform build logic
    if err := runBuildCommand(input.Name, outputDir); err != nil {
        return nil, fmt.Errorf("build failed: %w", err)
    }

    // Return versioned artifact
    return engineframework.CreateVersionedArtifact(
        input.Name,
        "binary",
        filepath.Join(outputDir, input.Name),
    )
}
```

**Step 2: Register with MCP server**

```go
func runMCPServer() error {
    v, _, _ := versionInfo.Get()
    server := mcpserver.New("my-builder", v)

    config := engineframework.BuilderConfig{
        Name:      "my-builder",
        Version:   v,
        BuildFunc: myBuildFunc,
    }

    if err := engineframework.RegisterBuilderTools(server, config); err != nil {
        return err
    }

    return server.RunDefault()
}
```

**Step 3: Keep cli.Bootstrap in main.go**

```go
func main() {
    cli.Bootstrap(runMCPServer, &versionInfo)
}
```

**What you get automatically:**

- `build` tool for single builds
- `buildBatch` tool for batch builds
- Input validation (Name, Engine required)
- Error conversion to MCP responses
- Artifact formatting

### Creating a Test Runner

**Step 1: Define your test function**

```go
func myTestRunnerFunc(ctx context.Context, input mcptypes.RunInput) (*forge.TestReport, error) {
    // Extract spec values
    testPattern := engineframework.ExtractStringWithDefault(input.Spec, "pattern", "./...")

    // Run tests
    output, err := runTestCommand(input.Stage, testPattern)
    if err != nil {
        // Execution error - couldn't run tests
        return nil, fmt.Errorf("failed to execute tests: %w", err)
    }

    // Parse test results
    report := parseTestOutput(output)

    // CRITICAL: Return report even if tests failed
    // Framework will use ErrorResultWithArtifact for failed tests
    return report, nil
}
```

**Step 2: Register with MCP server**

```go
func runMCPServer() error {
    v, _, _ := versionInfo.Get()
    server := mcpserver.New("my-test-runner", v)

    config := engineframework.TestRunnerConfig{
        Name:        "my-test-runner",
        Version:     v,
        RunTestFunc: myTestRunnerFunc,
    }

    if err := engineframework.RegisterTestRunnerTools(server, config); err != nil {
        return err
    }

    return server.RunDefault()
}
```

**Critical distinction:**

- **Test failures are NOT errors** - Return report with `Status="failed"`
- **Execution errors are errors** - Return `nil, error` when you can't run tests

**What you get automatically:**

- `run` tool for test execution
- Input validation (Stage, Name required)
- Report return even on test failure (uses ErrorResultWithArtifact)
- Summary generation from TestStats

### Creating a TestEnv Subengine

#### CreateInput Schema

The `CreateInput` struct provides all necessary context for creating a test environment resource:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `TestID` | string | Yes | Unique identifier for the test environment |
| `Stage` | string | Yes | Test stage name |
| `TmpDir` | string | Yes | Temporary directory for test artifacts |
| `RootDir` | string | No | Project root directory for resolving relative paths |
| `Metadata` | map[string]string | No | Metadata from previous subengines in the chain |
| `Spec` | map[string]any | No | Subengine-specific configuration |
| `Env` | map[string]string | No | Accumulated environment variables |
| `EnvPropagation` | EnvPropagation | No | Environment variable propagation settings |

**RootDir Usage:**
- Used to resolve relative paths to absolute paths based on the project root
- Populated by the testenv orchestrator via `os.Getwd()`
- Should be used with `filepath.Join()` for portable path resolution
- Example pattern for path resolution:
  ```go
  if input.RootDir != "" && !filepath.IsAbs(relativePath) {
      absolutePath = filepath.Join(input.RootDir, relativePath)
  }
  ```
- Always add fail-fast validation after resolution:
  ```go
  if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
      return nil, fmt.Errorf("path not found: %s", resolvedPath)
  }
  ```

**Step 1: Define create and delete functions**

```go
func myCreateFunc(ctx context.Context, input CreateInput) (*TestEnvArtifact, error) {
    // Extract spec values
    clusterVersion := engineframework.ExtractStringWithDefault(input.Spec, "version", "v1.27.0")

    // Create resource
    clusterName := fmt.Sprintf("myapp-%s", input.TestID)
    if err := createCluster(clusterName, clusterVersion); err != nil {
        return nil, fmt.Errorf("failed to create cluster: %w", err)
    }

    // Generate kubeconfig
    kubeconfigPath := filepath.Join(input.TmpDir, "kubeconfig")
    if err := writeKubeconfig(clusterName, kubeconfigPath); err != nil {
        return nil, fmt.Errorf("failed to write kubeconfig: %w", err)
    }

    // Return artifact
    return &TestEnvArtifact{
        TestID: input.TestID,
        Files: map[string]string{
            "my-engine.kubeconfig": "kubeconfig", // Relative to TmpDir
        },
        Metadata: map[string]string{
            "my-engine.clusterName": clusterName,
            "my-engine.version":     clusterVersion,
        },
        ManagedResources: []string{kubeconfigPath},
    }, nil
}

func myDeleteFunc(ctx context.Context, input DeleteInput) error {
    // Best-effort cleanup - don't fail if already gone
    clusterName := input.Metadata["my-engine.clusterName"]
    if clusterName == "" {
        clusterName = fmt.Sprintf("myapp-%s", input.TestID)
    }

    if err := deleteCluster(clusterName); err != nil {
        log.Printf("Warning: failed to delete cluster: %v", err)
        return nil // Don't fail on cleanup errors
    }

    return nil
}
```

**Step 2: Register with MCP server**

```go
func runMCPServer() error {
    v, _, _ := versionInfo.Get()
    server := mcpserver.New("my-testenv", v)

    config := engineframework.TestEnvSubengineConfig{
        Name:       "my-testenv",
        Version:    v,
        CreateFunc: myCreateFunc,
        DeleteFunc: myDeleteFunc,
    }

    if err := engineframework.RegisterTestEnvSubengineTools(server, config); err != nil {
        return err
    }

    return server.RunDefault()
}
```

**Important patterns:**

- Use `input.TmpDir` for file storage
- Return **relative paths** in Files map
- Store metadata for downstream consumers and cleanup
- Delete should be best-effort (don't fail if resource is gone)

**What you get automatically:**

- `create` tool for resource provisioning
- `delete` tool for cleanup
- Input validation (TestID, Stage, TmpDir for create; TestID for delete)
- Artifact serialization to map[string]interface{}

## Migration Guide

### Before: Manual MCP Registration

```go
// Old mcp.go - ~165 lines
func runMCPServer() error {
    server := mcpserver.New("my-builder", "1.0.0")

    mcpserver.RegisterTool(server, &mcp.Tool{
        Name:        "build",
        Description: "Build an artifact",
    }, handleBuildTool)

    mcpserver.RegisterTool(server, &mcp.Tool{
        Name:        "buildBatch",
        Description: "Build multiple artifacts",
    }, handleBuildBatchTool)

    return server.RunDefault()
}

func handleBuildTool(ctx context.Context, req *mcp.CallToolRequest, input BuildInput) (*mcp.CallToolResult, any, error) {
    // 30+ lines of validation, error handling, response formatting
    if input.Name == "" {
        return mcputil.ErrorResult("Build failed: name is required"), nil, nil
    }
    // ... more validation ...

    artifact, err := build(input)
    if err != nil {
        return mcputil.ErrorResult(fmt.Sprintf("Build failed: %v", err)), nil, nil
    }

    result, returnedArtifact := mcputil.SuccessResultWithArtifact("Build succeeded", artifact)
    return result, returnedArtifact, nil
}

func handleBuildBatchTool(ctx context.Context, req *mcp.CallToolRequest, input BatchBuildInput) (*mcp.CallToolResult, any, error) {
    // 40+ lines of batch handling, error collection, response formatting
    // ...
}

func build(input BuildInput) (*Artifact, error) {
    // Actual build logic
}
```

### After: Using Framework

```go
// New mcp.go - ~40 lines
func runMCPServer() error {
    v, _, _ := versionInfo.Get()
    server := mcpserver.New("my-builder", v)

    config := engineframework.BuilderConfig{
        Name:      "my-builder",
        Version:   v,
        BuildFunc: build,
    }

    if err := engineframework.RegisterBuilderTools(server, config); err != nil {
        return err
    }

    return server.RunDefault()
}

func build(ctx context.Context, input mcptypes.BuildInput) (*forge.Artifact, error) {
    // Extract spec values
    outputDir := engineframework.ExtractStringWithDefault(input.Spec, "outputDir", "./build")

    // Actual build logic
    // ...

    return engineframework.CreateVersionedArtifact(name, "binary", path)
}
```

**Migration checklist:**

1. ✅ Keep `cli.Bootstrap()` in main.go (DO NOT CHANGE)
2. ✅ Create BuilderFunc/TestRunnerFunc/CreateFunc+DeleteFunc
3. ✅ Move validation logic to framework (automatic)
4. ✅ Use spec extraction utilities for configuration
5. ✅ Use version utilities for artifacts
6. ✅ Remove manual tool registration code
7. ✅ Remove manual validation code
8. ✅ Remove manual error conversion code
9. ✅ Test with existing integration tests

## Detailed Documentation

### Function Type Approach

The framework uses **function types** instead of interface embedding:

```go
type BuilderFunc func(ctx context.Context, input mcptypes.BuildInput) (*forge.Artifact, error)
```

**Why not interfaces?**

```go
// ❌ Interface approach - more complex
type Builder interface {
    Build(ctx context.Context, input BuildInput) (*Artifact, error)
}

// Requires struct with methods
type MyBuilder struct{}
func (b *MyBuilder) Build(ctx context.Context, input BuildInput) (*Artifact, error) { ... }

// ✅ Function type approach - simpler
func myBuildFunc(ctx context.Context, input BuildInput) (*Artifact, error) { ... }
```

**Benefits of function types:**

- Simpler - just write a function, no struct needed
- More idiomatic Go for handler-style code
- Compiles cleanly without ceremony
- Matches existing MCP handler patterns

### Spec Extraction Utilities

The framework provides type-safe extraction from `map[string]any` specs:

```go
// Extract with type conversion
value, ok := engineframework.ExtractString(spec, "key")
sliceVal, ok := engineframework.ExtractStringSlice(spec, "tags")
mapVal, ok := engineframework.ExtractStringMap(spec, "labels")
boolVal, ok := engineframework.ExtractBool(spec, "enabled")
intVal, ok := engineframework.ExtractInt(spec, "timeout")

// Extract with defaults
value := engineframework.ExtractStringWithDefault(spec, "key", "default")
timeout := engineframework.ExtractIntWithDefault(spec, "timeout", 30)

// Require values (returns error if missing/wrong type)
value, err := engineframework.RequireString(spec, "key")
tags, err := engineframework.RequireStringSlice(spec, "tags")
```

**Handles JSON unmarshal edge cases:**

- `[]any` with string elements → `[]string`
- `map[string]any` with string values → `map[string]string`
- `float64` (JSON number) → `int` (if no decimal part)

### Git Versioning Utilities

```go
// Get current git commit SHA
version, err := engineframework.GetGitVersion()
// Returns: "abc123..." or "unknown" on error

// Create versioned artifact (uses git SHA)
artifact, err := engineframework.CreateVersionedArtifact("my-app", "binary", "./build/bin/my-app")
// artifact.Version = "abc123..."
// artifact.Timestamp = "2024-01-15T10:30:00Z"

// Create artifact without version (for generated code)
artifact := engineframework.CreateArtifact("openapi-client", "generated", "./pkg/generated")
// artifact.Version = ""
// artifact.Timestamp = "2024-01-15T10:30:00Z"

// Create artifact with custom version
artifact := engineframework.CreateCustomArtifact("my-app", "container", "localhost:5000/my-app:v1.2.3", "v1.2.3")
// artifact.Version = "v1.2.3"
// artifact.Timestamp = "2024-01-15T10:30:00Z"
```

**All timestamps are RFC3339 in UTC.**

## Troubleshooting

### Problem: "unknown tool buildBatch" error

**Cause:** MCP tool not registered

**Solution:** Use `RegisterBuilderTools()` instead of manually registering only `build` tool

```go
// ❌ Wrong - missing buildBatch
mcpserver.RegisterTool(server, &mcp.Tool{Name: "build"}, handleBuild)

// ✅ Correct - registers both build and buildBatch
engineframework.RegisterBuilderTools(server, config)
```

### Problem: Tests fail but report is nil

**Cause:** Returning error instead of report for test failures

**Solution:** Return report with Status="failed", only return error for execution failures

```go
// ❌ Wrong - returns error for test failures
if testsFailed {
    return nil, errors.New("tests failed")
}

// ✅ Correct - returns report with Status="failed"
if testsFailed {
    return &forge.TestReport{
        Status: "failed",
        TestStats: forge.TestStats{...},
    }, nil
}
```

### Problem: Spec extraction returns wrong type

**Cause:** JSON unmarshal converts types

**Solution:** Use framework extraction utilities that handle JSON edge cases

```go
// ❌ Wrong - doesn't handle []any with string elements
tags := spec["tags"].([]string) // Panics!

// ✅ Correct - handles JSON unmarshal edge cases
tags, ok := engineframework.ExtractStringSlice(spec, "tags")
```

### Problem: cli.Bootstrap conflicts with framework

**Cause:** Misunderstanding architecture

**Solution:** Use BOTH - cli.Bootstrap for lifecycle, framework for MCP registration

```go
// ✅ Correct - use both layers
func main() {
    cli.Bootstrap(runMCPServer, &versionInfo) // Lifecycle
}

func runMCPServer() error {
    server := mcpserver.New("my-engine", v)
    engineframework.RegisterBuilderTools(server, config) // MCP registration
    return server.RunDefault()
}
```

### Problem: Missing artifact version

**Cause:** Using `CreateArtifact()` for built binaries

**Solution:** Use `CreateVersionedArtifact()` for built artifacts

```go
// ❌ Wrong - no version for built binary
artifact := engineframework.CreateArtifact(name, "binary", path)

// ✅ Correct - includes git commit SHA as version
artifact, err := engineframework.CreateVersionedArtifact(name, "binary", path)
```

## Links

- [GoDoc](https://pkg.go.dev/github.com/alexandremahdhaoui/forge/pkg/engineframework)
- [Source Code](https://github.com/alexandremahdhaoui/forge/tree/main/pkg/engineframework)
- [Migration Plan](.ai/plan/common-framework/tasks.md)
- [Builder Examples](examples_test.go)
