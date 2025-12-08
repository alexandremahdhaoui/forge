# Container Build Usage Guide

## Purpose

`container-build` is a forge engine for building container images with multiple backend engines (docker, kaniko, podman). It provides automatic git versioning, artifact tracking, and dependency detection.

## Invocation

### CLI Mode

Run directly as a standalone command:

```bash
CONTAINER_BUILD_ENGINE=docker container-build
```

This reads the `forge.yaml` configuration and builds all container images defined with the `go://container-build` engine.

### MCP Mode

Run as an MCP server:

```bash
container-build --mcp
```

Forge invokes this automatically when using:

```yaml
engine: go://container-build
```

## Available MCP Tools

### `build`

Build a single container image.

**Input Schema:**
```json
{
  "name": "string (required)",
  "src": "string (required)",
  "dest": "string (optional)",
  "engine": "string (optional)",
  "spec": {}
}
```

**Output:**
```json
{
  "name": "string",
  "type": "container",
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
      "name": "my-app",
      "src": "./Containerfile"
    }
  }
}
```

### `buildBatch`

Build multiple container images in sequence.

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

List all available documentation for container-build.

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

### Basic Build with Docker

Build a simple container image using Docker:

```yaml
build:
  - name: my-app
    src: ./Containerfile
    engine: go://container-build
```

Run with:

```bash
CONTAINER_BUILD_ENGINE=docker forge build
```

### Build with Kaniko (Rootless)

Build using Kaniko for rootless, secure builds:

```bash
CONTAINER_BUILD_ENGINE=kaniko forge build
```

### Build with Podman

Build using Podman for rootless builds:

```bash
CONTAINER_BUILD_ENGINE=podman forge build
```

### Build with Custom Arguments

Pass build arguments to the container build:

```bash
BUILD_ARGS="VERSION=1.0.0,COMMIT=abc123" CONTAINER_BUILD_ENGINE=docker forge build
```

### Dependency Detection

Track dependencies for lazy rebuild:

```yaml
build:
  - name: my-app
    src: ./Containerfile
    engine: go://container-build
    spec:
      dependsOn:
        - engine: go://go-dependency-detector
          spec:
            filePath: ./cmd/myapp/main.go
            funcName: main
```

## Build Modes

### docker
Native Docker builds using `docker build`. Fast and requires Docker daemon.

### kaniko
Rootless builds using Kaniko executor (runs in container via docker). Secure, supports layer caching.

### podman
Native Podman builds using `podman build`. Rootless and requires Podman.

## Mode Comparison

| Feature | docker | kaniko | podman |
|---------|--------|--------|--------|
| Requires Daemon | Yes (Docker) | Yes (Docker to run Kaniko) | Yes (Podman) |
| Rootless | No | Yes | Yes |
| Build Speed | Fast | Moderate | Fast |
| Layer Caching | Native | Via cache dir | Native |

## Implementation Details

- Automatically tags with git commit SHA
- Tags both `<name>:<version>` and `<name>:latest`
- Stores artifact metadata in artifact store
- Kaniko mode: exports to tar, loads into container engine
- Docker/Podman modes: native builds (faster)
- Supports dependency detection for lazy rebuild

## See Also

- [Container Build Configuration Schema](schema.md)
- [go-build MCP Server](../../go-build/docs/usage.md)
