# Go Format Configuration Schema

## Overview

This document describes the configuration options for `go-format` in `forge.yaml`. The go-format engine formats Go source code using gofumpt for consistent code style.

## Basic Configuration

```yaml
build:
  - name: format-code
    src: .
    engine: go://go-format
```

## Configuration Options

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | The name of the format task. |
| `engine` | string | Must be `go://go-format` to use this formatter. |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `src` | string | `.` | The directory to format. |
| `path` | string | - | Alternative to `src` for specifying the path. |

## Examples

### Minimal Configuration

```yaml
build:
  - name: format
    engine: go://go-format
```

### Format Specific Directory

```yaml
build:
  - name: format-pkg
    src: ./pkg
    engine: go://go-format
```

### Full Configuration

```yaml
build:
  - name: format-code
    src: .
    engine: go://go-format
```

### Multiple Format Tasks

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

### In Pre-Build Pipeline

```yaml
build:
  # Format first
  - name: format-code
    src: .
    engine: go://go-format

  # Then build
  - name: myapp
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GOFUMPT_VERSION` | `v0.6.0` | Version of gofumpt to use |

To use a different version:

```bash
GOFUMPT_VERSION=v0.7.0 forge build
```

## Generated Artifacts

Each successful format creates an artifact entry:

```yaml
artifacts:
  - name: formatted-code
    type: formatted
    location: "."
    timestamp: "2024-01-15T10:30:00Z"
```

## Default Behavior

When optional fields are not provided:

- `src` defaults to current directory (`.`)
- Uses gofumpt v0.6.0 unless overridden by environment variable
- Formats all `.go` files recursively
- Writes changes directly to files

## Formatting Rules

Gofumpt applies these rules beyond standard gofmt:

- No empty lines at the start or end of a function body
- No empty lines around a lone statement in a block
- Imports are sorted and grouped properly
- Simplified slice expressions where possible
- Consistent newline handling

## See Also

- [Go Format Usage Guide](usage.md)
- [Forge Build Documentation](../../docs/forge-usage.md#building-artifacts)
