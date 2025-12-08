# Generic Builder Usage Guide

## Purpose

`generic-builder` is a forge engine for executing arbitrary shell commands as build steps. Use it to integrate any CLI tool into forge builds without writing custom Go code.

## Invocation

### MCP Mode

Run as an MCP server:

```bash
generic-builder --mcp
```

Forge invokes this automatically when using:

```yaml
engine: go://generic-builder
```

## Available MCP Tools

### `build`

Execute a shell command and return structured output.

**Input Schema:**
```json
{
  "name": "string (required)",
  "command": "string (required)",
  "args": ["string"],
  "env": {"key": "value"},
  "envFile": "string",
  "workDir": "string",
  "src": "string",
  "dest": "string",
  "version": "string"
}
```

**Output:**
```json
{
  "name": "string",
  "type": "command-output",
  "location": "string",
  "timestamp": "string",
  "version": "string"
}
```

**Example - Run formatter:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "build",
    "arguments": {
      "name": "format-code",
      "command": "gofumpt",
      "args": ["-w", "{{ .Src }}"],
      "src": "./cmd/myapp"
    }
  }
}
```

### `buildBatch`

Execute multiple commands in sequence.

**Input Schema:**
```json
{
  "specs": [
    {
      "name": "string",
      "command": "string",
      "args": ["string"]
    }
  ]
}
```

**Output:**
Array of Artifacts with summary of successes/failures.

### `docs-list`

List all available documentation for generic-builder.

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

## Template Support

Arguments support Go template syntax with these fields:
- `{{ .Name }}` - Build name
- `{{ .Src }}` - Source directory
- `{{ .Dest }}` - Destination directory
- `{{ .Version }}` - Version string

## Common Use Cases

### Code Formatting

```yaml
build:
  - name: format-code
    command: gofumpt
    args: ["-w", "./..."]
    workDir: .
    engine: go://generic-builder
```

### Code Generation with Protoc

```yaml
build:
  - name: generate-proto
    command: protoc
    args:
      - "--go_out={{ .Dest }}"
      - "--go_opt=paths=source_relative"
      - "{{ .Src }}/api.proto"
    src: ./proto
    dest: ./pkg/api
    engine: go://generic-builder
```

### Mock Generation

```yaml
build:
  - name: generate-mocks
    command: mockery
    args: ["--all", "--output", "{{ .Dest }}"]
    dest: ./mocks
    engine: go://generic-builder
```

### Asset Compilation

```yaml
build:
  - name: compile-assets
    command: npm
    args: ["run", "build"]
    workDir: ./frontend
    engine: go://generic-builder
```

### Custom Script Execution

```yaml
build:
  - name: custom-build
    command: ./scripts/build.sh
    args: ["{{ .Name }}", "{{ .Version }}"]
    env:
      BUILD_MODE: production
    engine: go://generic-builder
```

## Error Handling

- Exit code 0: Success - Returns Artifact
- Exit code != 0: Failure - Returns error with stdout/stderr

## Implementation Details

- Executes commands via exec.Command
- Captures stdout, stderr, and exit code
- Processes template arguments before execution
- Working directory defaults to current directory
- Environment variables are passed to the command

## See Also

- [Generic Builder Configuration Schema](schema.md)
- [go-build MCP Server](../../go-build/docs/usage.md)
