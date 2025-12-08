# OpenAPI Dependency Detector Usage Guide

## Purpose

`go-gen-openapi-dep-detector` is a forge engine for detecting file dependencies for OpenAPI code generation. It tracks OpenAPI specification files as dependencies, enabling lazy rebuild support for `go-gen-openapi`.

## Invocation

### MCP Mode

Run as an MCP server:

```bash
go-gen-openapi-dep-detector --mcp
```

Forge invokes this automatically when `go-gen-openapi` needs dependency detection.

## Available MCP Tools

### `detectDependencies`

Detect all file dependencies for OpenAPI code generation.

**Input Schema:**
```json
{
  "specSources": ["string (required)"],
  "rootDir": "string (optional)",
  "resolveRefs": "boolean (optional)"
}
```

**Output:**
```json
{
  "dependencies": [
    {
      "type": "file",
      "filePath": "string",
      "timestamp": "string"
    }
  ]
}
```

**Example:**
```json
{
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

### `docs-list`

List all available documentation for go-gen-openapi-dep-detector.

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

### Integration with go-gen-openapi

This detector is called automatically by `go-gen-openapi` after code generation:

1. `forge build <name>` invokes `go-gen-openapi`
2. `go-gen-openapi` generates code using oapi-codegen
3. `go-gen-openapi` extracts spec paths from its configuration
4. `go-gen-openapi` calls `go-gen-openapi-dep-detector` with the spec paths
5. Dependencies are stored in the artifact store with the artifact
6. On subsequent builds, forge compares file timestamps to decide if rebuild is needed

### Command Line Testing

```bash
# Start MCP server and send request via stdin
echo '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"detectDependencies","arguments":{"specSources":["/path/to/project/api/petstore.yaml"],"rootDir":"/path/to/project","resolveRefs":false}},"id":1}' | ./go-gen-openapi-dep-detector --mcp
```

## Implementation Details

- Iterates over provided spec source paths
- Verifies each file exists and gets its modification timestamp
- Returns a list of file dependencies with RFC3339 timestamps
- External references (`$ref`) are NOT resolved in v1

## Error Handling

### Common Errors

| Error | Description | Resolution |
|-------|-------------|------------|
| `spec file not found: /path/to/file` | The specified spec file does not exist | Verify the path is correct and the file exists |

## Limitations

### $ref Resolution NOT Supported (v1)

External references (`$ref`) are NOT automatically tracked. If your spec uses `$ref` to include external files, use `--force` for rebuilds:

```bash
forge build <name> --force
```

## See Also

- [OpenAPI Dependency Detector Configuration Schema](schema.md)
- [go-gen-openapi MCP Server](../../go-gen-openapi/MCP.md)
