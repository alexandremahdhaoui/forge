# Migrating Legacy Engines to forge-dev

**Replace hand-written boilerplate with generated code.**

> "I didn't realize how much of my engine was just plumbing until I migrated. The spec parsing, MCP registration, validation - all gone. Now I maintain 200 lines of actual logic instead of 600 lines of infrastructure."

## What does forge-dev generate vs what do I write?

```
forge-dev generates (DELETE from legacy):     You write (KEEP/CREATE):
├── zz_generated.main.go    (entry point)     ├── forge-dev.yaml      (metadata)
├── zz_generated.mcp.go     (MCP server)      ├── spec.openapi.yaml   (config schema)
├── zz_generated.spec.go    (Spec parsing)    └── <logic>.go          (business logic)
├── zz_generated.validate.go (validation)
└── zz_generated.docs.go    (docs embedding)
```

**The key insight:** forge-dev generates everything except your actual business logic. Your job is to:
1. Define what configuration your engine accepts (spec.openapi.yaml)
2. Implement what your engine does with that configuration (<logic>.go)

## What is boilerplate vs business logic?

**Boilerplate (forge-dev generates this):**
- `main()` function with CLI bootstrap
- MCP server setup and tool registration
- Spec struct definition and `FromMap()` parsing
- Input validation (required fields, enum values)
- Error wrapping for MCP responses

**Business logic (you write this):**
- The actual work your engine does (building, testing, creating resources)
- Domain-specific validation (e.g., "this path must exist")
- External tool invocations (go build, kubectl, helm)
- Result/artifact creation

## What function signature does each engine type need?

forge-dev generates wrapper code that calls YOUR function with a typed `*Spec`:

| Engine Type | Function | Signature |
|-------------|----------|-----------|
| `builder` | `Build` | `func(ctx, mcptypes.BuildInput, *Spec) (*forge.Artifact, error)` |
| `test-runner` | `Run` | `func(ctx, mcptypes.RunInput, *Spec) (*forge.TestReport, error)` |
| `testenv-subengine` | `Create` | `func(ctx, engineframework.CreateInput, *Spec) (*engineframework.TestEnvArtifact, error)` |
| `testenv-subengine` | `Delete` | `func(ctx, engineframework.DeleteInput, *Spec) error` |
| `dependency-detector` | (custom) | Implement MCP tool manually |

## How do I identify boilerplate in my legacy code?

Look for these patterns in your existing `main.go`:

```go
// BOILERPLATE - DELETE THIS:
func main() {
    // CLI flag parsing
    // MCP mode detection
    // enginecli.Bootstrap() or similar
}

// BOILERPLATE - DELETE THIS:
func runMCPServer() error {
    server := mcpserver.New(...)
    // Tool registration
    // Handler setup
    server.Run(...)
}

// BOILERPLATE - DELETE THIS:
type Spec struct { ... }
func parseSpec(m map[string]any) (*Spec, error) { ... }
func (s *Spec) Validate() error { ... }

// BUSINESS LOGIC - KEEP THIS:
func doBuild(input ..., spec *Spec) (*forge.Artifact, error) {
    // Actual build commands
    // exec.Command("go", "build", ...)
    // File operations
    // Artifact creation
}
```

## Step-by-step migration

### Step 1: Create forge-dev.yaml

Create `cmd/<your-engine>/forge-dev.yaml`:

```yaml
name: your-engine
type: builder          # or: test-runner, testenv-subengine, dependency-detector
version: 0.15.0
description: What your engine does
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
```

### Step 2: Create spec.openapi.yaml from your existing Spec

Find your Spec struct in your legacy code and convert it to OpenAPI:

**Legacy Go struct:**
```go
type Spec struct {
    OutputDir string            `json:"outputDir"`
    Verbose   bool              `json:"verbose,omitempty"`
    Tags      []string          `json:"tags,omitempty"`
    Env       map[string]string `json:"env,omitempty"`
}
```

**Becomes spec.openapi.yaml:**
```yaml
openapi: 3.0.3
info:
  title: your-engine Spec Schema
  version: 0.15.0
components:
  schemas:
    Spec:
      type: object
      properties:
        outputDir:
          type: string
          description: Output directory
        verbose:
          type: boolean
          description: Enable verbose output
        tags:
          type: array
          items:
            type: string
          description: Build tags
        env:
          type: object
          additionalProperties:
            type: string
          description: Environment variables
      required:
        - outputDir  # Only if truly required
```

**Type mapping:**

| Go | OpenAPI |
|----|---------|
| `string` | `type: string` |
| `bool` | `type: boolean` |
| `int` | `type: integer` |
| `float64` | `type: number` |
| `[]string` | `type: array` + `items: {type: string}` |
| `map[string]string` | `type: object` + `additionalProperties: {type: string}` |

### Step 3: Extract business logic to a separate file

**For builder engines**, create `build.go`:

```go
package main

import (
    "context"
    "github.com/alexandremahdhaoui/forge/pkg/forge"
    "github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// Build is called by generated code with parsed and validated Spec
func Build(ctx context.Context, input mcptypes.BuildInput, spec *Spec) (*forge.Artifact, error) {
    // Your actual build logic here
    // spec.OutputDir, spec.Verbose, spec.Tags are typed fields

    // Return artifact on success
    return &forge.Artifact{
        Name:    input.Name,
        Type:    "binary",
        Path:    outputPath,
        Version: version,
    }, nil
}
```

**For test-runner engines**, create `run.go`:

```go
package main

import (
    "context"
    "github.com/alexandremahdhaoui/forge/pkg/forge"
    "github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// Run is called by generated code with parsed and validated Spec
func Run(ctx context.Context, input mcptypes.RunInput, spec *Spec) (*forge.TestReport, error) {
    // Your test execution logic here
    // input.Stage, input.Name, input.TmpDir are available
    // spec fields are typed

    return &forge.TestReport{
        Name:   input.Name,
        Stage:  input.Stage,
        Status: "passed",
        Total:  testCount,
        Passed: passedCount,
    }, nil
}
```

**For testenv-subengine engines**, create `create.go` and optionally `delete.go`:

```go
package main

import (
    "context"
    "github.com/alexandremahdhaoui/forge/pkg/engineframework"
)

// Create provisions test environment resources
func Create(ctx context.Context, input engineframework.CreateInput, spec *Spec) (*engineframework.TestEnvArtifact, error) {
    // input.TestID, input.Stage, input.TmpDir are available
    // Provision your resources (cluster, registry, etc.)

    return &engineframework.TestEnvArtifact{
        TestID:   input.TestID,
        Files:    map[string]string{"kubeconfig": "kubeconfig"},
        Metadata: map[string]string{"cluster": clusterName},
        Env:      map[string]string{"KUBECONFIG": kubeconfigPath},
    }, nil
}

// Delete cleans up test environment resources
func Delete(ctx context.Context, input engineframework.DeleteInput, spec *Spec) error {
    // input.TestID, input.Metadata are available
    // Note: spec may be nil - use Metadata for cleanup info
    clusterName := input.Metadata["cluster"]
    // Clean up resources
    return nil
}
```

### Step 4: Delete boilerplate files

Delete these files - forge-dev will regenerate them:

- `main.go` → replaced by `zz_generated.main.go`
- Any `spec.go` or `types.go` with Spec definition → replaced by `zz_generated.spec.go`
- Any `mcp.go`, `server.go`, or `handler.go` → replaced by `zz_generated.mcp.go`
- Any `validate.go` → replaced by `zz_generated.validate.go`

**Keep** your business logic files (build.go, run.go, create.go, etc.) and any helper files they depend on.

### Step 5: Update forge.yaml

Add build targets for code generation:

```yaml
build:
  - name: gen-your-engine
    src: ./cmd/your-engine
    engine: go://forge-dev

  - name: your-engine
    src: ./cmd/your-engine
    dest: ./build/bin
    engine: go://go-build
    depends: [gen-your-engine]
```

### Step 6: Generate and test

```bash
forge build your-engine    # Generates code, then builds
go test ./cmd/your-engine/...
```

## What if my business logic was in main.go?

If your legacy engine has everything in `main.go`, you need to extract the business logic:

1. Create a new file (e.g., `build.go`)
2. Move the core logic function to that file
3. Update the function signature to match what forge-dev expects
4. Delete the old `main.go`

**Before (everything in main.go):**
```go
func main() {
    // 50 lines of CLI setup
    // 30 lines of MCP registration
    // Business logic mixed in
}

func handleBuild(...) {
    // Spec parsing
    spec := parseSpec(input.Spec)
    // Actual build work (KEEP THIS PART)
    cmd := exec.Command("go", "build", ...)
    // Artifact creation
}
```

**After (separated):**
```go
// build.go - ONLY business logic
func Build(ctx context.Context, input mcptypes.BuildInput, spec *Spec) (*forge.Artifact, error) {
    // Just the actual build work
    cmd := exec.Command("go", "build", ...)
    return artifact, nil
}
```

## What if I have no Spec?

If your engine doesn't need configuration, use an empty Spec:

```yaml
# spec.openapi.yaml
openapi: 3.0.3
info:
  title: my-engine Spec Schema
  version: 0.15.0
components:
  schemas:
    Spec:
      type: object
      description: No configuration required
```

Your function still receives `*Spec`, but it will be empty.

## What's next?

- [forge-dev Documentation](./forge-dev.md) - Full reference
- [Creating Build Engines](./creating-build-engine.md) - Engine patterns
- [Creating Test Runners](./creating-test-runner.md) - Test runner patterns
- [Creating Testenv Subengines](./creating-testenv-subengine.md) - Testenv patterns
