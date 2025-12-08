# OpenAPI Code Generator Configuration Schema

## Overview

This document describes the configuration options for `go-gen-openapi` in `forge.yaml`. The go-gen-openapi engine generates Go client and server code from OpenAPI specifications using oapi-codegen.

## Basic Configuration

```yaml
build:
  - name: example-api-v1
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/example-api.v1.yaml
      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: exampleclient
```

## Configuration Options

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Artifact identifier for the artifact store |
| `engine` | string | Must be `go://go-gen-openapi` to use this generator |

### Spec Options

The `spec` field contains engine-specific configuration:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `sourceFile` | string | - | Path to OpenAPI specification file (recommended) |
| `sourceDir` | string | - | Directory containing spec files (alternative to sourceFile) |
| `name` | string | - | API name for templated source path (used with sourceDir) |
| `version` | string | - | API version for templated source path (used with sourceDir) |
| `destinationDir` | string | `./pkg/generated` | Output directory for generated code |
| `client` | object | - | Client generation options |
| `server` | object | - | Server generation options |

### Client Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable client code generation |
| `packageName` | string | - | Go package name (required if enabled) |

### Server Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable server code generation |
| `packageName` | string | - | Go package name (required if enabled) |

## Validation Rules

1. **MUST** provide EITHER `sourceFile` OR all three of (`sourceDir`, `name`, `version`)
2. **IF** `client.enabled=true` **THEN** `client.packageName` is required
3. **IF** `server.enabled=true` **THEN** `server.packageName` is required
4. At least one of `client.enabled` or `server.enabled` must be true

## Examples

### Minimal Configuration (Client Only)

```yaml
build:
  - name: myapi
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/openapi.yaml
      client:
        enabled: true
        packageName: myapiclient
```

### Full Configuration

```yaml
build:
  - name: products-api-v1
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/products.v1.yaml
      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: productsclient
      server:
        enabled: true
        packageName: productsserver
```

### Using Templated Source Path

For backward compatibility with older configurations:

```yaml
build:
  - name: users-api-v1
    engine: go://go-gen-openapi
    spec:
      sourceDir: ./api
      name: users-api
      version: v1
      # Results in: ./api/users-api.v1.yaml
      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: usersclient
```

### Multiple APIs

```yaml
build:
  - name: users-api-v1
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/users.v1.yaml
      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: usersclient

  - name: products-api-v1
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/products.v1.yaml
      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: productsclient
      server:
        enabled: true
        packageName: productsserver
```

## Generated Output

Each build creates an artifact in the artifact store:

```yaml
artifacts:
  - name: example-api-v1
    type: generated
    location: ./pkg/generated
    timestamp: "2024-01-15T10:30:00Z"
    dependencies:
      - type: file
        filePath: /absolute/path/to/api/example-api.v1.yaml
        timestamp: "2024-01-14T09:00:00Z"
    dependencyDetectorEngine: go://go-gen-openapi-dep-detector
```

## Error Cases

### Missing Required Fields

| Error | Cause | Solution |
|-------|-------|----------|
| `name is required` | BuildInput.Name is missing | Provide `name` field in BuildSpec |
| `must provide either 'sourceFile' or all of...` | Invalid source specification | Use EITHER `sourceFile` OR all three of (`sourceDir`, `name`, `version`) |
| `client.packageName is required when client.enabled=true` | Missing package name | Provide `client.packageName` |
| `server.packageName is required when server.enabled=true` | Missing package name | Provide `server.packageName` |
| `at least one of client.enabled or server.enabled must be true` | No generators enabled | Enable at least one generator |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OAPI_CODEGEN_VERSION` | `v2.3.0` | Version of oapi-codegen to use |

## See Also

- [OpenAPI Code Generator Usage Guide](usage.md)
- [oapi-codegen Documentation](https://github.com/oapi-codegen/oapi-codegen)
