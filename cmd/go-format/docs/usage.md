# Go Format Usage Guide

## Purpose

`go-format` is a forge engine for formatting Go source code using gofumpt. It provides consistent code style enforcement with stricter formatting rules than the standard gofmt.

## Invocation

### MCP Mode

Run as an MCP server:

```bash
go-format --mcp
```

Forge invokes this automatically when using:

```yaml
engine: go://go-format
```

## Available MCP Tools

### `build`

Format Go code in the specified directory.

**Input Schema:**
```json
{
  "name": "string (required)",
  "src": "string (optional)",
  "path": "string (optional)",
  "engine": "string (optional)"
}
```

**Output:**
```json
{
  "name": "string",
  "type": "formatted",
  "location": "string",
  "timestamp": "string"
}
```

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "build",
    "arguments": {
      "name": "format-code",
      "src": "."
    }
  }
}
```

### `buildBatch`

Format multiple directories in sequence.

**Input Schema:**
```json
{
  "specs": [
    {
      "name": "string",
      "src": "string"
    }
  ]
}
```

**Output:**
Array of Artifacts with summary of successes/failures.

### `docs-list`

List all available documentation for go-format.

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

### Format Entire Project

```yaml
build:
  - name: format-code
    src: .
    engine: go://go-format
```

Run with:

```bash
forge build
```

### Format Specific Directory

```yaml
build:
  - name: format-pkg
    src: ./pkg
    engine: go://go-format
```

### Format Multiple Directories

```yaml
build:
  - name: format-cmd
    src: ./cmd
    engine: go://go-format

  - name: format-internal
    src: ./internal
    engine: go://go-format

  - name: format-pkg
    src: ./pkg
    engine: go://go-format
```

### Use with Parallel Builder

```yaml
build:
  - name: format-all
    engine: go://parallel-builder
    spec:
      builders:
        - name: format-cmd
          engine: go://go-format
          spec:
            name: cmd
            src: ./cmd
        - name: format-internal
          engine: go://go-format
          spec:
            name: internal
            src: ./internal
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GOFUMPT_VERSION` | `v0.6.0` | Version of gofumpt to use |

## Implementation Details

- Uses `go run mvdan.cc/gofumpt@{version}` to run gofumpt
- Applies stricter formatting rules than standard gofmt
- Modifies files in-place with `-w` flag
- Formats all `.go` files recursively in the specified directory

## Gofumpt vs Gofmt

Gofumpt is a stricter version of gofmt that applies additional formatting rules:

- No empty lines at the start or end of a function body
- No empty lines around a lone statement in a block
- Imports are sorted and grouped properly
- Simplified slice expressions where possible
- And more...

## See Also

- [Go Format Configuration Schema](schema.md)
- [go-lint MCP Server](../../go-lint/docs/usage.md)
- [Gofumpt Documentation](https://github.com/mvdan/gofumpt)
