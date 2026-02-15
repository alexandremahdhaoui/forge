# Contributing to Forge

**Everything you need to go from clone to merged PR.**

## Quick Start

```bash
git clone https://github.com/alexandremahdhaoui/forge.git
cd forge
go run ./cmd/forge build          # Build all 28 engines
go run ./cmd/forge test-all       # Build + run all 7 test stages
```

Requires Go 1.25.0+, Docker (for integration tests), and Kind (for test environments).

## How do I structure commits?

Each commit uses an emoji prefix and a structured body.

**Emoji conventions:**

| Emoji | Meaning |
|---|---|
| `✨` | New feature (feat:) |
| `🐛` | Bug fix (fix:) |
| `📖` | Documentation (docs:) |
| `🌱` | Misc (chore:, test:, and others) |
| `⚠` | Breaking changes -- never use without maintainer approval |

**Commit body format:**

```
✨ Short imperative summary (50 chars or less)

Why: Explain the motivation. What problem exists?

How: Describe the approach. What strategy did you choose?

What:
- pkg/foo/bar.go: description of change
- cmd/baz/main.go: description of change

How changes were verified:
- Unit tests for new logic (go test)
- forge test-all: all stages passed

Signed-off-by: Your Name <your@email.com>
```

Every commit requires `Signed-off-by`. Use `git commit -s` to add it automatically.

## How do I submit a pull request?

1. Create a feature branch from `main`.
2. Make focused, atomic commits following the format above.
3. Run `go run ./cmd/forge test-all` and confirm all stages pass.
4. Open a PR against `main`. Describe what changed and why.
5. PRs that break `test-all` will not be merged.

## How do I run tests?

Forge defines 7 test stages in `forge.yaml`, executed in order with fail-fast behavior:

```bash
go run ./cmd/forge test-all                  # Run all stages (build + test)

# Individual stages
go run ./cmd/forge test lint-tags run        # Verify build tags on test files
go run ./cmd/forge test lint-license run     # Verify Apache 2.0 license headers
go run ./cmd/forge test lint run             # Run golangci-lint (41 linters)
go run ./cmd/forge test unit run             # Run unit tests
go run ./cmd/forge test integration run      # Run integration tests (creates Kind cluster)
go run ./cmd/forge test e2e run              # Run end-to-end tests
go run ./cmd/forge test e2e-stub run         # Run e2e tests with stub testenv
```

Integration tests create real Kind clusters with TLS registries and Helm charts. They take longer and require Docker. Unit and lint stages have no external dependencies.

**Test environment management:**

```bash
go run ./cmd/forge test integration create       # Create persistent environment
go run ./cmd/forge test integration list          # List environments
go run ./cmd/forge test integration get <id>      # Get environment details
go run ./cmd/forge test integration delete <id>   # Teardown environment
```

## How is the project structured?

```
forge/
├── cmd/                  # 28 CLI tools, each an MCP server
│   ├── forge/            # Main orchestrator
│   ├── forge-dev/        # Engine scaffolding generator
│   ├── forge-e2e/        # End-to-end test runner
│   ├── ci-orchestrator/  # CI orchestration (planned)
│   ├── go-build/         # Go binary builder
│   ├── container-build/  # Container image builder (Kaniko)
│   ├── ...               # See tool catalog below
│   └── */MCP.md          # Per-tool MCP documentation
├── pkg/                  # Public packages (importable)
├── internal/             # Internal packages
├── docs/                 # User, developer, and architecture docs
├── forge.yaml            # Build and test configuration
└── DESIGN.md             # System design document
```

## What does each CLI tool do?

All 28 tools are MCP servers speaking JSON-RPC 2.0 over stdio.

**Orchestration (4):**

| Tool | Purpose |
|---|---|
| `forge` | Main CLI orchestrator. Reads `forge.yaml`, resolves engines, runs builds and tests. |
| `forge-dev` | Engine scaffolding code generator. Creates new engines from OpenAPI specs. |
| `forge-e2e` | End-to-end test runner for Forge itself. |
| `ci-orchestrator` | CI pipeline orchestration (planned). |

**Build Engines (4):**

| Tool | Purpose |
|---|---|
| `go-build` | Builds Go binaries with git version injection via ldflags. |
| `container-build` | Builds container images using Kaniko. |
| `generic-builder` | Wraps any shell command as a build step. |
| `parallel-builder` | Runs multiple build engines concurrently. |

**Test Environment (5):**

| Tool | Purpose |
|---|---|
| `testenv` | Testenv chain orchestrator. Runs sub-engines sequentially, propagating env vars. |
| `testenv-kind` | Creates and manages Kind (Kubernetes in Docker) clusters. |
| `testenv-lcr` | Deploys TLS-enabled local container registries into Kind clusters. |
| `testenv-helm-install` | Installs Helm charts from helm-repo, git, OCI, or S3 sources. |
| `testenv-stub` | Stub testenv for fast testing without real resources. |

**Test Runners (4):**

| Tool | Purpose |
|---|---|
| `go-test` | Runs Go tests with coverage reporting. |
| `go-lint-tags` | Verifies all test files have build tags. |
| `generic-test-runner` | Wraps any shell command as a test. |
| `parallel-test-runner` | Runs multiple test runners concurrently. |

**Test Management (1):**

| Tool | Purpose |
|---|---|
| `test-report` | Aggregates and queries test reports. |

**Code Quality (3):**

| Tool | Purpose |
|---|---|
| `go-format` | Formats Go code with gofmt and goimports. |
| `go-lint` | Runs golangci-lint. |
| `go-lint-licenses` | Verifies Apache 2.0 license headers on all Go files. |

**Code Generation (4):**

| Tool | Purpose |
|---|---|
| `go-gen-mocks` | Generates mock implementations for testing. |
| `go-gen-openapi` | Generates Go client/server code from OpenAPI specs. |
| `go-gen-protobuf` | Compiles Protocol Buffer definitions. |
| `go-gen-bpf` | Generates BPF bytecode. |

**Dependency Detection (3):**

| Tool | Purpose |
|---|---|
| `go-dependency-detector` | Scans Go AST for file and package dependencies (lazy rebuild). |
| `go-gen-mocks-dep-detector` | Detects dependencies for mock generation targets. |
| `go-gen-openapi-dep-detector` | Detects dependencies for OpenAPI generation targets. |

## What does each package do?

**Public packages (`pkg/`):**

| Package | Purpose |
|---|---|
| `enginecli` | Engine CLI framework. Parses flags, starts MCP server. |
| `enginedocs` | Generates MCP documentation from engine schemas. |
| `engineframework` | Type-safe engine framework. Schema validation, handler registration. |
| `engineversion` | Engine version management and reporting. |
| `eventualconfig` | Async configuration with eventual consistency for testenv chains. |
| `flaterrors` | Flattens nested error trees into flat lists. |
| `forge` | Core types: BuildSpec, TestSpec, TestReport, artifact store. |
| `mcpserver` | MCP server framework. JSON-RPC 2.0 message handling. |
| `mcptypes` | MCP protocol types: BuildInput, RunInput, CreateInput. |
| `mcputil` | MCP utilities: validation, batch handling, response formatting. |
| `portalloc` | Dynamic port allocation with flock-based persistence. |
| `templateutil` | Template utilities for Go text/template expansion. |
| `testenvutil` | Test environment utilities shared across testenv sub-engines. |

**Internal packages (`internal/`):**

| Package | Purpose |
|---|---|
| `cmdutil` | Command execution utilities (exec.Command wrappers). |
| `engineresolver` | Resolves engine URIs (`go://`, `alias://`) to MCP server binaries. |
| `enginetest` | Test helpers for engine development. |
| `forgepath` | Forge path resolution (config, artifact store, build output). |
| `gitutil` | Git operations (SHA, version, dirty state). |
| `integration` | Integration test utilities (Kind, registry helpers). |
| `mcpcaller` | MCP client caller. Sends JSON-RPC 2.0 requests to engines. |
| `orchestrate` | Build and test orchestration logic. |
| `testutil` | General test utilities (temp dirs, assertions). |
| `util` | General utilities (string helpers, env manipulation). |

## How do I create a new engine?

Use `forge-dev` to scaffold the engine from an OpenAPI spec:

```bash
# 1. Create directory structure
mkdir -p cmd/my-engine

# 2. Create OpenAPI spec at cmd/my-engine/openapi.yaml

# 3. Generate scaffolding
go run ./cmd/forge-dev generate --src ./cmd/my-engine

# 4. Implement the handler in cmd/my-engine/handler.go

# 5. Add to forge.yaml build section
#    - name: my-engine
#      src: ./cmd/my-engine
#      dest: ./build/bin
#      engine: go://go-build

# 6. Reference in forge.yaml test or build section
#    engine: go://my-engine
```

See [Engine Development](./docs/dev/getting-started.md) and [forge-dev](./docs/dev/forge-dev.md) for complete guides.

## What conventions must I follow?

**Build tags on all test files.** Every `_test.go` file requires a build tag as the first line:

```go
//go:build unit
```

Valid tags: `unit`, `integration`, `functional`, `e2e`. The `lint-tags` stage enforces this.

**License headers on all Go files.** Every `.go` file starts with:

```go
// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// ...
```

The `lint-license` stage enforces this. Test files place the build tag before the license header.

**Generated files.** Files produced by `forge-dev` (in `cmd/*/`) are regenerated on build. Do not edit generated code manually -- modify the OpenAPI spec and regenerate.

**Engine URIs.** Reference engines as `go://engine-name` or `alias://alias-name` in `forge.yaml`. The `engine:` field names the engine; `builder:` is not a valid field.

**Linting.** The project runs golangci-lint with 41 linters enabled. Run `go run ./cmd/forge test lint run` before pushing.

## Documentation

| Audience | Documents |
|---|---|
| Users | [getting-started](./docs/user/getting-started.md), [forge-cli](./docs/user/forge-cli.md), [forge-yaml-schema](./docs/user/forge-yaml-schema.md), [testing](./docs/user/testing.md) |
| Engine developers | [getting-started](./docs/dev/getting-started.md), [forge-dev](./docs/dev/forge-dev.md), [creating-build-engine](./docs/dev/creating-build-engine.md) |
| Architecture | [DESIGN.md](./DESIGN.md), [testenv-architecture](./docs/architecture/testenv-architecture.md) |
