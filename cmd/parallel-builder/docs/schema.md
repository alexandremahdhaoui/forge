# Parallel Builder Configuration Schema

## Overview

This document describes the configuration options for `parallel-builder` in `forge.yaml`. The parallel-builder engine executes multiple sub-builders concurrently.

## Basic Configuration

```yaml
build:
  - name: parallel-builds
    engine: go://parallel-builder
    spec:
      builders:
        - engine: go://go-build
          spec:
            name: myapp
            src: ./cmd/myapp
```

## Configuration Options

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | The name of the parallel build task. |
| `engine` | string | Must be `go://parallel-builder` to use this builder. |
| `spec.builders` | array | List of builder configurations to run in parallel. |

### Builder Configuration

Each builder in the `builders` array has these fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | No | Optional name for this builder (used in logs/errors). |
| `engine` | string | Yes | Engine URI (e.g., `go://go-build`, `go://generic-builder`). |
| `spec` | object | Yes | Build specification passed to the sub-builder. |

## Examples

### Minimal Configuration

```yaml
build:
  - name: parallel-build
    engine: go://parallel-builder
    spec:
      builders:
        - engine: go://go-build
          spec:
            name: myapp
            src: ./cmd/myapp
```

### Full Cross-Platform Build

```yaml
build:
  - name: cross-platform-builds
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
              CGO_ENABLED: "0"

        - name: linux-arm64
          engine: go://go-build
          spec:
            name: myapp-linux-arm64
            src: ./cmd/myapp
            dest: ./build/bin
            env:
              GOOS: linux
              GOARCH: arm64
              CGO_ENABLED: "0"

        - name: darwin-amd64
          engine: go://go-build
          spec:
            name: myapp-darwin-amd64
            src: ./cmd/myapp
            dest: ./build/bin
            env:
              GOOS: darwin
              GOARCH: amd64
              CGO_ENABLED: "0"

        - name: darwin-arm64
          engine: go://go-build
          spec:
            name: myapp-darwin-arm64
            src: ./cmd/myapp
            dest: ./build/bin
            env:
              GOOS: darwin
              GOARCH: arm64
              CGO_ENABLED: "0"

        - name: windows-amd64
          engine: go://go-build
          spec:
            name: myapp-windows-amd64.exe
            src: ./cmd/myapp
            dest: ./build/bin
            env:
              GOOS: windows
              GOARCH: amd64
              CGO_ENABLED: "0"
```

### Multiple Services

```yaml
build:
  - name: all-services
    engine: go://parallel-builder
    spec:
      builders:
        - name: api-server
          engine: go://go-build
          spec:
            name: api-server
            src: ./cmd/api
            dest: ./build/bin

        - name: worker
          engine: go://go-build
          spec:
            name: worker
            src: ./cmd/worker
            dest: ./build/bin

        - name: cli
          engine: go://go-build
          spec:
            name: cli
            src: ./cmd/cli
            dest: ./build/bin

        - name: scheduler
          engine: go://go-build
          spec:
            name: scheduler
            src: ./cmd/scheduler
            dest: ./build/bin
```

### Mixed Engines

```yaml
build:
  - name: full-build
    engine: go://parallel-builder
    spec:
      builders:
        - name: generate-mocks
          engine: go://go-gen-mocks
          spec:
            name: mocks

        - name: format-code
          engine: go://go-format
          spec:
            name: format
            src: .

        - name: build-binary
          engine: go://go-build
          spec:
            name: myapp
            src: ./cmd/myapp
```

### Generic Commands in Parallel

```yaml
build:
  - name: parallel-scripts
    engine: go://parallel-builder
    spec:
      builders:
        - name: npm-build
          engine: go://generic-builder
          spec:
            name: frontend
            command: npm
            args: ["run", "build"]
            workDir: ./frontend

        - name: go-generate
          engine: go://generic-builder
          spec:
            name: generate
            command: go
            args: ["generate", "./..."]
```

## Generated Artifacts

Each successful parallel build creates a meta-artifact:

```yaml
artifacts:
  - name: parallel-builds
    type: parallel-build
    location: "multiple"  # or first artifact's location if only one
    timestamp: "2024-01-15T10:30:00Z"
    version: "5-artifacts"  # number of sub-artifacts
```

## Default Behavior

- All builders execute concurrently
- No ordering or dependency handling between builders
- Combined artifact version shows count of sub-artifacts
- Location is "multiple" if more than one sub-artifact

## Error Handling

- All builders run to completion (no early abort)
- Partial failures are reported with counts
- Error message includes all failed builder errors
- Combined artifact returned even with partial failures

## See Also

- [Parallel Builder Usage Guide](usage.md)
- [Forge Build Documentation](../../docs/forge-usage.md#building-artifacts)
