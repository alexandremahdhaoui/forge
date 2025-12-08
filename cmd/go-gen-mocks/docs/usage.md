# Go Gen Mocks Usage Guide

## Purpose

`go-gen-mocks` is a forge engine for generating Go mock implementations using mockery. It provides automated mock generation for unit testing with dependency injection patterns.

## Invocation

### MCP Mode

Run as an MCP server:

```bash
go-gen-mocks --mcp
```

Forge invokes this automatically when using:

```yaml
engine: go://go-gen-mocks
```

## Available MCP Tools

### `build`

Generate Go mocks using mockery.

**Input Schema:**
```json
{
  "name": "string (required)",
  "engine": "string (optional)",
  "rootDir": "string (optional)"
}
```

**Output:**
```json
{
  "name": "string",
  "type": "generated",
  "location": "string",
  "timestamp": "string",
  "dependencies": []
}
```

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "build",
    "arguments": {
      "name": "go-gen-mocks"
    }
  }
}
```

### `buildBatch`

Generate mocks for multiple configurations in sequence.

### `docs-list`

List all available documentation for go-gen-mocks.

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

### Basic Mock Generation

```yaml
build:
  - name: go-gen-mocks
    engine: go://go-gen-mocks
```

Run with:

```bash
forge build
```

### Custom Mocks Directory

```bash
MOCKS_DIR=./test/mocks forge build
```

### Specific Mockery Version

```bash
MOCKERY_VERSION=v3.6.0 forge build
```

### In Build Pipeline

```yaml
build:
  # Generate mocks first
  - name: go-gen-mocks
    engine: go://go-gen-mocks

  # Then build
  - name: myapp
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
```

### With Parallel Builder

```yaml
build:
  - name: code-gen
    engine: go://parallel-builder
    spec:
      builders:
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

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MOCKERY_VERSION` | `v3.5.5` | Version of mockery to use |
| `MOCKS_DIR` | `./internal/util/mocks` | Directory to clean and generate mocks |

## Mockery Configuration

Uses `.mockery.yaml` or `mockery.yaml` in project root:

```yaml
with-expecter: true
dir: "./internal/util/mocks"
packages:
  github.com/myorg/myproject/pkg/interfaces:
    interfaces:
      MyInterface:
```

## Implementation Details

- Cleans existing mocks directory before generating
- Runs `go run github.com/vektra/mockery/v3@{version}`
- Discovers interfaces automatically via mockery configuration
- Supports lazy rebuild via go-gen-mocks-dep-detector
- Returns generated artifact metadata with dependencies

## Lazy Rebuild Support

The engine integrates with `go-gen-mocks-dep-detector` to track:
- Interface definition files
- Mockery configuration
- Generated mock files

On subsequent builds, if no dependencies changed, mock generation is skipped.

## See Also

- [Go Gen Mocks Configuration Schema](schema.md)
- [go-build MCP Server](../../go-build/docs/usage.md)
- [Mockery Documentation](https://vektra.github.io/mockery/)
