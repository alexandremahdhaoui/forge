# go-gen-openapi-dep-detector MCP Server

Dependency detector for OpenAPI code generation. This MCP server tracks OpenAPI specification files as dependencies, enabling lazy rebuild support for `go-gen-openapi`.

## Overview

The `go-gen-openapi-dep-detector` is an MCP server that detects file dependencies for OpenAPI code generation. It is called by `go-gen-openapi` after code generation to track which specification files affect the generated code, enabling forge's lazy rebuild system to skip unnecessary rebuilds when spec files haven't changed.

**URI:** `go://go-gen-openapi-dep-detector`

## Tools

### detectDependencies

Detects all dependencies for OpenAPI code generation by:
1. Iterating over the provided spec source paths
2. Verifying each file exists and getting its modification timestamp
3. Returning a list of file dependencies with timestamps

## Input Schema

```json
{
  "type": "object",
  "properties": {
    "specSources": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "Absolute paths to OpenAPI spec files"
    },
    "rootDir": {
      "type": "string",
      "description": "Project root directory (for future $ref resolution)"
    },
    "resolveRefs": {
      "type": "boolean",
      "description": "Whether to resolve $ref references (v1: always ignored, not implemented)"
    }
  },
  "required": ["specSources"]
}
```

### Input Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `specSources` | array of strings | Yes | Absolute paths to OpenAPI specification files to track as dependencies |
| `rootDir` | string | No | Project root directory. Reserved for future `$ref` resolution support |
| `resolveRefs` | boolean | No | Whether to resolve `$ref` references. **Note: Not implemented in v1** |

## Output Schema

```json
{
  "type": "object",
  "properties": {
    "dependencies": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "type": {
            "type": "string",
            "enum": ["file"],
            "description": "Dependency type (always 'file' for this detector)"
          },
          "filePath": {
            "type": "string",
            "description": "Absolute path to the dependency file"
          },
          "timestamp": {
            "type": "string",
            "description": "RFC3339 timestamp of the file's last modification in UTC"
          }
        }
      }
    }
  }
}
```

### Output Dependencies

The detector returns the following types of dependencies:

1. **OpenAPI spec files** - All specification files provided in `specSources`

## Example Usage

### MCP Request

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "detectDependencies",
    "arguments": {
      "specSources": [
        "/path/to/project/api/petstore.yaml",
        "/path/to/project/api/users.yaml"
      ],
      "rootDir": "/path/to/project",
      "resolveRefs": false
    }
  }
}
```

### MCP Response (Success)

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Detected 2 dependencies for OpenAPI generation"
      }
    ],
    "_meta": {
      "dependencies": [
        {
          "type": "file",
          "filePath": "/path/to/project/api/petstore.yaml",
          "timestamp": "2025-11-25T10:00:00Z"
        },
        {
          "type": "file",
          "filePath": "/path/to/project/api/users.yaml",
          "timestamp": "2025-11-24T15:30:00Z"
        }
      ]
    }
  }
}
```

### Command Line Testing

```bash
# Start MCP server and send request via stdin
echo '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"detectDependencies","arguments":{"specSources":["/path/to/project/api/petstore.yaml"],"rootDir":"/path/to/project","resolveRefs":false}},"id":1}' | ./go-gen-openapi-dep-detector --mcp
```

## Error Handling

### Common Errors

| Error | Description | Resolution |
|-------|-------------|------------|
| `spec file not found: /path/to/file` | The specified spec file does not exist | Verify the path is correct and the file exists |

### Error Response Example

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "OpenAPI dependency detection failed: spec file not found: /path/to/missing.yaml: stat /path/to/missing.yaml: no such file or directory"
      }
    ],
    "isError": true
  }
}
```

## Scope Limitations

### $ref Resolution NOT Supported (v1)

**Important:** `$ref` resolution is NOT implemented in v1. Only the explicitly provided spec files are tracked as dependencies.

If your OpenAPI specification uses `$ref` to reference external files:

```yaml
# petstore.yaml
openapi: "3.0.0"
paths:
  /pets:
    get:
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: "./schemas/pet.yaml#/Pet"  # NOT tracked in v1
```

The referenced file (`./schemas/pet.yaml`) will NOT be automatically tracked. Only the main spec file (`petstore.yaml`) is tracked.

**Behavior when `resolveRefs: true` is requested:**
- A warning is logged: `Warning: $ref resolution requested but not implemented in v1`
- Detection continues with only the explicit spec files
- No error is returned

### Workaround for $ref Dependencies

When files referenced via `$ref` change, the lazy rebuild system will not detect this change. Use the `--force` flag to ensure code is regenerated:

```bash
forge build <name> --force
```

**Alternative:** Explicitly include all referenced files in your build configuration:

```yaml
build:
  - name: api-client
    engine: go://go-gen-openapi
    spec:
      specs:
        - source: ./api/petstore.yaml
          destinationDir: ./pkg/petstore
          client:
            enabled: true
        - source: ./api/schemas/pet.yaml     # Explicitly include referenced files
          destinationDir: ./pkg/petstore
          client:
            enabled: false                    # Don't generate, just track
```

### Reporting Issues

If you encounter issues with `$ref` resolution or need this limitation addressed, please report at:
https://github.com/alexandremahdhaoui/forge/issues

## Integration with go-gen-openapi

This detector is called automatically by `go-gen-openapi` after code generation. The dependencies are stored in the artifact store and used by forge's `shouldRebuild()` logic to determine if code needs to be regenerated.

**Build Flow:**
1. `forge build <name>` invokes `go-gen-openapi`
2. `go-gen-openapi` generates code using oapi-codegen
3. `go-gen-openapi` extracts spec paths from its configuration
4. `go-gen-openapi` calls `go-gen-openapi-dep-detector` with the spec paths
5. Dependencies are stored in the artifact store with the artifact
6. On subsequent builds, forge compares file timestamps to decide if rebuild is needed

## Empty Input Handling

If `specSources` is an empty array, the detector returns an empty dependencies list without error:

```json
{
  "specSources": [],
  "rootDir": "/path/to/project",
  "resolveRefs": false
}
```

Response:
```json
{
  "dependencies": []
}
```

## Version Information

The detector follows the standard forge versioning:
- Version is injected via ldflags during build
- CommitSHA and BuildTimestamp are also available
- Use `--version` flag to display version information

## Related Documentation

- [Built-in Tools Reference](../../docs/built-in-tools.md)
- [Forge Architecture](../../ARCHITECTURE.md) - Lazy Rebuild section
- [go-gen-openapi](../go-gen-openapi/MCP.md)
- [OpenAPI Migration Guide](../../docs/migration-go-gen-openapi.md)
