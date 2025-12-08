# Go Dependency Detector Usage Guide

## Purpose

`go-dependency-detector` is a forge engine for detecting Go code dependencies to enable lazy rebuild optimization. It analyzes Go source code to find all dependencies (local files and external packages) for a given function.

## Invocation

### CLI Mode

Run directly as a standalone command:

```bash
go-dependency-detector
```

Note: CLI mode is not yet fully implemented. Use MCP mode instead.

### MCP Mode

Run as an MCP server:

```bash
go-dependency-detector --mcp
```

Forge invokes this automatically when configured as a dependency detector.

## Available MCP Tools

### `detectDependencies`

Detect all dependencies for a specific Go function.

**Input Schema:**
```json
{
  "filePath": "string (required)",
  "funcName": "string (required)",
  "spec": {}
}
```

**Output:**
```json
{
  "dependencies": [
    {
      "type": "file",
      "filePath": "string",
      "timestamp": "string"
    },
    {
      "type": "externalPackage",
      "externalPackage": "string",
      "semver": "string"
    }
  ]
}
```

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "detectDependencies",
    "arguments": {
      "filePath": "./cmd/myapp/main.go",
      "funcName": "main"
    }
  }
}
```

### `docs-list`

List all available documentation for go-dependency-detector.

### `docs-get`

Get a specific documentation by name.

**Input Schema:**
```json
{
  "name": "string (required)"
}
```

### `docs-validate`

Validate documentation completeness.

## Common Use Cases

### Lazy Rebuild with go-build

Enable dependency-based rebuild decisions:

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

### Container Build Dependencies

Track Go dependencies for container builds:

```yaml
build:
  - name: my-container
    src: ./Containerfile
    engine: go://container-build
    spec:
      dependsOn:
        - engine: go://go-dependency-detector
          spec:
            filePath: ./cmd/myapp/main.go
            funcName: main
```

### Multiple Entrypoints

Detect dependencies for multiple binaries:

```yaml
build:
  - name: cli
    src: ./cmd/cli
    engine: go://go-build
    spec:
      dependsOn:
        - engine: go://go-dependency-detector
          spec:
            filePath: ./cmd/cli/main.go
            funcName: main

  - name: server
    src: ./cmd/server
    engine: go://go-build
    spec:
      dependsOn:
        - engine: go://go-dependency-detector
          spec:
            filePath: ./cmd/server/main.go
            funcName: main
```

## Dependency Types

### File Dependencies (`type: "file"`)

Local Go source files imported by the function:
- Includes transitive dependencies (A imports B, B imports C)
- Contains absolute file path
- Contains file modification timestamp

### External Package Dependencies (`type: "externalPackage"`)

Third-party packages from go.mod:
- Package identifier (e.g., `github.com/spf13/cobra`)
- Semantic version (e.g., `v1.8.0`)
- Supports pseudo-versions

## Implementation Details

- Parses Go AST to find function dependencies
- Recursively follows local package imports (transitive dependencies)
- Extracts versions for external packages from go.mod
- Handles replace directives in go.mod
- Prevents infinite loops on circular dependencies
- Returns absolute file paths with timestamps
- Skips standard library imports

## How Lazy Rebuild Works

1. On first build, dependencies are detected and stored with artifact
2. On subsequent builds, timestamps are compared
3. If no dependencies changed, build is skipped
4. If any dependency changed, rebuild is triggered

## See Also

- [Go Dependency Detector Configuration Schema](schema.md)
- [go-build MCP Server](../../go-build/docs/usage.md)
