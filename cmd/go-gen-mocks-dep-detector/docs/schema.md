# Mock Dependency Detector Configuration Schema

## Overview

This document describes the input and output schemas for `go-gen-mocks-dep-detector`. This engine is a dependency detector that analyzes mockery configuration to find source files.

## Input Schema

### detectDependencies Tool

```json
{
  "type": "object",
  "properties": {
    "workDir": {
      "type": "string",
      "description": "Directory to search for .mockery.yaml (typically project root)"
    }
  },
  "required": ["workDir"]
}
```

### Input Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `workDir` | string | Yes | - | The working directory where the mockery config is located |

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
      "workDir": "/path/to/project"
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
      "filePath": "/path/to/project/.mockery.yaml",
      "timestamp": "2024-01-15T10:30:00Z"
    },
    {
      "type": "file",
      "filePath": "/path/to/project/go.mod",
      "timestamp": "2024-01-14T09:00:00Z"
    },
    {
      "type": "file",
      "filePath": "/path/to/project/pkg/store/store.go",
      "timestamp": "2024-01-13T15:30:00Z"
    }
  ]
}
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `MOCKERY_CONFIG_PATH` | Override the mockery config file path |

## Mockery Config Format

The detector parses mockery configuration in YAML format:

```yaml
# .mockery.yaml
packages:
  github.com/myorg/myproject/pkg/store:
    interfaces:
      Repository:
  github.com/myorg/myproject/pkg/cache:
    interfaces:
      Cache:
```

For each package listed, the detector:
1. Resolves the package path to a local directory
2. Finds all `.go` files (excluding `_test.go`)
3. Records each file with its modification timestamp

## Dependency Types

The detector tracks three categories of dependencies:

1. **Configuration**: The mockery config file (`.mockery.yaml`)
2. **Module**: The `go.mod` file
3. **Source**: Go source files containing interfaces

## See Also

- [Mock Dependency Detector Usage Guide](usage.md)
- [go-gen-mocks MCP Server](../../go-gen-mocks/MCP.md)
