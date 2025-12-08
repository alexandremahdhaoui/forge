# BPF Code Generator Configuration Schema

## Overview

This document describes the configuration options for `go-gen-bpf` in `forge.yaml`. The go-gen-bpf engine generates Go code from BPF C source files using bpf2go.

## Basic Configuration

```yaml
build:
  - name: my-bpf-program
    src: ./bpf/program.c
    dest: ./pkg/bpf
    engine: go://go-gen-bpf
    spec:
      ident: myProgram
```

## Configuration Options

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Artifact identifier for the artifact store |
| `src` | string | Path to the BPF C source file (must be a file, not directory) |
| `dest` | string | Output directory for generated Go files |
| `engine` | string | Must be `go://go-gen-bpf` to use this generator |
| `spec.ident` | string | Go identifier for generated types (e.g., `myProgram` generates `myProgramObjects`) |

### Spec Options

The `spec` field contains engine-specific configuration:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `ident` | string | *required* | Go identifier for generated types |
| `bpf2goVersion` | string | `latest` | Version of bpf2go tool to use |
| `goPackage` | string | basename of dest | Go package name for generated code |
| `outputStem` | string | `zz_generated` | Filename prefix for generated files |
| `tags` | string[] | `["linux"]` | Build tags for generated files |
| `types` | string[] | all | Specific types to generate (empty = all) |
| `cflags` | string[] | - | C compiler flags for BPF compilation |
| `cc` | string | - | C compiler binary (uses bpf2go default if empty) |

## Examples

### Minimal Configuration

```yaml
build:
  - name: simple-bpf
    src: ./bpf/simple.c
    dest: ./pkg/bpf
    engine: go://go-gen-bpf
    spec:
      ident: simple
```

### Full Configuration

```yaml
build:
  - name: advanced-bpf
    src: ./bpf/advanced.c
    dest: ./pkg/bpf/advanced
    engine: go://go-gen-bpf
    spec:
      ident: advanced
      bpf2goVersion: "v0.12.0"
      goPackage: "advanced"
      outputStem: "zz_generated"
      tags:
        - "linux"
        - "amd64"
      types:
        - "event"
        - "config"
      cflags:
        - "-I./include"
        - "-I./vendor/libbpf/include"
        - "-D__TARGET_ARCH_x86"
      cc: "clang-15"
```

### With Include Paths

```yaml
build:
  - name: my-bpf
    src: ./bpf/program.c
    dest: ./pkg/bpf
    engine: go://go-gen-bpf
    spec:
      ident: myProgram
      cflags:
        - "-I./bpf/include"
        - "-I/usr/include/bpf"
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
      goPackage: "tracepoint"

  - name: kprobe-bpf
    src: ./bpf/kprobe.c
    dest: ./pkg/bpf/kprobe
    engine: go://go-gen-bpf
    spec:
      ident: kprobe
      goPackage: "kprobe"
```

## Generated Output

Each build creates an artifact in the artifact store:

```yaml
artifacts:
  - name: my-bpf-program
    type: bpf
    location: ./pkg/bpf
    timestamp: "2024-01-15T10:30:00Z"
    dependencies:
      - type: file
        filePath: /absolute/path/to/bpf/program.c
        timestamp: "2024-01-14T09:00:00Z"
    dependencyDetectorEngine: go://go-gen-bpf
```

## Generated Files

bpf2go generates several files in the destination directory:

- `{outputStem}_{ident}_bpfel.go` - Little-endian BPF bytecode
- `{outputStem}_{ident}_bpfeb.go` - Big-endian BPF bytecode
- `{outputStem}_{ident}_bpfel.o` - Compiled little-endian object
- `{outputStem}_{ident}_bpfeb.o` - Compiled big-endian object

With default settings (`outputStem: zz_generated`, `ident: myProgram`):
- `zz_generated_myProgram_bpfel.go`
- `zz_generated_myProgram_bpfeb.go`

## Bpf2go Arguments

Arguments are passed to bpf2go in this order:

1. `--go-package` - Go package name
2. `--output-dir` - Output directory
3. `--output-stem` - Filename prefix
4. `--tags` - Build tags (comma-separated)
5. `--type` - Specific types (one flag per type)
6. `{ident}` - Go identifier
7. `{src}` - Source file path
8. `-- {cflags...}` - C compiler flags (after separator)

## Environment Variables

| Variable | Description |
|----------|-------------|
| `BPF2GO_CC` | C compiler to use (set when `cc` is specified) |

## Default Behavior

When minimal spec is provided, go-gen-bpf uses these defaults:

- `bpf2goVersion: "latest"`
- `goPackage`: basename of dest directory
- `outputStem: "zz_generated"`
- `tags: ["linux"]`
- All types generated (no filtering)

## Error Cases

| Error | Cause | Solution |
|-------|-------|----------|
| `src is required` | Missing src field | Add `src` field pointing to .c file |
| `dest is required` | Missing dest field | Add `dest` field for output directory |
| `spec.ident is required` | Missing ident | Add `ident` to spec |
| `src must be a file` | src is a directory | Point to specific .c file |
| `bpf2go failed` | Compilation error | Check C source for errors |

## See Also

- [BPF Code Generator Usage Guide](usage.md)
- [cilium/ebpf bpf2go](https://github.com/cilium/ebpf/tree/main/cmd/bpf2go)
