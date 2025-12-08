# OpenAPI Code Generator Usage Guide

## Purpose

`go-gen-openapi` is a forge engine for generating Go client and server code from OpenAPI specifications using oapi-codegen. It provides type-safe API implementations with automatic dependency tracking for lazy rebuild support.

## Invocation

### MCP Mode

Run as an MCP server:

```bash
go-gen-openapi --mcp
```

Forge invokes this automatically when using:

```yaml
engine: go://go-gen-openapi
```

## Available MCP Tools

### `build`

Generate OpenAPI client and server code from a specification file.

**Input Schema:**
```json
{
  "name": "string (required)",
  "engine": "string (required)",
  "spec": {
    "sourceFile": "string",
    "destinationDir": "string",
    "client": {
      "enabled": true,
      "packageName": "string"
    },
    "server": {
      "enabled": true,
      "packageName": "string"
    }
  }
}
```

**Output:**
```json
{
  "name": "string",
  "type": "generated",
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
      "name": "example-api-v1",
      "engine": "go://go-gen-openapi",
      "spec": {
        "sourceFile": "./api/example-api.v1.yaml",
        "destinationDir": "./pkg/generated",
        "client": {
          "enabled": true,
          "packageName": "exampleclient"
        }
      }
    }
  }
}
```

### `buildBatch`

Generate OpenAPI code for multiple specifications in batch.

**Input Schema:**
```json
{
  "specs": [
    {
      "name": "string",
      "engine": "string",
      "spec": { }
    }
  ]
}
```

**Output:**
Array of Artifacts with summary of successes/failures.

### `docs-list`

List all available documentation for go-gen-openapi.

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

### Basic Client Generation

Generate a client from an OpenAPI specification:

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

Run with:

```bash
forge build
```

### Client and Server Generation

Generate both client and server code:

```yaml
build:
  - name: products-api-v1
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/products-api.v1.yaml
      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: productsclient
      server:
        enabled: true
        packageName: productsserver
```

This generates two packages:
- `./pkg/generated/productsclient/zz_generated.oapi-codegen.go`
- `./pkg/generated/productsserver/zz_generated.oapi-codegen.go`

### Multiple API Versions

Each API version requires a separate BuildSpec:

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

  - name: example-api-v2
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/example-api.v2.yaml
      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: exampleclientv2
```

## Implementation Details

- Runs `go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@{version}`
- Generates client and/or server code based on `spec` configuration
- Creates temporary oapi-codegen config files for each package
- Generates code concurrently for client and server (when both enabled)
- Tracks spec files as dependencies for lazy rebuild

## Environment Variables

| Variable | Description |
|----------|-------------|
| `OAPI_CODEGEN_VERSION` | Version of oapi-codegen to use (default: `v2.3.0`) |

## Generated Code

- **Client**: HTTP client with typed methods for API calls
- **Server**: HTTP server interfaces and strict handlers
- **Models**: Go structs for request/response types
- **Embedded Spec**: OpenAPI specification embedded in generated code

Output files follow the pattern:
- `{destinationDir}/{packageName}/zz_generated.oapi-codegen.go`

## See Also

- [OpenAPI Code Generator Configuration Schema](schema.md)
- [go-gen-openapi-dep-detector MCP Server](../../go-gen-openapi-dep-detector/MCP.md)
- [oapi-codegen Documentation](https://github.com/oapi-codegen/oapi-codegen)
