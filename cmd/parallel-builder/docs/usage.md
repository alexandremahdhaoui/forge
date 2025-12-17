# parallel-builder

**Execute multiple builders concurrently for faster builds.**

> "Building our CLI for 6 platforms used to take 3 minutes sequentially. With parallel-builder, all platforms build simultaneously and we're done in 40 seconds."

## What problem does parallel-builder solve?

Sequential builds waste time when targets are independent. parallel-builder orchestrates concurrent execution of sub-builders, dramatically reducing total build time for cross-platform builds, multiple binaries, or mixed build types.

## How do I use parallel-builder?

Add a parallel build target to `forge.yaml`:

```yaml
build:
  - name: cross-platform
    engine: go://parallel-builder
    spec:
      builders:
        - name: linux
          engine: go://go-build
          spec:
            name: myapp-linux
            src: ./cmd/myapp
            env: { GOOS: linux, GOARCH: amd64 }
        - name: darwin
          engine: go://go-build
          spec:
            name: myapp-darwin
            src: ./cmd/myapp
            env: { GOOS: darwin, GOARCH: arm64 }
```

Run the build:

```bash
forge build
```

## What configuration options are available?

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Parallel build name |
| `spec.builders` | Yes | Array of sub-builder configurations |
| `spec.builders[].name` | No | Sub-builder name |
| `spec.builders[].engine` | Yes | Engine URI for sub-builder |
| `spec.builders[].spec` | Yes | Engine-specific configuration |

## How do I build multiple binaries in parallel?

```yaml
build:
  - name: all-binaries
    engine: go://parallel-builder
    spec:
      builders:
        - name: cli
          engine: go://go-build
          spec: { name: mycli, src: ./cmd/cli }
        - name: server
          engine: go://go-build
          spec: { name: myserver, src: ./cmd/server }
        - name: worker
          engine: go://go-build
          spec: { name: myworker, src: ./cmd/worker }
```

## How do I mix different build types?

```yaml
build:
  - name: all-artifacts
    engine: go://parallel-builder
    spec:
      builders:
        - name: binary
          engine: go://go-build
          spec: { name: myapp, src: ./cmd/myapp }
        - name: mocks
          engine: go://go-gen-mocks
          spec: { name: generate-mocks }
        - name: format
          engine: go://go-format
          spec: { name: format-code, src: . }
```

## How does error handling work?

- Partial failures are reported with error count
- Combined artifact returns even with failures
- Error format: `parallel-builder: X/Y builders failed: [details]`
- All builders run to completion regardless of individual failures

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../MCP.md) - MCP tool documentation
