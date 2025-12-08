# Protobuf Code Generator Configuration Schema

## Overview

This document describes the configuration options for `go-gen-protobuf` in `forge.yaml`. The go-gen-protobuf engine compiles Protocol Buffer definitions to Go code using protoc.

## Basic Configuration

```yaml
build:
  - name: protobuf
    src: ./proto
    dest: ./proto
    engine: go://go-gen-protobuf
```

## Configuration Options

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Artifact identifier for the artifact store |
| `src` | string | Root directory to search for .proto files (recursive) |
| `dest` | string | Output directory for generated Go files |
| `engine` | string | Must be `go://go-gen-protobuf` to use this generator |

### Spec Options

The `spec` field contains engine-specific configuration:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `go_opt` | string | `paths=source_relative` | Value for `--go_opt` flag |
| `go-grpc_opt` | string | `paths=source_relative` | Value for `--go-grpc_opt` flag |
| `proto_path` | string or string[] | - | Additional proto import paths |
| `plugin` | string[] | - | Custom protoc plugins in `name=path` format |
| `extra_args` | string[] | - | Additional raw protoc arguments |

## Examples

### Minimal Configuration

```yaml
build:
  - name: protobuf
    src: ./proto
    dest: ./proto
    engine: go://go-gen-protobuf
```

### Full Configuration

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

### With Google APIs

```yaml
build:
  - name: api-protobuf
    src: ./api
    dest: ./api
    engine: go://go-gen-protobuf
    spec:
      proto_path:
        - "./vendor/googleapis"
      extra_args:
        - "--go-grpc_out=require_unimplemented_servers=false:./api"
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
    spec:
      go_opt: "paths=source_relative,M=./api/proto"
```

## Generated Output

Each build creates an artifact in the artifact store:

```yaml
artifacts:
  - name: api-protobuf
    type: protobuf
    location: ./api
    timestamp: "2024-01-15T10:30:00Z"
    dependencies:
      - type: file
        filePath: /absolute/path/to/api/v1/service.proto
        timestamp: "2024-01-14T09:00:00Z"
      - type: file
        filePath: /absolute/path/to/api/v1/messages.proto
        timestamp: "2024-01-14T09:00:00Z"
    dependencyDetectorEngine: go://go-gen-protobuf
```

## Proto Discovery Rules

The engine discovers proto files with these rules:

1. Recursively walks the `src` directory
2. Includes all files with `.proto` extension
3. Skips hidden directories (starting with `.`)
4. Skips `vendor` directory

## Protoc Arguments Order

Arguments are passed to protoc in this order:

1. `--proto_path={src}` - Source directory (always first)
2. `--go_out={dest}` - Go output directory
3. `--go_opt={go_opt}` - Go options
4. `--go-grpc_out={dest}` - gRPC output directory
5. `--go-grpc_opt={go-grpc_opt}` - gRPC options
6. User `proto_path` entries - Additional import paths
7. `plugin` entries - Custom plugins
8. `extra_args` - Additional arguments
9. Proto files - All discovered .proto files

## Default Behavior

When no `spec` is provided, go-gen-protobuf uses these defaults:

- `go_opt=paths=source_relative`
- `go-grpc_opt=paths=source_relative`
- Source directory added as first `--proto_path`
- Both Go message code and gRPC service code generated

## Error Cases

| Error | Cause | Solution |
|-------|-------|----------|
| `src is required` | Missing src field | Add `src` field pointing to proto directory |
| `dest is required` | Missing dest field | Add `dest` field for output directory |
| `no .proto files found` | Empty source directory | Ensure src contains .proto files |
| `protoc failed` | Invalid proto syntax | Check protoc output for details |

## See Also

- [Protobuf Code Generator Usage Guide](usage.md)
- [Protocol Buffers Language Guide](https://protobuf.dev/programming-guides/proto3/)
