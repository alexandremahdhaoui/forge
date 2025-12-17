# go-format

**Format Go code with gofumpt for consistent style.**

> "Our team had constant debates about code formatting. go-format with gofumpt's stricter rules settled everything - now every PR has consistent style without manual review."

## What problem does go-format solve?

Standard gofmt leaves room for style variations. go-format uses gofumpt, which applies stricter rules for import grouping, empty lines, and slice expressions - ensuring truly consistent code style across your codebase.

## How do I use go-format?

Add a build target to `forge.yaml`:

```yaml
build:
  - name: format-code
    src: .
    engine: go://go-format
```

Run the formatter:

```bash
forge build
```

## What configuration options are available?

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Build step name |
| `src` | No | Directory to format (default: current directory) |

## How do I format specific directories?

```yaml
build:
  - name: format-pkg
    src: ./pkg
    engine: go://go-format

  - name: format-cmd
    src: ./cmd
    engine: go://go-format
```

## How do I format directories in parallel?

```yaml
build:
  - name: format-all
    engine: go://parallel-builder
    spec:
      builders:
        - name: cmd
          engine: go://go-format
          spec: { name: cmd, src: ./cmd }
        - name: internal
          engine: go://go-format
          spec: { name: internal, src: ./internal }
```

## What does gofumpt do differently than gofmt?

- No empty lines at start/end of function bodies
- No empty lines around lone statements in blocks
- Imports sorted and grouped properly
- Simplified slice expressions where possible

## What environment variables are available?

| Variable | Default | Description |
|----------|---------|-------------|
| `GOFUMPT_VERSION` | `v0.6.0` | Version of gofumpt to use |

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../MCP.md) - MCP tool documentation
- [gofumpt docs](https://github.com/mvdan/gofumpt) - Upstream documentation
