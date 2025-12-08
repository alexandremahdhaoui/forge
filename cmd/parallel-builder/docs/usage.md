# Parallel Builder Usage Guide

## Purpose

`parallel-builder` is a forge engine for executing multiple builders in parallel. It orchestrates concurrent execution of sub-builders, combining their results into a single meta-artifact.

## Invocation

### MCP Mode

Run as an MCP server:

```bash
parallel-builder --mcp
```

Forge invokes this automatically when using:

```yaml
engine: go://parallel-builder
```

## Available MCP Tools

### `build`

Execute multiple builders in parallel.

**Input Schema:**
```json
{
  "name": "string (required)",
  "spec": {
    "builders": [
      {
        "name": "string (optional)",
        "engine": "string (required)",
        "spec": {}
      }
    ]
  }
}
```

**Output:**
```json
{
  "name": "string",
  "type": "parallel-build",
  "location": "string",
  "timestamp": "string",
  "version": "string"
}
```

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "build",
    "arguments": {
      "name": "parallel-builds",
      "spec": {
        "builders": [
          {
            "name": "build-linux",
            "engine": "go://go-build",
            "spec": {
              "name": "myapp-linux",
              "src": "./cmd/myapp",
              "env": {"GOOS": "linux", "GOARCH": "amd64"}
            }
          },
          {
            "name": "build-darwin",
            "engine": "go://go-build",
            "spec": {
              "name": "myapp-darwin",
              "src": "./cmd/myapp",
              "env": {"GOOS": "darwin", "GOARCH": "arm64"}
            }
          }
        ]
      }
    }
  }
}
```

### `buildBatch`

Execute multiple parallel build specs in sequence.

### `docs-list`

List all available documentation for parallel-builder.

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

### Cross-Platform Builds

Build for multiple platforms simultaneously:

```yaml
build:
  - name: cross-platform
    engine: go://parallel-builder
    spec:
      builders:
        - name: linux-amd64
          engine: go://go-build
          spec:
            name: myapp-linux-amd64
            src: ./cmd/myapp
            dest: ./build/bin
            env:
              GOOS: linux
              GOARCH: amd64
        - name: darwin-arm64
          engine: go://go-build
          spec:
            name: myapp-darwin-arm64
            src: ./cmd/myapp
            dest: ./build/bin
            env:
              GOOS: darwin
              GOARCH: arm64
        - name: windows-amd64
          engine: go://go-build
          spec:
            name: myapp-windows-amd64.exe
            src: ./cmd/myapp
            dest: ./build/bin
            env:
              GOOS: windows
              GOARCH: amd64
```

### Multiple Binaries

Build multiple binaries in parallel:

```yaml
build:
  - name: all-binaries
    engine: go://parallel-builder
    spec:
      builders:
        - name: cli
          engine: go://go-build
          spec:
            name: mycli
            src: ./cmd/cli
        - name: server
          engine: go://go-build
          spec:
            name: myserver
            src: ./cmd/server
        - name: worker
          engine: go://go-build
          spec:
            name: myworker
            src: ./cmd/worker
```

### Mixed Build Types

Run different build engines in parallel:

```yaml
build:
  - name: all-artifacts
    engine: go://parallel-builder
    spec:
      builders:
        - name: binary
          engine: go://go-build
          spec:
            name: myapp
            src: ./cmd/myapp
        - name: mocks
          engine: go://go-gen-mocks
          spec:
            name: generate-mocks
        - name: format
          engine: go://go-format
          spec:
            name: format-code
            src: .
```

## Error Handling

- Partial failures are reported with error count
- Combined artifact is returned even with failures
- Individual builder errors are collected and reported
- Error message format: `parallel-builder: X/Y builders failed: [error details]`

## Implementation Details

- Uses goroutines for concurrent execution
- WaitGroup ensures all builds complete
- Results collected via buffered channel
- Combined artifact tracks number of sub-artifacts
- Location is "multiple" when more than one artifact

## Performance Considerations

- All builders run concurrently (no dependency ordering)
- CPU and I/O bound by number of parallel builds
- Network builds (container push) may be bandwidth limited
- Consider resource limits when running many parallel builds

## See Also

- [Parallel Builder Configuration Schema](schema.md)
- [go-build MCP Server](../../go-build/docs/usage.md)
