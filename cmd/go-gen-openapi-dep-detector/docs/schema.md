# OpenAPI Dependency Detector Configuration Schema

## Overview

This document describes the input and output schemas for `go-gen-openapi-dep-detector`. This engine is a dependency detector that tracks OpenAPI specification files for lazy rebuild support.

## Input Schema

### detectDependencies Tool

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

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `specSources` | array of strings | Yes | - | Absolute paths to OpenAPI specification files |
| `rootDir` | string | No | - | Project root directory (reserved for future `$ref` resolution) |
| `resolveRefs` | boolean | No | `false` | Whether to resolve `$ref` references (not implemented in v1) |

## Output Schema

### detectDependencies Response

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

### Output Fields

| Field | Type | Description |
|-------|------|-------------|
| `dependencies` | array | List of file dependencies |
| `dependencies[].type` | string | Always `"file"` for this detector |
| `dependencies[].filePath` | string | Absolute path to the dependency file |
| `dependencies[].timestamp` | string | RFC3339 UTC timestamp of file modification |

## Examples

### Basic Detection

**Request:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "detectDependencies",
    "arguments": {
      "specSources": [
        "/path/to/project/api/petstore.yaml"
      ],
      "rootDir": "/path/to/project",
      "resolveRefs": false
    }
  }
}
```

**Response:**
```json
{
  "dependencies": [
    {
      "type": "file",
      "filePath": "/path/to/project/api/petstore.yaml",
      "timestamp": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### Multiple Spec Files

**Request:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "detectDependencies",
    "arguments": {
      "specSources": [
        "/path/to/project/api/petstore.yaml",
        "/path/to/project/api/users.yaml"
      ]
    }
  }
}
```

**Response:**
```json
{
  "dependencies": [
    {
      "type": "file",
      "filePath": "/path/to/project/api/petstore.yaml",
      "timestamp": "2024-01-15T10:30:00Z"
    },
    {
      "type": "file",
      "filePath": "/path/to/project/api/users.yaml",
      "timestamp": "2024-01-14T09:00:00Z"
    }
  ]
}
```

### Empty Input

**Request:**
```json
{
  "specSources": [],
  "rootDir": "/path/to/project",
  "resolveRefs": false
}
```

**Response:**
```json
{
  "dependencies": []
}
```

## Dependency Types

The detector tracks only OpenAPI specification files:

1. **Spec files**: All specification files provided in `specSources`

## Future: $ref Resolution

The `resolveRefs` and `rootDir` parameters are reserved for future implementation of `$ref` resolution. When implemented, the detector will:

1. Parse each spec file
2. Find all `$ref` references to external files
3. Recursively track referenced files as dependencies

Currently, if `resolveRefs: true` is requested:
- A warning is logged
- Detection continues with only the explicit spec files
- No error is returned

## See Also

- [OpenAPI Dependency Detector Usage Guide](usage.md)
- [go-gen-openapi MCP Server](../../go-gen-openapi/MCP.md)
