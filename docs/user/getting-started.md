# Getting Started with Forge

**Build and test your Go project in under 5 minutes.**

> "I had my first build running in 2 minutes. No Makefile, just a simple YAML file."

## How do I install forge?

```bash
go install github.com/alexandremahdhaoui/forge/cmd/forge@latest
forge version
```

## How do I create my first forge.yaml?

```yaml
name: my-project
artifactStorePath: .forge/artifacts.yaml

build:
  - name: my-app
    src: ./cmd/my-app
    dest: ./build/bin
    engine: go://go-build

test:
  - name: unit
    runner: go://go-test
```

## How do I build and test?

```bash
forge build          # Build all artifacts
forge test unit run  # Run unit tests
forge test-all       # Build + run all tests (fail-fast)
```

## What's next?

- [CLI Reference](./forge-cli.md) - All commands
- [Schema Reference](./forge-yaml-schema.md) - forge.yaml options
- [Testing](./testing.md) - Tests and environments
