# Go Build Usage Guide

## Purpose

`go-build` is a forge engine for building Go binaries with automatic git versioning and artifact tracking. It provides consistent build flags, version injection via ldflags, and integration with the forge artifact store.

## Invocation

### CLI Mode

Run directly as a standalone command:

```bash
go-build
```

This reads the `forge.yaml` configuration and builds all Go binaries defined with the `go://go-build` engine.

### MCP Mode

Run as an MCP server:

```bash
go-build --mcp
```

Forge invokes this automatically when using:

```yaml
engine: go://go-build
```

## Available MCP Tools

### `build`

Build a single Go binary.

**Input Schema:**
```json
{
  "name": "string (required)",
  "src": "string (required)",
  "dest": "string (optional)",
  "engine": "string (optional)",
  "args": ["string"],
  "env": {"key": "value"}
}
```

**Output:**
```json
{
  "name": "string",
  "type": "binary",
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
      "name": "myapp",
      "src": "./cmd/myapp",
      "dest": "./build/bin"
    }
  }
}
```

### `buildBatch`

Build multiple Go binaries in sequence.

**Input Schema:**
```json
{
  "specs": [
    {
      "name": "string",
      "src": "string",
      "dest": "string"
    }
  ]
}
```

**Output:**
Array of Artifacts with summary of successes/failures.

### `docs-list`

List all available documentation for go-build.

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

### Basic Build

Build a simple Go binary:

```yaml
build:
  - name: myapp
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
```

Run with:

```bash
forge build
```

### Static Binary with Build Tags

Build a fully static binary with netgo tag:

```yaml
build:
  - name: static-binary
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
    spec:
      args:
        - "-tags=netgo"
        - "-ldflags=-w -s"
      env:
        CGO_ENABLED: "0"
```

### Cross-Compilation

Build for multiple platforms:

```yaml
build:
  - name: myapp-linux-amd64
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
    spec:
      env:
        GOOS: "linux"
        GOARCH: "amd64"
        CGO_ENABLED: "0"

  - name: myapp-darwin-arm64
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
    spec:
      env:
        GOOS: "darwin"
        GOARCH: "arm64"
        CGO_ENABLED: "0"
```

### Custom Linker Flags

Inject build-time variables:

```yaml
build:
  - name: myapp-optimized
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
    spec:
      args:
        - "-ldflags=-w -s -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
      env:
        CGO_ENABLED: "0"
```

## Implementation Details

- Runs `go build` with optimized flags
- Injects version via ldflags (git commit SHA)
- Outputs binary to `{dest}/{name}`
- Stores artifact metadata in artifact store
- Uses current git HEAD for versioning
- Supports custom build arguments via `args` field
- Supports custom environment variables via `env` field
- Sets `CGO_ENABLED=0` by default (can be overridden)

## Build Flags

**Default build command:**
```bash
CGO_ENABLED=0 go build -o {dest}/{name} {src}
```

**With custom args and env:**
```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags=netgo -ldflags="-w -s" -o {dest}/{name} {src}
```

**Notes:**
- Custom `args` are inserted before the source path
- Custom `env` variables override defaults
- `CGO_ENABLED=0` is set by default but can be overridden via `env`

## See Also

- [Go Build Configuration Schema](schema.md)
- [container-build MCP Server](../../container-build/docs/usage.md)
