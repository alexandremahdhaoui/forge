# Workspace Development

**Run forge across multiple repos from a single Go workspace.**

> "I have forge, shaper, and testenv-vm in one Go workspace. Now I can build and test any repo
> without leaving the workspace root or switching directories."

## What problem does this solve?

When developing multiple Go modules that depend on each other, you typically use a
[Go workspace](https://go.dev/doc/tutorial/workspaces) (`go.work`). But forge resolves all
paths relative to its working directory -- so running forge from the workspace root breaks
config loading, build paths, and engine execution.

The `--config` flag with directory components solves this. Forge automatically changes to the
config file's directory before doing any work, so all relative paths in `forge.yaml` resolve
correctly.

## How do I set up a Go workspace with forge?

```
my-workspace/
  go.work          # references ./forge, ./my-app, ./my-lib
  forge/
    cmd/forge/
    forge.yaml
  my-app/
    forge.yaml
  my-lib/
    forge.yaml
```

Your `go.work` references each module:

```
use (
    ./forge
    ./my-app
    ./my-lib
)
```

## How do I run forge from the workspace root?

Point `--config` at the target repo's `forge.yaml`:

```bash
# Build my-app from workspace root
go run ./forge/cmd/forge --config=./my-app/forge.yaml build

# Run my-lib's unit tests from workspace root
go run ./forge/cmd/forge --config=./my-lib/forge.yaml test unit run

# List forge's own targets
go run ./forge/cmd/forge --config=./forge/forge.yaml list
```

Forge prints the directory change to stderr so you know where it's running:

```
forge: changed working directory to my-app
```

## How do I run forge from a sibling repo?

When your working directory is already inside a repo, forge finds `forge.yaml` in the
current directory -- no `--config` needed:

```bash
cd my-app
go run ../forge/cmd/forge list        # uses my-app/forge.yaml
go run ../forge/cmd/forge build       # builds my-app
go run ../forge/cmd/forge test-all    # tests my-app
```

## How does --config work?

Both syntaxes are supported:

```bash
go run ./forge/cmd/forge --config=./my-app/forge.yaml build
go run ./forge/cmd/forge --config ./my-app/forge.yaml build
```

When `--config` includes a directory (anything beyond a bare filename), forge:

1. Changes the working directory to the config file's directory
2. Strips the path, keeping only the filename (`forge.yaml`)
3. Proceeds normally -- all relative paths in `forge.yaml` resolve from the correct directory

When `--config` is a bare filename or omitted, forge stays in the current directory.

## What about .envrc files?

Each repo typically has its own `.envrc` (gitignored, copied from `.envrc.example`) that sets
environment variables like `CONTAINER_BUILD_ENGINE` and `FORGE_RUN_LOCAL_ENABLED`. Since forge
changes to the target repo's directory first, it sources that repo's `.envrc` automatically.

Make sure each repo has its `.envrc` populated:

```bash
cp my-app/.envrc.example my-app/.envrc
# Edit my-app/.envrc with your local values
```

## What's next?

- [Getting Started](./getting-started.md) - First 5 minutes with forge
- [CLI Reference](./forge-cli.md) - All commands and flags
- [forge.yaml Schema](./forge-yaml-schema.md) - Configuration reference
