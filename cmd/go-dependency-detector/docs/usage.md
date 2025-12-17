# go-dependency-detector

**Detect Go code dependencies to enable lazy rebuild optimization.**

> "I was rebuilding my Go binaries every time, even when nothing changed. Now forge only rebuilds when my actual dependencies change."

## What problem does go-dependency-detector solve?

Building Go binaries is fast, but unnecessary rebuilds waste time. This detector analyzes Go source code to find all dependencies (local files and external packages) for a given function, enabling forge to skip builds when nothing has changed.

## How do I use go-dependency-detector?

You don't invoke it directly. Forge calls it automatically when configured in your build spec:

```yaml
build:
  - name: myapp
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
    spec:
      dependsOn:
        - engine: go://go-dependency-detector
          spec:
            filePath: ./cmd/myapp/main.go
            funcName: main
```

## What does it detect?

### File Dependencies

Local Go source files imported by the function:
- Includes transitive dependencies (A imports B, B imports C)
- Contains absolute file path and modification timestamp

### External Package Dependencies

Third-party packages from go.mod:
- Package identifier (e.g., `github.com/spf13/cobra`)
- Semantic version (e.g., `v1.8.0`)

## How does lazy rebuild work?

1. On first build, dependencies are detected and stored with the artifact
2. On subsequent builds, timestamps are compared
3. If no dependencies changed, build is skipped
4. If any dependency changed, rebuild is triggered

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../MCP.md) - MCP tool documentation
- [go-build usage](../../go-build/docs/usage.md) - Build engine documentation
