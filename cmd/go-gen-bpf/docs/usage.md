# BPF Code Generator Usage Guide

## Purpose

`go-gen-bpf` is a forge engine for generating Go code from BPF C source files using bpf2go from the cilium/ebpf library. It provides automatic dependency tracking for lazy-rebuild support.

## Invocation

### MCP Mode

Run as an MCP server:

```bash
go-gen-bpf --mcp
```

Forge invokes this automatically when using:

```yaml
engine: go://go-gen-bpf
```

## Available MCP Tools

### `build`

Generate Go code from a BPF C source file.

**Input Schema:**
```json
{
  "name": "string (required)",
  "src": "string (required)",
  "dest": "string (required)",
  "engine": "string (required)",
  "spec": {
    "ident": "string (required)",
    "bpf2goVersion": "string (optional)",
    "goPackage": "string (optional)",
    "outputStem": "string (optional)",
    "tags": ["string"] (optional)",
    "types": ["string"] (optional)",
    "cflags": ["string"] (optional)",
    "cc": "string (optional)"
  }
}
```

**Output:**
```json
{
  "name": "string",
  "type": "bpf",
  "location": "string",
  "timestamp": "string",
  "dependencies": [
    {
      "type": "file",
      "filePath": "string",
      "timestamp": "string"
    }
  ],
  "dependencyDetectorEngine": "go://go-gen-bpf"
}
```

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "build",
    "arguments": {
      "name": "my-bpf-program",
      "src": "./bpf/program.c",
      "dest": "./pkg/bpf",
      "engine": "go://go-gen-bpf",
      "spec": {
        "ident": "myProgram"
      }
    }
  }
}
```

### `buildBatch`

Generate Go code for multiple BPF programs in sequence.

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

List all available documentation for go-gen-bpf.

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

### Basic BPF Generation

Generate Go code from a BPF C source file:

```yaml
build:
  - name: my-bpf-program
    src: ./bpf/program.c
    dest: ./pkg/bpf
    engine: go://go-gen-bpf
    spec:
      ident: myProgram
```

Run with:

```bash
forge build
```

### With Custom Options

```yaml
build:
  - name: advanced-bpf
    src: ./bpf/advanced.c
    dest: ./pkg/bpf
    engine: go://go-gen-bpf
    spec:
      ident: advancedProgram
      bpf2goVersion: "v0.12.0"
      goPackage: "bpf"
      outputStem: "zz_generated"
      tags:
        - "linux"
      types:
        - "event"
        - "config"
      cflags:
        - "-I./include"
        - "-D__TARGET_ARCH_x86"
```

### Multiple BPF Programs

```yaml
build:
  - name: tracepoint-bpf
    src: ./bpf/tracepoint.c
    dest: ./pkg/bpf/tracepoint
    engine: go://go-gen-bpf
    spec:
      ident: tracepoint

  - name: kprobe-bpf
    src: ./bpf/kprobe.c
    dest: ./pkg/bpf/kprobe
    engine: go://go-gen-bpf
    spec:
      ident: kprobe
```

## Implementation Details

- Executes `go run github.com/cilium/ebpf/cmd/bpf2go@{version}`
- Generates Go code that embeds compiled BPF bytecode
- Tracks the source C file as a dependency for lazy-rebuild
- Sets `DependencyDetectorEngine` to enable incremental builds

## Bpf2go Command Construction

Given the configuration, the engine constructs:

```bash
go run github.com/cilium/ebpf/cmd/bpf2go@latest \
  --go-package mypackage \
  --output-dir ./pkg/bpf \
  --output-stem zz_generated \
  --tags linux \
  myProgram \
  ./bpf/program.c \
  -- -I./include
```

## Prerequisites

- Go 1.21+ (for go run with version suffix)
- C compiler (clang recommended)
- Linux kernel headers (for BPF development)

## Error Handling

### Common Errors

| Error | Description | Resolution |
|-------|-------------|------------|
| `src is required` | Source file not specified | Provide `src` field pointing to .c file |
| `dest is required` | Destination directory not specified | Provide `dest` field for output directory |
| `spec.ident is required` | Go identifier not specified | Provide `ident` in spec for generated types |
| `src must be a file, not directory` | src points to directory | Provide path to a specific .c file |
| `bpf2go failed` | Compilation error | Check bpf2go output for C errors |

## See Also

- [BPF Code Generator Configuration Schema](schema.md)
- [cilium/ebpf Documentation](https://github.com/cilium/ebpf)
- [bpf2go Documentation](https://pkg.go.dev/github.com/cilium/ebpf/cmd/bpf2go)
