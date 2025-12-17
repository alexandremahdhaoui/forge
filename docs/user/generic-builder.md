# generic-builder

**Wrap any CLI command as a forge build engine.**

> "I wanted to use protoc and prettier in my build. With generic-builder, I just added a few lines to forge.yaml - no Go code, no custom plugins."

## What problem does generic-builder solve?

You have CLI tools (formatters, generators, compilers) that work perfectly from the command line. You want them in your forge build without writing custom Go code. Generic-builder lets you wrap any command as a build engine using YAML configuration.

## How do I configure generic-builder?

Define an engine alias in forge.yaml, then reference it in your build specs:

```yaml
engines:
  - alias: protoc
    type: builder
    builder:
      - engine: go://generic-builder
        spec:
          command: "protoc"
          args: ["--go_out=.", "api/service.proto"]

build:
  - name: generate-proto
    src: ./api
    engine: alias://protoc
```

Run with: `forge build generate-proto`

## What configuration options are available?

| Option | Required | Description |
|--------|----------|-------------|
| `command` | Yes | Executable to run (in PATH or full path) |
| `args` | No | Array of command arguments |
| `env` | No | Environment variables as key-value map |
| `envFile` | No | Path to .envrc file with environment variables |
| `workDir` | No | Working directory for execution |

**Environment precedence** (highest to lowest): `env` > `envFile` > system environment

## When should I use a built-in engine instead?

Use built-in engines when available:
- **Go binaries**: `go://go-build`
- **Containers**: `go://container-build`
- **Go formatting**: `go://go-format`

Use generic-builder when no built-in exists for your tool.

## Quick examples

**Code formatter:**
```yaml
engines:
  - alias: prettier
    type: builder
    builder:
      - engine: go://generic-builder
        spec:
          command: "prettier"
          args: ["--write", "src/**/*.ts"]
          workDir: "./frontend"

build:
  - name: format-frontend
    src: ./frontend
    engine: alias://prettier
```

**Docker build:**
```yaml
engines:
  - alias: docker-build
    type: builder
    builder:
      - engine: go://generic-builder
        spec:
          command: "docker"
          args: ["build", "-t", "myapp:latest", "."]
          env:
            DOCKER_BUILDKIT: "1"

build:
  - name: container
    src: ./Dockerfile
    engine: alias://docker-build
```

## How do I debug a failing command?

1. Extract the command and run it manually
2. Check exit code: `echo $?` (0 = success)
3. Verify paths are relative to forge.yaml location

## What's next?

- [generic-test-runner](./generic-test-runner.md) - Wrap CLI tools as test runners
- [Schema Reference](./forge-yaml-schema.md) - Full forge.yaml options
