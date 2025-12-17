# go-gen-openapi-dep-detector

**Detect file dependencies for OpenAPI code generation.**

> "My OpenAPI client code was regenerating on every build. Now forge only regenerates it when my spec files actually change."

## What problem does go-gen-openapi-dep-detector solve?

OpenAPI code generation can be slow. This detector tracks OpenAPI specification files as dependencies, enabling lazy rebuild support for `go-gen-openapi`.

## How do I use go-gen-openapi-dep-detector?

You don't invoke it directly. It's called automatically by `go-gen-openapi` after code generation:

1. `forge build <name>` invokes `go-gen-openapi`
2. `go-gen-openapi` generates code using oapi-codegen
3. `go-gen-openapi` extracts spec paths from its configuration
4. `go-gen-openapi` calls this detector with the spec paths
5. Dependencies are stored with the artifact
6. On subsequent builds, forge compares timestamps to decide if rebuild is needed

## What does it detect?

- **OpenAPI spec files** - All YAML/JSON specification files configured in the build
- Verifies each file exists and captures its modification timestamp

## What are the limitations?

External references (`$ref`) are NOT automatically tracked. If your spec uses `$ref` to include external files, use `--force` for rebuilds:

```bash
forge build <name> --force
```

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../MCP.md) - MCP tool documentation
- [go-gen-openapi](../../go-gen-openapi/docs/usage.md) - OpenAPI generator documentation
