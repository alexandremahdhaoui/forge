# go-gen-protobuf

**Compile Protocol Buffer files to Go code with gRPC support.**

> "Managing protoc commands across multiple proto directories was a nightmare. go-gen-protobuf handles all the discovery and compilation automatically - I just point it at a directory and get perfectly generated Go code."

## What problem does go-gen-protobuf solve?

Compiling .proto files requires running protoc with multiple flags and plugins. go-gen-protobuf automates proto file discovery, protoc invocation, and dependency tracking for lazy rebuild support.

## How do I use go-gen-protobuf?

Add a build target to `forge.yaml`:

```yaml
build:
  - name: protobuf
    src: ./proto
    dest: ./proto
    engine: go://go-gen-protobuf
```

Run the generator:

```bash
forge build
```

## What configuration options are available?

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Build step name |
| `src` | Yes | Directory containing .proto files |
| `dest` | Yes | Output directory for generated Go code |
| `spec.go_opt` | No | Options for --go_opt flag |
| `spec.go-grpc_opt` | No | Options for --go-grpc_opt flag |
| `spec.proto_path` | No | Additional proto include paths |
| `spec.plugin` | No | Additional protoc plugins |
| `spec.extra_args` | No | Additional protoc arguments |

## How do I use custom protoc options?

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

## How do I compile multiple proto directories?

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

## What prerequisites are required?

Install protoc plugins:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## How does it work?

- Recursively discovers all .proto files in src directory
- Skips hidden directories and vendor
- Generates both message code (--go_out) and gRPC code (--go-grpc_out)
- Tracks all .proto files as dependencies for lazy rebuild

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../MCP.md) - MCP tool documentation
- [Protocol Buffers docs](https://protobuf.dev/) - Upstream documentation
