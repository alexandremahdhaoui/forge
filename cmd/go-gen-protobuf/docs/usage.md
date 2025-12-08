# Protobuf Code Generator Usage Guide

## Purpose

`go-gen-protobuf` is a forge engine for compiling Protocol Buffer (.proto) files to Go code using protoc. It provides automatic dependency tracking for lazy-rebuild support and handles gRPC service generation.

## Invocation

### MCP Mode

Run as an MCP server:

```bash
go-gen-protobuf --mcp
```

Forge invokes this automatically when using:

```yaml
engine: go://go-gen-protobuf
```

## Available MCP Tools

### `build`

Generate Go code from Protocol Buffer files.

**Input Schema:**
```json
{
  "name": "string (required)",
  "src": "string (required)",
  "dest": "string (required)",
  "engine": "string (required)",
  "spec": {
    "go_opt": "string (optional)",
    "go-grpc_opt": "string (optional)",
    "proto_path": "string | string[] (optional)",
    "plugin": "string[] (optional)",
    "extra_args": "string[] (optional)"
  }
}
```

**Output:**
```json
{
  "name": "string",
  "type": "protobuf",
  "location": "string",
  "timestamp": "string",
  "dependencies": [
    {
      "type": "file",
      "filePath": "string",
      "timestamp": "string"
    }
  ],
  "dependencyDetectorEngine": "go://go-gen-protobuf"
}
```

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "build",
    "arguments": {
      "name": "api-protobuf",
      "src": "./api",
      "dest": "./api",
      "engine": "go://go-gen-protobuf",
      "spec": {
        "go_opt": "paths=source_relative",
        "go-grpc_opt": "paths=source_relative"
      }
    }
  }
}
```

### `buildBatch`

Generate Go code for multiple proto directories in sequence.

**Input Schema:**
```json
{
  "specs": [
    {
      "name": "string",
      "src": "string",
      "dest": "string",
      "engine": "string",
      "spec": { }
    }
  ]
}
```

**Output:**
Array of Artifacts with summary of successes/failures.

### `docs-list`

List all available documentation for go-gen-protobuf.

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

### Basic Protobuf Generation

Generate Go code from proto files:

```yaml
build:
  - name: protobuf
    src: ./proto
    dest: ./proto
    engine: go://go-gen-protobuf
```

Run with:

```bash
forge build
```

### With Custom Options

```yaml
build:
  - name: api-protobuf
    src: ./api
    dest: ./api
    engine: go://go-gen-protobuf
    spec:
      go_opt: "paths=source_relative"
      go-grpc_opt: "paths=source_relative"
      proto_path:
        - "./vendor/googleapis"
        - "./vendor/grpc-gateway"
```

### Multiple Proto Directories

```yaml
build:
  - name: api-protobuf
    src: ./api/proto
    dest: ./api/proto
    engine: go://go-gen-protobuf

  - name: internal-protobuf
    src: ./internal/proto
    dest: ./internal/proto
    engine: go://go-gen-protobuf
```

## Implementation Details

- Recursively discovers all `.proto` files in the source directory
- Skips hidden directories (starting with `.`) and `vendor` directory
- Executes `protoc` with configured options
- Generates both message code (`--go_out`) and gRPC service code (`--go-grpc_out`)
- Tracks all `.proto` files as dependencies for lazy-rebuild
- Sets `DependencyDetectorEngine` to enable incremental builds

## Protoc Command Construction

Given `src: ./api` and `dest: ./api`, the engine constructs:

```bash
protoc \
  --proto_path=./api \
  --go_out=./api \
  --go_opt=paths=source_relative \
  --go-grpc_out=./api \
  --go-grpc_opt=paths=source_relative \
  [additional proto_paths] \
  [plugins] \
  [extra_args] \
  ./api/v1/foo.proto ./api/v1/bar.proto
```

## Prerequisites

The following tools must be installed and available in PATH:
- `protoc` - Protocol Buffer compiler
- `protoc-gen-go` - Go code generator plugin
- `protoc-gen-go-grpc` - gRPC code generator plugin

Install with:
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## Error Handling

### Common Errors

| Error | Description | Resolution |
|-------|-------------|------------|
| `src is required` | Source directory not specified | Provide `src` field in BuildSpec |
| `dest is required` | Destination directory not specified | Provide `dest` field in BuildSpec |
| `no .proto files found in {src}` | No proto files in source directory | Verify source path contains .proto files |
| `protoc failed: {error}` | protoc command failed | Check protoc output for syntax errors or missing imports |

## See Also

- [Protobuf Code Generator Configuration Schema](schema.md)
- [Protocol Buffers Documentation](https://protobuf.dev/)
- [gRPC Documentation](https://grpc.io/docs/)
