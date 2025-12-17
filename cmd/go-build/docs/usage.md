# go-build

**Build Go binaries with automatic git versioning and artifact tracking.**

> "I was tired of manually managing version strings and build flags across our Go projects. go-build handles all that automatically - I just point it at my source and it produces versioned binaries ready for deployment."

## What problem does go-build solve?

Building Go binaries consistently across projects requires managing version injection, build flags, and artifact tracking. go-build automates this, ensuring every binary is versioned with git commit SHA and tracked in the artifact store.

## How do I use go-build?

Add a build target to `forge.yaml`:

```yaml
build:
  - name: myapp
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
```

Run the build:

```bash
forge build
```

## What configuration options are available?

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Binary name (output filename) |
| `src` | Yes | Source directory containing main package |
| `dest` | No | Output directory (default: current directory) |
| `spec.args` | No | Additional go build arguments |
| `spec.env` | No | Environment variables for the build |

## How do I cross-compile?

Use environment variables in the spec:

```yaml
build:
  - name: myapp-linux-amd64
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
    spec:
      env:
        GOOS: linux
        GOARCH: amd64
        CGO_ENABLED: "0"
```

## How do I add custom build flags?

Use the `args` field:

```yaml
build:
  - name: myapp
    src: ./cmd/myapp
    engine: go://go-build
    spec:
      args:
        - "-tags=netgo"
        - "-ldflags=-w -s"
      env:
        CGO_ENABLED: "0"
```

## How does it work?

The engine runs `go build` with these defaults:
- Sets `CGO_ENABLED=0` (overridable via `env`)
- Injects git commit SHA via ldflags
- Outputs binary to `{dest}/{name}`
- Stores artifact metadata in the artifact store

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../MCP.md) - MCP tool documentation
