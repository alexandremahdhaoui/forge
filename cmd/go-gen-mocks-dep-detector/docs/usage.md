# Mock Dependency Detector Usage Guide

## Purpose

`go-gen-mocks-dep-detector` is a forge engine for detecting file dependencies for mockery mock generation. It analyzes `.mockery.yaml` configuration files and resolves Go package paths to source files, enabling lazy rebuild support for `go-gen-mocks`.

## Invocation

### MCP Mode

Run as an MCP server:

```bash
go-gen-mocks-dep-detector --mcp
```

Forge invokes this automatically when `go-gen-mocks` needs dependency detection.

## Available MCP Tools

### `detectDependencies`

Detect all file dependencies for mock generation.

**Input Schema:**
```json
{
  "workDir": "string (required)"
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
      "workDir": "/path/to/project"
    }
  }
}
```

### `docs-list`

List all available documentation for go-gen-mocks-dep-detector.

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

### Integration with go-gen-mocks

This detector is called automatically by `go-gen-mocks` after mock generation:

1. `forge build mocks` invokes `go-gen-mocks`
2. `go-gen-mocks` generates mocks using mockery
3. `go-gen-mocks` calls `go-gen-mocks-dep-detector` to detect dependencies
4. Dependencies are stored in the artifact store with the artifact
5. On subsequent builds, forge compares file timestamps to decide if rebuild is needed

### Mockery Config Discovery

The detector searches for mockery configuration in the following order:

1. `MOCKERY_CONFIG_PATH` environment variable (if set and file exists)
2. `.mockery.yaml` in workDir
3. `.mockery.yml` in workDir
4. `mockery.yaml` in workDir
5. `mockery.yml` in workDir

### Detected Dependencies

The detector returns these types of dependencies:

1. **Mockery config file** - The `.mockery.yaml` configuration file itself
2. **go.mod file** - The project's go.mod file
3. **Interface source files** - All `.go` files (excluding `_test.go`) from packages listed in the mockery config

## Implementation Details

- Parses mockery YAML configuration to find packages
- Resolves Go package paths to local file paths
- Tracks all `.go` files in interface packages
- Returns RFC3339 timestamps for lazy rebuild support
- External packages are logged as warnings but do not fail detection

## Error Handling

### Common Errors

| Error | Description | Resolution |
|-------|-------------|------------|
| `no mockery config found` | No mockery configuration file found in workDir | Create a `.mockery.yaml` file or set `MOCKERY_CONFIG_PATH` |
| `go.mod not found` | Cannot find go.mod in workDir or parent directories | Ensure the workDir is within a Go module |
| `failed to parse mockery config` | Invalid YAML syntax in mockery config | Fix the YAML syntax in your mockery config file |

## Limitations

### External Packages NOT Supported (v1)

External packages referenced in `.mockery.yaml` are NOT tracked for lazy rebuild in v1. When external interface dependencies change, use `--force`:

```bash
forge build mocks --force
```

## See Also

- [Mock Dependency Detector Configuration Schema](schema.md)
- [go-gen-mocks MCP Server](../../go-gen-mocks/MCP.md)
