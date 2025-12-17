# go-gen-bpf

**Generate Go bindings from BPF C source files with automatic dependency tracking.**

> "I was manually running bpf2go and tracking which BPF programs needed rebuilding. Now forge handles it automatically - I just define the source and destination, and go-gen-bpf does the rest with proper incremental builds."

## What problem does go-gen-bpf solve?

Building eBPF programs requires compiling C code and generating Go bindings via bpf2go. go-gen-bpf integrates this into forge's build system with dependency tracking for incremental rebuilds.

## How do I use go-gen-bpf?

Add a build target to `forge.yaml`:

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

This generates Go code in `./pkg/bpf` that embeds the compiled BPF bytecode.

## What configuration options are available?

| Option | Required | Description |
|--------|----------|-------------|
| `ident` | Yes | Go identifier prefix for generated types (e.g., `myProgram` generates `myProgramObjects`) |
| `bpf2goVersion` | No | Version of bpf2go to use (default: `latest`) |
| `goPackage` | No | Package name for generated files |
| `outputStem` | No | Prefix for generated filenames (default: based on ident) |
| `tags` | No | Build tags to include (e.g., `["linux"]`) |
| `types` | No | BPF types to export to Go |
| `cflags` | No | Additional C compiler flags (e.g., `["-I./include"]`) |
| `cc` | No | C compiler to use (default: clang) |

### Example with all options

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
      tags: ["linux"]
      types: ["event", "config"]
      cflags: ["-I./include", "-D__TARGET_ARCH_x86"]
```

## What are the prerequisites?

- Go 1.21+ (for `go run` with version suffix)
- C compiler (clang recommended)
- Linux kernel headers (for BPF development)

## What errors might I encounter?

| Error | Cause | Fix |
|-------|-------|-----|
| `src is required` | Missing source file | Add `src` pointing to .c file |
| `spec.ident is required` | Missing identifier | Add `ident` in spec |
| `src must be a file, not directory` | src is a directory | Point to specific .c file |
| `bpf2go failed` | C compilation error | Check bpf2go output for details |

## What's next?

- [schema.md](schema.md) - Configuration reference
- [cilium/ebpf](https://github.com/cilium/ebpf) - eBPF library documentation
- [bpf2go](https://pkg.go.dev/github.com/cilium/ebpf/cmd/bpf2go) - Code generator documentation
