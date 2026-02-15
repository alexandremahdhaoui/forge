# go-gen-mocks-dep-detector MCP Server

Dependency detector for mockery mock generation. This MCP server analyzes `.mockery.yaml` configuration files and resolves Go package paths to source files, enabling lazy rebuild support for `go-gen-mocks`.

## Overview

The `go-gen-mocks-dep-detector` is an MCP server that detects file dependencies for mock generation. It is called by `go-gen-mocks` after mock generation to track which source files affect the generated mocks, enabling forge's lazy rebuild system to skip unnecessary rebuilds when source files haven't changed.

**URI:** `go://go-gen-mocks-dep-detector`

## Tools

### detectDependencies

Detects all dependencies for mockery mock generation by:
1. Finding and parsing the mockery configuration file
2. Finding go.mod to determine the module path
3. Resolving each package in the config to its source files
4. Returning a list of file dependencies with timestamps

## Input Schema

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

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `workDir` | string | Yes | The working directory where the mockery config is located. This is typically the project root directory. |

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

1. **Mockery config file** - The `.mockery.yaml` (or similar) configuration file itself
2. **go.mod file** - The project's go.mod file
3. **Interface source files** - All `.go` files (excluding `_test.go`) from packages listed in the mockery config

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
      "workDir": "/path/to/project"
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
        "text": "Detected 5 dependencies for mock generation"
      }
    ],
    "_meta": {
      "dependencies": [
        {
          "type": "file",
          "filePath": "/path/to/project/.mockery.yaml",
          "timestamp": "2025-11-25T10:00:00Z"
        },
        {
          "type": "file",
          "filePath": "/path/to/project/go.mod",
          "timestamp": "2025-11-25T09:30:00Z"
        },
        {
          "type": "file",
          "filePath": "/path/to/project/pkg/store/store.go",
          "timestamp": "2025-11-25T08:00:00Z"
        },
        {
          "type": "file",
          "filePath": "/path/to/project/pkg/cache/cache.go",
          "timestamp": "2025-11-24T15:30:00Z"
        },
        {
          "type": "file",
          "filePath": "/path/to/project/pkg/cache/types.go",
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
echo '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"detectDependencies","arguments":{"workDir":"/path/to/project"}},"id":1}' | ./go-gen-mocks-dep-detector --mcp
```

## Mockery Config Discovery

The detector searches for mockery configuration in the following order:

1. `MOCKERY_CONFIG_PATH` environment variable (if set and file exists)
2. `.mockery.yaml` in workDir
3. `.mockery.yml` in workDir
4. `mockery.yaml` in workDir
5. `mockery.yml` in workDir

If no configuration file is found, the tool returns an error.

## Error Handling

### Common Errors

| Error | Description | Resolution |
|-------|-------------|------------|
| `no mockery config found` | No mockery configuration file found in workDir | Create a `.mockery.yaml` file or set `MOCKERY_CONFIG_PATH` |
| `go.mod not found` | Cannot find go.mod in workDir or parent directories | Ensure the workDir is within a Go module |
| `failed to parse mockery config` | Invalid YAML syntax in mockery config | Fix the YAML syntax in your mockery config file |
| `package X is external, not tracked in v1` | Package is from an external module | This is a warning, not a failure. Use `--force` flag for rebuilds when external deps change |

### Error Response Example

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Mock dependency detection failed: no mockery config found in /path/to/project and MOCKERY_CONFIG_PATH not set"
      }
    ],
    "isError": true
  }
}
```

## Scope Limitations

### External Packages NOT Supported (v1)

**Important:** External packages referenced in `.mockery.yaml` are NOT tracked for lazy rebuild in v1.

If your mockery configuration references an external package (e.g., `github.com/some/external/pkg`), the detector will:

1. Log a warning: `Warning: skipping package github.com/some/external/pkg: package is external (not under module X), not tracked in v1`
2. Skip the package (the detection does NOT fail)
3. Continue processing other packages
4. Return dependencies for LOCAL packages only

**Example:**
```yaml
# .mockery.yaml
packages:
  github.com/myorg/myproject/pkg/store:  # LOCAL - tracked
    interfaces:
      Repository:
  github.com/external/library/client:     # EXTERNAL - NOT tracked (warning logged)
    interfaces:
      Client:
```

In the above example:
- `github.com/myorg/myproject/pkg/store` will be resolved and its files tracked
- `github.com/external/library/client` will be skipped with a warning

### Workaround for External Package Changes

When external interface dependencies change (e.g., after `go get -u`), the lazy rebuild system will not detect this change. Use the `--force` flag to ensure mocks are regenerated:

```bash
forge build mocks --force
```

### Reporting Issues

If you encounter issues with external package handling or need this limitation addressed, please report at:
https://github.com/alexandremahdhaoui/forge/issues

## Integration with go-gen-mocks

This detector is called automatically by `go-gen-mocks` after mock generation. The dependencies are stored in the artifact store and used by forge's `shouldRebuild()` logic to determine if mocks need to be regenerated.

**Build Flow:**
1. `forge build mocks` invokes `go-gen-mocks`
2. `go-gen-mocks` generates mocks using mockery
3. `go-gen-mocks` calls `go-gen-mocks-dep-detector` to detect dependencies
4. Dependencies are stored in the artifact store with the artifact
5. On subsequent builds, forge compares file timestamps to decide if rebuild is needed

## Version Information

The detector follows the standard forge versioning:
- Version is injected via ldflags during build
- CommitSHA and BuildTimestamp are also available
- Use `--version` flag to display version information

## Related Documentation

- [Built-in Tools Reference](../../docs/built-in-tools.md)
- [Forge Design Document](../../DESIGN.md) - Lazy Rebuild section
- [go-gen-mocks](../go-gen-mocks/README.md)
