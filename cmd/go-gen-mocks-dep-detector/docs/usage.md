# go-gen-mocks-dep-detector

**Detect file dependencies for mockery mock generation.**

> "My mocks were regenerating on every build. Now forge only regenerates them when my interface files actually change."

## What problem does go-gen-mocks-dep-detector solve?

Mock generation can be slow, especially for large projects. This detector analyzes `.mockery.yaml` configuration files and resolves Go package paths to source files, enabling lazy rebuild support for `go-gen-mocks`.

## How do I use go-gen-mocks-dep-detector?

You don't invoke it directly. It's called automatically by `go-gen-mocks` after mock generation:

1. `forge build mocks` invokes `go-gen-mocks`
2. `go-gen-mocks` generates mocks using mockery
3. `go-gen-mocks` calls this detector to find dependencies
4. Dependencies are stored with the artifact
5. On subsequent builds, forge compares timestamps to decide if rebuild is needed

## What does it detect?

- **Mockery config file** - The `.mockery.yaml` configuration file itself
- **go.mod file** - The project's go.mod file
- **Interface source files** - All `.go` files (excluding `_test.go`) from packages listed in the mockery config

### Mockery Config Discovery

The detector searches for mockery configuration in order:
1. `MOCKERY_CONFIG_PATH` environment variable
2. `.mockery.yaml` / `.mockery.yml` in workDir
3. `mockery.yaml` / `mockery.yml` in workDir

## What are the limitations?

External packages referenced in `.mockery.yaml` are NOT tracked for lazy rebuild. When external interface dependencies change, use `--force`:

```bash
forge build mocks --force
```

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../MCP.md) - MCP tool documentation
- [go-gen-mocks](../../go-gen-mocks/docs/usage.md) - Mock generator documentation
