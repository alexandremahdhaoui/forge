# Forge

**AI-native build orchestration that replaces Makefiles with declarative YAML and makes every tool an MCP server.**

## What problem does Forge solve?

Teams build Go services with Makefiles, shell scripts, and ad-hoc tooling. These scripts break silently, produce unreproducible builds, and are invisible to AI coding agents that communicate through structured protocols. Test environment setup -- Kind clusters, TLS registries, Helm charts -- requires manual steps that differ per developer machine. Forge replaces this with a single `forge.yaml` configuration file where every component is an MCP server speaking JSON-RPC 2.0 over stdio. AI agents invoke builds, create test environments, and run test suites directly through the MCP protocol.

## Quick Start

```bash
go install github.com/alexandremahdhaoui/forge/cmd/forge@latest
```

```yaml
# forge.yaml
name: my-project
artifactStorePath: .forge/artifact-store.yaml

build:
  - name: my-app
    src: ./cmd/my-app
    dest: ./build/bin
    engine: go://go-build

test:
  - name: unit
    runner: go://go-test
```

```bash
forge build       # Build all artifacts
forge test-all    # Build + run all test stages (fail-fast)
```

## How does it work?

```
+-------------------+
| AI Agent / User   |
+---------+---------+
          |
     forge.yaml
          |
+---------+---------+
|    forge CLI      |  MCP server + orchestrator
|                   |  Reads config, resolves engines,
|                   |  tracks artifacts, runs tests
+---------+---------+
          | MCP over stdio (JSON-RPC 2.0)
          |
  +-------+--------+--------------+
  |                |              |
+-+--------+ +-----+------+ +----+--------+
| Build    | | Test       | | TestEnv     |
| Engines  | | Runners    | | Managers    |
|          | |            | |             |
| go-build | | go-test    | | testenv     |
| container| | go-lint    | |  +--kind    |
| generic  | | generic    | |  +--lcr     |
| parallel | | parallel   | |  +--helm    |
+----------+ +------------+ +-------------+
```

Forge starts each engine as a child process communicating over stdio. The `parallel-builder` and `parallel-test-runner` engines wrap multiple sub-engines for concurrent execution. Engine URIs (`go://engine-name`, `alias://custom-name`) resolve to MCP servers at runtime. See [DESIGN.md](./DESIGN.md) for the full architecture.

## Table of Contents

- [How do I configure Forge?](#how-do-i-configure-forge)
- [How do I build and test?](#how-do-i-build-and-test)
- [What tools are included?](#what-tools-are-included)
- [How do I extend Forge?](#how-do-i-extend-forge)
- [FAQ](#faq)
- [Documentation](#documentation)

## How do I configure Forge?

```yaml
name: my-project
artifactStorePath: .forge/artifact-store.yaml

build:
  - name: my-app            # Artifact name
    src: ./cmd/my-app        # Source directory
    dest: ./build/bin        # Output directory
    engine: go://go-build    # Engine URI

test:
  - name: unit
    runner: go://go-test     # Test runner engine

  - name: integration
    runner: go://go-test
    testenv: alias://setup-integration  # Testenv chain alias

engines:
  - alias: setup-integration
    type: testenv
    testenv:                 # Testenv chain: engines run sequentially,
      - engine: go://testenv-kind      # each propagating env vars to the next
      - engine: go://testenv-lcr
        spec:
          enabled: true
      - engine: go://testenv-helm-install
        spec:
          charts:
            - name: my-chart
              sourceType: helm-repo
              url: https://charts.example.com
              chartName: my-chart
              namespace: default
```

See [forge-yaml-schema.md](./docs/user/forge-yaml-schema.md) for the complete field reference.

## How do I build and test?

```bash
# Build
forge build                          # Build all artifacts
forge build my-app                   # Build one artifact
forge build --force                  # Force rebuild (skip lazy rebuild)

# Test
forge test-all                       # Build + run all test stages
forge test unit run                  # Run one test stage
forge test integration run           # Run with auto-created environment

# Test environment management
forge test integration create        # Create environment
forge test integration list          # List environments
forge test integration get <id>      # Get environment details
forge test integration delete <id>   # Delete environment
```

See [forge-cli.md](./docs/user/forge-cli.md) for the full command reference.

## What tools are included?

Forge ships 28 CLI tools, all implemented as MCP servers.

| Category | Count | Tools | Description |
|---|---|---|---|
| Orchestration | 4 | forge, forge-dev, forge-e2e, ci-orchestrator | CLI orchestrator, engine scaffolding, e2e testing, CI (planned) |
| Build Engines | 4 | go-build, container-build, generic-builder, parallel-builder | Go binaries, container images, arbitrary commands, parallel builds |
| Dependency Detection | 3 | go-dependency-detector, go-gen-mocks-dep-detector, go-gen-openapi-dep-detector | Track file and package dependencies for lazy rebuild |
| Test Environment | 5 | testenv, testenv-kind, testenv-lcr, testenv-helm-install, testenv-stub | Orchestrate Kind clusters, TLS registries, Helm charts, stubs |
| Test Runners | 4 | go-test, go-lint-tags, generic-test-runner, parallel-test-runner | Go tests, build tag verification, arbitrary commands, parallel runs |
| Test Management | 1 | test-report | Aggregate and query test reports |
| Code Quality | 3 | go-format, go-lint, go-lint-licenses | Format code, lint, verify license headers |
| Code Generation | 4 | go-gen-mocks, go-gen-openapi, go-gen-protobuf, go-gen-bpf | Generate mocks, OpenAPI clients, protobuf, BPF code |

## How do I extend Forge?

All engines implement JSON-RPC 2.0 over stdio. To create a new engine:

1. Define the engine's MCP tool schema (OpenAPI spec)
2. Run `forge-dev` to generate the server scaffolding
3. Implement the tool handler
4. Reference the engine in `forge.yaml` with a `go://` URI

See [Engine Development](./docs/dev/getting-started.md) and [forge-dev](./docs/dev/forge-dev.md) for step-by-step guides.

## FAQ

**Does Forge only work with Go?**
No. `generic-builder` wraps any shell command as a build step. `generic-test-runner` wraps any command as a test. Native engines exist for Go, but the architecture is language-agnostic.

**How does lazy rebuild work?**
`go-dependency-detector` scans Go AST to record file paths, modification timestamps, and `go.mod` package versions. On subsequent builds, Forge compares current state against stored dependencies and skips unchanged artifacts. See [lazy-rebuild.md](./docs/user/lazy-rebuild.md).

**Can AI agents use Forge directly?**
Yes. Run `forge --mcp` to start Forge as an MCP server. All 28 engines speak MCP natively. AI agents invoke builds, tests, and environment operations through JSON-RPC 2.0 without wrapper scripts.

**How do test environments work?**
Testenv sub-engines compose into sequential chains. Each sub-engine (Kind cluster, TLS registry, Helm charts) propagates environment variables to the next via `envPropagation` config. Template expansion (`{{.Env.KUBECONFIG}}`) enables dynamic configuration. See [testing.md](./docs/user/testing.md).

**How do I run Forge from a Go workspace?**
Use the `--config` flag to point at the target repo's `forge.yaml`. See [workspace-development.md](./docs/user/workspace-development.md).

**What is the test stage execution order?**
`forge test-all` runs stages in the order defined in `forge.yaml` with fail-fast behavior: lint-tags, lint-license, lint, unit, integration, e2e, e2e-stub. Each stage creates and tears down its own test environment if one is configured.

## Documentation

**User guides:** [getting-started](./docs/user/getting-started.md) | [forge-cli](./docs/user/forge-cli.md) | [forge-yaml-schema](./docs/user/forge-yaml-schema.md) | [testing](./docs/user/testing.md) | [generic-builder](./docs/user/generic-builder.md) | [generic-test-runner](./docs/user/generic-test-runner.md) | [lazy-rebuild](./docs/user/lazy-rebuild.md) | [migrating-from-makefile](./docs/user/migrating-from-makefile.md)

**Engine development:** [getting-started](./docs/dev/getting-started.md) | [forge-dev](./docs/dev/forge-dev.md) | [creating-build-engine](./docs/dev/creating-build-engine.md) | [creating-test-runner](./docs/dev/creating-test-runner.md) | [creating-testenv-subengine](./docs/dev/creating-testenv-subengine.md)

**Design:** [DESIGN.md](./DESIGN.md) | [testenv-architecture](./docs/architecture/testenv-architecture.md)

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for commit conventions, testing workflow, project structure, and development patterns.

## License

Apache 2.0
