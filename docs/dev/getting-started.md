# Getting Started: Extending Forge

**Create custom engines to integrate any tool into forge.**

> "I needed a custom linter. I created an engine and now it's fully integrated with forge test."

## What is an engine?

An engine is an MCP server that performs a specific task. Forge communicates with engines via stdio-based JSON-RPC.

| Type | Purpose | Example |
|------|---------|---------|
| **builder** | Build artifacts | `go-build`, `container-build` |
| **test-runner** | Run tests, produce reports | `go-test`, `go-lint` |
| **testenv-subengine** | Manage test infrastructure | `testenv-kind`, `testenv-lcr` |
| **dependency-detector** | Detect dependencies for lazy rebuild | `go-dependency-detector` |

## When should I use generic-* vs custom engine?

**Use `generic-builder` / `generic-test-runner`:** Single command, no validation needed, quick integration.

**Create custom engine:** Typed config, complex logic, reusable component, IDE autocompletion.

## How do I create a custom engine?

1. Create `cmd/my-engine/forge-dev.yaml`:
```yaml
name: my-engine
type: builder
version: 0.1.0
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
```

2. Create `cmd/my-engine/spec.openapi.yaml`:
```yaml
openapi: 3.0.3
info:
  title: my-engine Spec
  version: 0.1.0
components:
  schemas:
    Spec:
      type: object
      properties:
        outputDir:
          type: string
      required: [outputDir]
```

3. Generate and implement:
```bash
forge build generate-my-engine  # Then implement build logic in SetupMCPServer callback
```

## What's next?

- [forge-dev](./forge-dev.md) - Code generation from OpenAPI specs
- [Creating Build Engines](./creating-build-engine.md) - Step-by-step guide
