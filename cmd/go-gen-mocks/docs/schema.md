# Go Gen Mocks Configuration Schema

## Overview

This document describes the configuration options for `go-gen-mocks` in `forge.yaml`. The go-gen-mocks engine generates mock implementations of Go interfaces using mockery.

## Basic Configuration

```yaml
build:
  - name: go-gen-mocks
    engine: go://go-gen-mocks
```

## Configuration Options

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | The name of the mock generation task. |
| `engine` | string | Must be `go://go-gen-mocks` to use this generator. |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `rootDir` | string | `.` | Root directory for mock generation. |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MOCKERY_VERSION` | `v3.5.5` | Version of mockery to use |
| `MOCKS_DIR` | `./internal/util/mocks` | Directory to clean and generate mocks |

## Examples

### Minimal Configuration

```yaml
build:
  - name: go-gen-mocks
    engine: go://go-gen-mocks
```

### Full Configuration

```yaml
build:
  - name: go-gen-mocks
    engine: go://go-gen-mocks
```

With environment variables:

```bash
MOCKERY_VERSION=v3.6.0 MOCKS_DIR=./test/mocks forge build
```

### In Build Pipeline

```yaml
build:
  # Generate mocks first
  - name: go-gen-mocks
    engine: go://go-gen-mocks

  # Format generated code
  - name: format-mocks
    src: ./internal/util/mocks
    engine: go://go-format

  # Build application
  - name: myapp
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
```

## Mockery Configuration

The engine uses mockery's configuration file (`.mockery.yaml` or `mockery.yaml`):

### Basic Mockery Config

```yaml
with-expecter: true
dir: "./internal/util/mocks"
packages:
  github.com/myorg/myproject/pkg/interfaces:
    interfaces:
      MyInterface:
```

### Advanced Mockery Config

```yaml
with-expecter: true
dir: "./internal/util/mocks"
outpkg: "mocks"
mockname: "Mock{{.InterfaceName}}"
packages:
  github.com/myorg/myproject/pkg/interfaces:
    config:
      outpkg: "interfaces_mocks"
    interfaces:
      Reader:
      Writer:
  github.com/myorg/myproject/internal/service:
    interfaces:
      UserService:
      OrderService:
```

## Generated Artifacts

Each successful mock generation creates an artifact entry:

```yaml
artifacts:
  - name: go-gen-mocks
    type: generated
    location: "./internal/util/mocks"
    timestamp: "2024-01-15T10:30:00Z"
    dependencies:
      - type: file
        filePath: ./pkg/interfaces/interface.go
        timestamp: "2024-01-15T10:00:00Z"
    dependencyDetectorEngine: "go://go-gen-mocks-dep-detector"
```

## Behavior

### Pre-Generation

1. Cleans existing mocks directory (removes all files)
2. Reads mockery configuration

### Generation

1. Runs `go run github.com/vektra/mockery/v3@{version}`
2. Mockery discovers interfaces from configuration
3. Generates mock implementations

### Post-Generation

1. Detects dependencies via go-gen-mocks-dep-detector
2. Returns artifact with dependency information
3. Enables lazy rebuild on subsequent runs

## Default Behavior

When environment variables are not set:

- Uses mockery v3.5.5
- Generates mocks to `./internal/util/mocks`
- Cleans target directory before generation
- Tracks dependencies for lazy rebuild

## See Also

- [Go Gen Mocks Usage Guide](usage.md)
- [Mockery Documentation](https://vektra.github.io/mockery/)
- [Forge Build Documentation](../../docs/forge-usage.md#building-artifacts)
