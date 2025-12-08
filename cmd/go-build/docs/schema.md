# Go Build Configuration Schema

## Overview

This document describes the configuration options for `go-build` in `forge.yaml`. The go-build engine compiles Go source code into binaries with automatic versioning and artifact tracking.

## Basic Configuration

```yaml
build:
  - name: myapp
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
```

## Configuration Options

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | The name of the binary to build. This becomes the output filename. |
| `src` | string | The source directory containing the Go main package (e.g., `./cmd/myapp`). |
| `engine` | string | Must be `go://go-build` to use this builder. |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `dest` | string | `./build/bin` | The output directory for the compiled binary. |
| `spec` | object | `{}` | Engine-specific configuration options (see below). |

## Spec Options

The `spec` field contains engine-specific configuration:

### `args` (array of strings)

Additional arguments to pass to `go build`. These are inserted before the source path.

```yaml
spec:
  args:
    - "-tags=netgo"
    - "-ldflags=-w -s"
    - "-trimpath"
```

**Common arguments:**
- `-tags=<tags>` - Build tags to include
- `-ldflags=<flags>` - Linker flags
- `-trimpath` - Remove file system paths from binary
- `-race` - Enable race detector
- `-v` - Verbose output

### `env` (map of string to string)

Environment variables to set for the build. These override any existing environment variables.

```yaml
spec:
  env:
    GOOS: "linux"
    GOARCH: "amd64"
    CGO_ENABLED: "0"
```

**Common environment variables:**
- `GOOS` - Target operating system (linux, darwin, windows)
- `GOARCH` - Target architecture (amd64, arm64, arm)
- `CGO_ENABLED` - Enable/disable CGO ("0" or "1")
- `CC` - C compiler for CGO
- `CXX` - C++ compiler for CGO

## Examples

### Minimal Configuration

```yaml
build:
  - name: myapp
    src: ./cmd/myapp
    engine: go://go-build
```

### Full Configuration

```yaml
build:
  - name: myapp
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
    spec:
      args:
        - "-tags=netgo,osusergo"
        - "-ldflags=-w -s -extldflags '-static'"
        - "-trimpath"
      env:
        GOOS: "linux"
        GOARCH: "amd64"
        CGO_ENABLED: "0"
```

### Cross-Compilation Matrix

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

  - name: myapp-linux-arm64
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
    spec:
      env:
        GOOS: "linux"
        GOARCH: "arm64"

  - name: myapp-darwin-amd64
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
    spec:
      env:
        GOOS: "darwin"
        GOARCH: "amd64"

  - name: myapp-darwin-arm64
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
    spec:
      env:
        GOOS: "darwin"
        GOARCH: "arm64"

  - name: myapp-windows-amd64.exe
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
    spec:
      env:
        GOOS: "windows"
        GOARCH: "amd64"
```

### Development vs Production Builds

```yaml
build:
  # Development build with race detector
  - name: myapp-dev
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
    spec:
      args:
        - "-race"
      env:
        CGO_ENABLED: "1"

  # Production build, optimized
  - name: myapp
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
    spec:
      args:
        - "-ldflags=-w -s"
        - "-trimpath"
      env:
        CGO_ENABLED: "0"
```

## Environment Variables

In addition to `spec.env`, go-build respects the following environment variables:

| Variable | Description |
|----------|-------------|
| `GO_BUILD_LDFLAGS` | Linker flags applied to all builds. Combined with `spec.args` ldflags. |

## Generated Artifacts

Each successful build creates an artifact entry in the artifact store:

```yaml
artifacts:
  - name: myapp
    type: binary
    location: ./build/bin/myapp
    timestamp: "2024-01-15T10:30:00Z"
    version: "abc123def"  # Git commit SHA
    dependencies:
      - ./cmd/myapp
      - ./pkg/...
```

## Default Behavior

When no `spec` is provided, go-build uses these defaults:

- `CGO_ENABLED=0` (static binary)
- Output to `./build/bin/{name}`
- Version set to current git HEAD commit SHA
- No additional build tags or ldflags

## See Also

- [Go Build Usage Guide](usage.md)
- [Forge Build Documentation](../../docs/forge-usage.md#building-artifacts)
