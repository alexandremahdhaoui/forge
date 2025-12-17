# go-gen-mocks

**Generate Go mock implementations using mockery.**

> "Writing mocks manually was tedious and error-prone. go-gen-mocks regenerates all our mocks automatically whenever interfaces change - now our tests always have up-to-date mocks."

## What problem does go-gen-mocks solve?

Unit testing with dependency injection requires mock implementations. go-gen-mocks automates mock generation using mockery, keeping mocks synchronized with interface definitions and supporting lazy rebuild.

## How do I use go-gen-mocks?

Add a build target to `forge.yaml`:

```yaml
build:
  - name: go-gen-mocks
    engine: go://go-gen-mocks
```

Run the generator:

```bash
forge build
```

## What configuration options are available?

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Build step name |
| `spec.rootDir` | No | Project root directory |

## How do I configure mockery?

Create `.mockery.yaml` in your project root:

```yaml
with-expecter: true
dir: "./internal/util/mocks"
packages:
  github.com/myorg/myproject/pkg/interfaces:
    interfaces:
      MyInterface:
```

## How do I use a custom mocks directory?

```bash
MOCKS_DIR=./test/mocks forge build
```

## How do I combine with other build steps?

```yaml
build:
  - name: go-gen-mocks
    engine: go://go-gen-mocks

  - name: myapp
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
```

## What environment variables are available?

| Variable | Default | Description |
|----------|---------|-------------|
| `MOCKERY_VERSION` | `v3.5.5` | Version of mockery to use |
| `MOCKS_DIR` | `./internal/util/mocks` | Directory to clean and generate mocks |

## How does it work?

- Cleans existing mocks directory before generating
- Runs `go run github.com/vektra/mockery/v3@{version}`
- Discovers interfaces via mockery configuration
- Supports lazy rebuild via dependency detection

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../MCP.md) - MCP tool documentation
- [mockery docs](https://vektra.github.io/mockery/) - Upstream documentation
