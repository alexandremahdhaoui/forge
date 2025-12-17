# Creating Build Engines

**Implement custom build logic for forge.**

> "I needed to integrate our custom code generator that wasn't covered by generic-builder. I defined the schema with forge-dev, implemented one function, and it just worked."

## What problem does this solve?

Build engines transform source code into artifacts (binaries, containers, generated files). When generic-builder is too limited, you need a custom engine with typed configuration and rich error handling.

## What is a build engine?

A build engine is an MCP server that:
- Receives build specifications from forge
- Executes build operations (compile, generate, format)
- Returns structured `Artifact` results with metadata

## What MCP tools must a build engine provide?

| Tool | Required | Description |
|------|----------|-------------|
| `build` | Yes | Build a single artifact from `BuildInput` |
| `buildBatch` | Recommended | Build multiple artifacts in one call |
| `config-validate` | Yes | Validate forge.yaml configuration |

**BuildInput:**
```go
type BuildInput struct {
    Name   string         `json:"name"`   // Artifact name
    Src    string         `json:"src"`    // Source path
    Dest   string         `json:"dest"`   // Output path
    Spec   map[string]any `json:"spec"`   // Engine-specific config
}
```

## What is the Artifact type?

```go
type Artifact struct {
    Name      string `json:"name"`      // Artifact name
    Type      string `json:"type"`      // "binary", "container", etc.
    Location  string `json:"location"`  // Output path
    Timestamp string `json:"timestamp"` // RFC3339 build time
    Version   string `json:"version"`   // Git SHA or version
}
```

## How do I implement a build engine?

Use forge-dev. Create `forge-dev.yaml` and `spec.openapi.yaml` (see [forge-dev.md](./forge-dev.md)), then implement:

```go
func Build(ctx context.Context, input mcptypes.BuildInput, spec *Spec) (*forge.Artifact, error) {
    // spec fields are typed and validated
    return &forge.Artifact{
        Name:      input.Name,
        Type:      "generated",
        Location:  filepath.Join(input.Dest, input.Name),
        Timestamp: time.Now().Format(time.RFC3339),
        Version:   getGitSHA(),
    }, nil
}
```

**Add to forge.yaml:**
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

## When to use generic-builder instead?

Use `generic-builder` when:
- You're wrapping a simple CLI tool
- Exit code is sufficient for error handling

Use a custom engine when:
- You need typed, validated configuration
- You need rich artifact metadata
- You need complex build logic
