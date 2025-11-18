# go-gen-openapi MCP Server

MCP server for generating OpenAPI client and server code from specifications.

## Purpose

Provides MCP tools for generating Go client and server code from OpenAPI specifications using oapi-codegen, enabling type-safe API implementations.

## Invocation

```bash
go-gen-openapi --mcp
```

Forge invokes this automatically via:
```yaml
engine: go://go-gen-openapi
```

## Available Tools

### `build`

Generate OpenAPI client and server code from a specification file.

**Input Schema:**
```json
{
  "name": "string (required)",        // Build artifact name (e.g., "example-api-v1")
  "engine": "string (required)",      // Engine reference (e.g., "go://go-gen-openapi")
  "spec": {
    // Source file specification (EITHER sourceFile OR sourceDir+name+version)
    "sourceFile": "string",           // Explicit path to OpenAPI spec (RECOMMENDED)

    // OR templated source file:
    "sourceDir": "string",            // Directory containing spec files
    "name": "string",                 // API name for templating
    "version": "string",              // API version for templating

    // Destination
    "destinationDir": "string",       // Output directory (defaults to "./pkg/generated")

    // Client generation
    "client": {
      "enabled": true,                // Enable client generation (defaults to false)
      "packageName": "string"         // Package name (required if enabled=true)
    },

    // Server generation
    "server": {
      "enabled": true,                // Enable server generation (defaults to false)
      "packageName": "string"         // Package name (required if enabled=true)
    }
  }
}
```

**Validation Rules:**
1. MUST provide EITHER `sourceFile` OR all three of (`sourceDir`, `name`, `version`)
2. IF `client.enabled=true` THEN `client.packageName` is required
3. IF `server.enabled=true` THEN `server.packageName` is required
4. At least one of `client.enabled` or `server.enabled` must be true

**Default Values:**
- `destinationDir`: `"./pkg/generated"` (if not specified)
- `client.enabled`: `false` (if not specified)
- `server.enabled`: `false` (if not specified)

**Output Schema:**
```json
{
  "name": "string",                   // From input.Name (e.g., "example-api-v1")
  "type": "generated",                // Fixed type
  "location": "string",               // From spec.destinationDir (e.g., "./pkg/generated")
  "timestamp": "string"               // RFC3339 format (UTC)
}
```

**Note:** The artifact does NOT include a `version` field. Generated code is versioned by the source specification file, not by git commit.

**Example 1: Client generation with sourceFile (RECOMMENDED)**
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

**Example 2: Client and server with templated source**
```json
{
  "method": "tools/call",
  "params": {
    "name": "build",
    "arguments": {
      "name": "products-api-v2",
      "engine": "go://go-gen-openapi",
      "spec": {
        "sourceDir": "./api",
        "name": "products-api",
        "version": "v2",
        "destinationDir": "./pkg/generated",
        "client": {
          "enabled": true,
          "packageName": "productsclient"
        },
        "server": {
          "enabled": true,
          "packageName": "productsserver"
        }
      }
    }
  }
}
```

### `buildBatch`

Generate OpenAPI client and server code for multiple specifications in batch.

**Input Schema:**
```json
{
  "specs": [
    {
      "name": "string (required)",
      "engine": "string (required)",
      "spec": {
        // Same structure as build tool spec field
        // See build tool documentation above for full spec schema
      }
    }
  ]
}
```

**Output Schema:**
```json
{
  "artifacts": [
    {
      "name": "string",
      "type": "generated",
      "location": "string",
      "timestamp": "string"
    }
  ],
  "summary": "string",
  "count": number
}
```

**Behavior:**
- Processes multiple OpenAPI specs in sequence
- Each spec is processed using the same logic as the `build` tool
- Returns all successfully generated artifacts
- If any specs fail, returns error result with details

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "buildBatch",
    "arguments": {
      "specs": [
        {
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
        },
        {
          "name": "products-api-v2",
          "engine": "go://go-gen-openapi",
          "spec": {
            "sourceFile": "./api/products-api.v2.yaml",
            "destinationDir": "./pkg/generated",
            "client": {
              "enabled": true,
              "packageName": "productsclientv2"
            }
          }
        }
      ]
    }
  }
}
```

**Note:** This tool is automatically invoked by forge when building multiple OpenAPI specs. You typically don't need to call it directly.

## Integration with Forge

### Basic Usage

In `forge.yaml`:
```yaml
build:
  # One BuildSpec per API version
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

### With Client and Server

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

This single BuildSpec generates TWO packages:
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

### Using Templated Source Path

For backward compatibility, you can use the templated pattern:

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

## Environment Variables

- **OAPI_CODEGEN_VERSION**: Version of oapi-codegen to use (default: `v2.3.0`)

## Implementation Details

- Runs `go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@{version}`
- Generates client and/or server code based on `spec` configuration
- Creates temporary oapi-codegen config files for each package
- Generates code concurrently for client and server (when both enabled)
- Returns artifact with actual output directory location

## Error Cases

### Missing Required Fields

**Error:** "name is required"
- **Cause:** BuildInput.Name is missing
- **Solution:** Provide `name` field in BuildSpec

**Error:** "engine is required"
- **Cause:** BuildInput.Engine is missing
- **Solution:** Provide `engine: go://go-gen-openapi` in BuildSpec

### Invalid Source Specification

**Error:** "must provide either 'sourceFile' or all of 'sourceDir', 'name', and 'version'"
- **Cause:** Neither source pattern is complete
- **Solution:** Use EITHER `sourceFile` OR all three of (`sourceDir`, `name`, `version`)

### Missing Package Names

**Error:** "client.packageName is required when client.enabled=true"
- **Cause:** Client is enabled but packageName is missing
- **Solution:** Provide `client.packageName` when `client.enabled=true`

**Error:** "server.packageName is required when server.enabled=true"
- **Cause:** Server is enabled but packageName is missing
- **Solution:** Provide `server.packageName` when `server.enabled=true`

### No Generators Enabled

**Error:** "at least one of client.enabled or server.enabled must be true"
- **Cause:** Both client and server are disabled or missing
- **Solution:** Enable at least one of `client.enabled` or `server.enabled`

## Generated Code

- **Client**: HTTP client with typed methods for API calls
- **Server**: HTTP server interfaces and strict handlers
- **Models**: Go structs for request/response types
- **Embedded Spec**: OpenAPI specification embedded in generated code

Output files follow the pattern:
- `{destinationDir}/{packageName}/zz_generated.oapi-codegen.go`

## See Also

- [go-build MCP Server](../go-build/MCP.md)
- [go-gen-mocks MCP Server](../go-gen-mocks/MCP.md)
- [Migration Guide](../../docs/migration-go-gen-openapi.md)
- [oapi-codegen Documentation](https://github.com/oapi-codegen/oapi-codegen)
