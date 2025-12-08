# Go Dependency Detector Configuration Schema

## Overview

This document describes the configuration options for `go-dependency-detector` in `forge.yaml`. The go-dependency-detector engine analyzes Go code to detect all dependencies for lazy rebuild optimization.

## Basic Configuration

```yaml
build:
  - name: myapp
    src: ./cmd/myapp
    engine: go://go-build
    spec:
      dependsOn:
        - engine: go://go-dependency-detector
          spec:
            filePath: ./cmd/myapp/main.go
            funcName: main
```

## Configuration Options

### Required Fields (in spec)

| Field | Type | Description |
|-------|------|-------------|
| `filePath` | string | Path to the Go source file containing the function. |
| `funcName` | string | Name of the function to analyze (typically `main`). |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `spec` | object | `{}` | Additional engine-specific configuration. |

## Examples

### Minimal Configuration

```yaml
spec:
  dependsOn:
    - engine: go://go-dependency-detector
      spec:
        filePath: ./cmd/myapp/main.go
        funcName: main
```

### Full Configuration with go-build

```yaml
build:
  - name: myapp
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
    spec:
      dependsOn:
        - engine: go://go-dependency-detector
          spec:
            filePath: ./cmd/myapp/main.go
            funcName: main
```

### Multiple Dependencies

```yaml
build:
  - name: myapp
    src: ./cmd/myapp
    engine: go://go-build
    spec:
      dependsOn:
        - engine: go://go-dependency-detector
          spec:
            filePath: ./cmd/myapp/main.go
            funcName: main
        - engine: go://go-dependency-detector
          spec:
            filePath: ./pkg/config/config.go
            funcName: Load
```

### Container Build with Dependencies

```yaml
build:
  - name: myapp-container
    src: ./Containerfile
    engine: go://container-build
    spec:
      dependsOn:
        - engine: go://go-dependency-detector
          spec:
            filePath: ./cmd/myapp/main.go
            funcName: main
```

## Output Format

The detector returns dependencies in this format:

### File Dependencies

```json
{
  "type": "file",
  "filePath": "/absolute/path/to/file.go",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### External Package Dependencies

```json
{
  "type": "externalPackage",
  "externalPackage": "github.com/spf13/cobra",
  "semver": "v1.8.0"
}
```

## Dependency Resolution

The detector resolves dependencies by:

1. Parsing the specified file's AST
2. Finding the specified function
3. Analyzing all imports used by the function
4. Recursively following local package imports
5. Extracting external package versions from go.mod
6. Handling replace directives

## Supported Patterns

- Direct function dependencies
- Transitive dependencies through local packages
- External package version tracking
- Replace directive handling
- Pseudo-version support

## Limitations

- Only analyzes static dependencies (runtime reflection not supported)
- Build tags not fully supported
- CGO dependencies not tracked
- Test files excluded from analysis

## See Also

- [Go Dependency Detector Usage Guide](usage.md)
- [Forge Build Documentation](../../docs/forge-usage.md#building-artifacts)
