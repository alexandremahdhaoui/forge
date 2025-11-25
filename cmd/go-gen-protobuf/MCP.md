# go-gen-protobuf MCP Server

MCP server for compiling Protocol Buffer (.proto) files to Go code using protoc.

## Purpose

Provides MCP tools for generating Go code from Protocol Buffer definitions using protoc with dependency tracking for lazy-rebuild support. Tracks all .proto files as dependencies to enable incremental builds.

## Invocation

```bash
go-gen-protobuf --mcp
```

Forge invokes this automatically via:
```yaml
engine: go://go-gen-protobuf
```

## Available Tools

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
    "go_opt": "string (optional, default: paths=source_relative)",
    "go-grpc_opt": "string (optional, default: paths=source_relative)",
    "proto_path": "string | string[] (optional)",
    "plugin": "string[] (optional)",
    "extra_args": "string[] (optional)"
  }
}
```

**Field Descriptions:**

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Artifact identifier for the artifact store |
| `src` | Yes | Root directory to search for .proto files (recursive) |
| `dest` | Yes | Output directory for generated Go files |
| `engine` | Yes | Must be `go://go-gen-protobuf` |
| `spec.go_opt` | No | Value for `--go_opt` (default: `paths=source_relative`) |
| `spec.go-grpc_opt` | No | Value for `--go-grpc_opt` (default: `paths=source_relative`) |
| `spec.proto_path` | No | Additional proto import paths (single string or array) |
| `spec.plugin` | No | Custom protoc plugins in `name=path` format |
| `spec.extra_args` | No | Additional raw protoc arguments |

**Output Schema:**
```json
{
  "name": "string",
  "type": "protobuf",
  "location": "string",
  "timestamp": "string (RFC3339)",
  "dependencies": [
    {
      "type": "file",
      "filePath": "string (absolute path)",
      "timestamp": "string (RFC3339)"
    }
  ],
  "dependencyDetectorEngine": "go://go-gen-protobuf"
}
```

**Output Field Descriptions:**

| Field | Description |
|-------|-------------|
| `name` | Artifact name from input |
| `type` | Fixed value: `protobuf` |
| `location` | Output directory containing generated files |
| `timestamp` | Build completion time in RFC3339 format (UTC) |
| `dependencies` | Array of tracked .proto files with modification times |
| `dependencyDetectorEngine` | Engine URI for lazy-rebuild support |

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

**Note:** This tool is automatically registered by the engine framework. You typically don't need to call it directly.

## Integration with Forge

### Basic Usage

In `forge.yaml`:
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

### Full Configuration with All Options

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
      plugin:
        - "protoc-gen-go=/go/bin/protoc-gen-go"
        - "protoc-gen-go-grpc=/go/bin/protoc-gen-go-grpc"
      extra_args:
        - "--experimental_allow_proto3_optional"
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
- Skips hidden directories (starting with `.`)
- Executes `protoc` with configured options
- Generates both message code (`--go_out`) and gRPC service code (`--go-grpc_out`)
- Tracks all `.proto` files as dependencies for lazy-rebuild
- Sets `DependencyDetectorEngine` to enable incremental builds

**Protoc Command Construction:**

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

**Note:** The source directory is always added as the first `--proto_path` to enable import resolution between proto files.

## Error Cases

### Missing Required Fields

**Error:** `src is required`
- **Cause:** BuildInput.Src is missing or empty
- **Solution:** Provide `src` field pointing to proto directory

**Error:** `dest is required`
- **Cause:** BuildInput.Dest is missing or empty
- **Solution:** Provide `dest` field for output directory

### No Proto Files Found

**Error:** `no .proto files found in {src}`
- **Cause:** Source directory contains no .proto files
- **Solution:** Verify source path and ensure .proto files exist

### Protoc Execution Failure

**Error:** `protoc failed: {error}`
- **Cause:** protoc command failed (syntax error, missing imports, etc.)
- **Solution:** Check protoc output for details, verify proto files are valid

### Missing Dependencies

**Error:** `failed to discover proto files: {error}`
- **Cause:** Source directory doesn't exist or isn't readable
- **Solution:** Verify source directory exists and has correct permissions

## Prerequisites

The following tools must be installed and available in PATH:
- `protoc` - Protocol Buffer compiler
- `protoc-gen-go` - Go code generator plugin (`go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`)
- `protoc-gen-go-grpc` - gRPC code generator plugin (`go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest`)

## See Also

- [go-build MCP Server](../go-build/MCP.md)
- [go-gen-openapi MCP Server](../go-gen-openapi/MCP.md)
- [Forge Build Documentation](../../docs/forge-usage.md#building-artifacts)
- [Protocol Buffers Documentation](https://protobuf.dev/)
