# forge-dev

**Generate type-safe engine code from OpenAPI specs.**

> "I was spending hours writing boilerplate for each new engine - parsing config maps, validating fields, setting up MCP handlers. Now I define a schema in YAML and forge-dev generates everything. I just implement the business logic."

## What problem does forge-dev solve?

Every forge engine needs the same boilerplate:
- A `Spec` struct to hold configuration from forge.yaml
- Parsing logic to convert `map[string]any` to typed structs
- Validation to check required fields and enum values
- MCP server setup with tool registration
- Documentation generation

forge-dev generates all of this from an OpenAPI schema. You define your config structure once, and get type-safe code with validation for free.

## How do I create an engine with forge-dev?

Create 3 files in your engine directory (`cmd/my-engine/`):

**1. forge-dev.yaml** - Engine metadata:
```yaml
name: my-engine
type: builder  # or: test-runner, testenv-subengine, dependency-detector
version: 0.15.0
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
```

**2. spec.openapi.yaml** - Configuration schema:
```yaml
openapi: 3.0.3
info:
  title: my-engine Spec Schema
  version: 0.15.0
components:
  schemas:
    Spec:
      type: object
      properties:
        outputDir:
          type: string
          description: Output directory for generated files
        verbose:
          type: boolean
      required:
        - outputDir
```

**3. docs/usage.md** - Documentation for your engine.

Run generation:
```bash
forge build generate-my-engine  # Assuming forge.yaml has the build target
```

## What goes in forge-dev.yaml?

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Engine name (lowercase, hyphens allowed) |
| `type` | Yes | `builder`, `test-runner`, `testenv-subengine`, or `dependency-detector` |
| `version` | Yes | Semantic version (x.y.z) |
| `description` | No | Human-readable description |
| `openapi.specPath` | Yes | Path to OpenAPI spec file |
| `generate.packageName` | Yes | Go package name (usually `main`) |
| `generate.specTypes.enabled` | No | Generate spec types to a separate package |
| `generate.specTypes.outputPath` | When enabled | Path relative to go.mod (e.g., `pkg/api/v1`) |
| `generate.specTypes.packageName` | When enabled | Go package name for spec types (e.g., `v1`) |

## How do I generate spec types in a separate package?

For engines with multi-package architectures, you can generate the `Spec` struct and helper types in a separate importable package:

```yaml
name: my-engine
type: builder
version: 0.15.0
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
  specTypes:
    enabled: true
    outputPath: pkg/api/v1    # Relative to go.mod location
    packageName: v1
```

When enabled:
- `zz_generated.spec.go` is written to `outputPath` with the specified package name
- Other generated files import and use qualified type references (e.g., `v1.Spec`)
- Import path is derived from `go.mod` module name + `outputPath`

This allows other packages in your project to import and use the spec types directly.

## What goes in spec.openapi.yaml?

Define a `Spec` schema under `components.schemas`:

```yaml
openapi: 3.0.3
info:
  title: my-engine
  version: 0.1.0
components:
  schemas:
    Spec:
      type: object
      properties:
        # Your fields here
      required:
        - requiredField
```

**Supported types:**

| OpenAPI | Go Type | Example |
|---------|---------|---------|
| `string` | `string` | `outputDir: {type: string}` |
| `boolean` | `bool` | `verbose: {type: boolean}` |
| `integer` | `int` | `retries: {type: integer}` |
| `number` | `float64` | `timeout: {type: number}` |
| `array` + `items: {type: string}` | `[]string` | tags array |
| `array` + `items: {type: integer}` | `[]int` | ports array |
| `object` + `additionalProperties: {type: string}` | `map[string]string` | env vars |
| `string` + `enum: [a, b, c]` | `string` (validated) | log levels |

**Limitations:** No `$ref`, `oneOf`, `anyOf`, `allOf`, or arrays of objects.

## What files does forge-dev generate?

| File | Description |
|------|-------------|
| `zz_generated.spec.go` | `Spec` struct with `FromMap()` and `ToMap()` |
| `zz_generated.validate.go` | `Validate()` and `ValidateMap()` functions |
| `zz_generated.mcp.go` | `SetupMCPServer()` with typed function wrappers |
| `zz_generated.main.go` | `main()` with CLI bootstrap |
| `zz_generated.docs.go` | Documentation embedding |
| `docs/schema.md` | Generated schema documentation |
| `docs/list.yaml` | Documentation manifest |

All generated files include a checksum header. Regeneration is skipped if sources haven't changed.

## How do I implement my engine's logic?

Create a file (e.g., `build.go`) and implement the typed function:

**For `type: builder`:**
```go
func Build(ctx context.Context, input mcptypes.BuildInput, spec *Spec) (*forge.Artifact, error) {
    // spec.OutputDir, spec.Verbose are typed and validated
    return &forge.Artifact{Name: input.Name, ...}, nil
}
```

**For `type: test-runner`:**
```go
func Run(ctx context.Context, input mcptypes.RunInput, spec *Spec) (*forge.TestReport, error) {
    return &forge.TestReport{Passed: true, ...}, nil
}
```

**For `type: testenv-subengine`:**
```go
func Create(ctx context.Context, input engineframework.CreateInput, spec *Spec) (*engineframework.TestEnvArtifact, error) {
    return &engineframework.TestEnvArtifact{...}, nil
}

func Delete(ctx context.Context, input engineframework.DeleteInput, spec *Spec) error {
    return nil
}
```

**For `type: dependency-detector`:** Implement `detectDependencies` tool manually (detector inputs vary).

The generated `zz_generated.main.go` references your function by name. The `SetupMCPServer()` in `zz_generated.mcp.go` handles parsing, validation, and MCP registration.

## How do I build and test my engine?

Add to your `forge.yaml`:
```yaml
build:
  - name: generate-my-engine
    src: ./cmd/my-engine
    engine: go://forge-dev

  - name: my-engine
    src: ./cmd/my-engine
    dest: ./build/bin
    engine: go://go-build
    depends: [generate-my-engine]
```

Build:
```bash
forge build my-engine          # Generates code first (via depends)
forge test-all                 # Run all tests including your engine
```

The generated code includes validation that runs automatically when your engine is invoked via MCP.
