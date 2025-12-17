# container-build

**Build container images with multiple backend engines and automatic versioning.**

> "Our CI pipeline needed to support both Docker and rootless Kaniko builds. container-build lets us switch backends with an environment variable while keeping consistent versioning and artifact tracking."

## What problem does container-build solve?

Container image builds need to work across different environments - some have Docker daemons, others require rootless builds. container-build provides a unified interface for docker, kaniko, and podman backends while handling git-based versioning and artifact tracking automatically.

## How do I use container-build?

Add a build target to `forge.yaml`:

```yaml
build:
  - name: my-app
    src: ./Containerfile
    engine: go://container-build
```

Run the build:

```bash
CONTAINER_BUILD_ENGINE=docker forge build
```

## What backend engines are available?

| Engine | Environment Variable | Characteristics |
|--------|---------------------|-----------------|
| docker | `CONTAINER_BUILD_ENGINE=docker` | Fast, requires Docker daemon |
| kaniko | `CONTAINER_BUILD_ENGINE=kaniko` | Rootless, secure, runs in container |
| podman | `CONTAINER_BUILD_ENGINE=podman` | Rootless, requires Podman |

## What configuration options are available?

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Image name |
| `src` | Yes | Path to Containerfile/Dockerfile |
| `dest` | No | Registry destination (for push) |
| `spec.dependsOn` | No | Dependency detection configuration |

## How do I pass build arguments?

Use the `BUILD_ARGS` environment variable:

```bash
BUILD_ARGS="VERSION=1.0.0,COMMIT=abc123" CONTAINER_BUILD_ENGINE=docker forge build
```

## How do I track dependencies for lazy rebuild?

```yaml
build:
  - name: my-app
    src: ./Containerfile
    engine: go://container-build
    spec:
      dependsOn:
        - engine: go://go-dependency-detector
          spec:
            filePath: ./cmd/myapp/main.go
            funcName: main
```

## How does it work?

- Tags images with `<name>:<git-sha>` and `<name>:latest`
- Stores artifact metadata in artifact store
- Kaniko exports to tar, then loads into container engine
- Docker/Podman use native builds for faster execution

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../MCP.md) - MCP tool documentation
