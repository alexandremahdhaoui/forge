# Container Build Configuration Schema

## Overview

This document describes the configuration options for `container-build` in `forge.yaml`. The container-build engine creates container images using docker, kaniko, or podman backends.

## Basic Configuration

```yaml
build:
  - name: my-app
    src: ./Containerfile
    engine: go://container-build
```

## Configuration Options

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | The name of the container image. This becomes the image name. |
| `src` | string | Path to the Containerfile/Dockerfile (e.g., `./Containerfile`). |
| `engine` | string | Must be `go://container-build` to use this builder. |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `dest` | string | `local` | Destination registry or location. |
| `spec` | object | `{}` | Engine-specific configuration options (see below). |

## Spec Options

The `spec` field contains engine-specific configuration:

### `dependsOn` (array of dependency specs)

Configure dependency detection for lazy rebuild optimization.

```yaml
spec:
  dependsOn:
    - engine: go://go-dependency-detector
      spec:
        filePath: ./cmd/myapp/main.go
        funcName: main
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CONTAINER_BUILD_ENGINE` | Yes | - | Build mode: docker, kaniko, or podman |
| `BUILD_ARGS` | No | - | Additional build arguments (comma-separated) |
| `KANIKO_CACHE_DIR` | No | `~/.kaniko-cache` | Cache directory for kaniko mode |

## Examples

### Minimal Configuration

```yaml
build:
  - name: my-app
    src: ./Containerfile
    engine: go://container-build
```

### Full Configuration with Dependencies

```yaml
build:
  - name: my-app
    src: ./Containerfile
    dest: localhost:5000
    engine: go://container-build
    spec:
      dependsOn:
        - engine: go://go-dependency-detector
          spec:
            filePath: ./cmd/myapp/main.go
            funcName: main
```

### Multi-Stage Build

```yaml
build:
  # Build binary first
  - name: my-binary
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build

  # Build container with binary
  - name: my-app
    src: ./Containerfile
    engine: go://container-build
```

### Multiple Containers

```yaml
build:
  - name: frontend
    src: ./docker/Containerfile.frontend
    engine: go://container-build

  - name: backend
    src: ./docker/Containerfile.backend
    engine: go://container-build

  - name: worker
    src: ./docker/Containerfile.worker
    engine: go://container-build
```

## Generated Artifacts

Each successful build creates an artifact entry in the artifact store:

```yaml
artifacts:
  - name: my-app
    type: container
    location: "my-app:abc123def"
    timestamp: "2024-01-15T10:30:00Z"
    version: "abc123def"  # Git commit SHA
    dependencies:
      - type: file
        filePath: ./cmd/myapp/main.go
        timestamp: "2024-01-15T10:00:00Z"
```

## Default Behavior

When no `spec` is provided, container-build uses these defaults:

- Version set to current git HEAD commit SHA
- Tags image with both `<name>:<version>` and `<name>:latest`
- No dependency tracking (always rebuilds)

## See Also

- [Container Build Usage Guide](usage.md)
- [Forge Build Documentation](../../docs/forge-usage.md#building-artifacts)
