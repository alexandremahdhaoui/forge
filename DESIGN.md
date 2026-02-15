# Forge Design Document

**Forge is an MCP-native build orchestration system where every component -- CLI, build engines, test runners, environment managers -- is an MCP server composable via declarative YAML.**

## Problem Statement

Go teams build services with Makefiles, shell scripts, and ad-hoc tooling. These scripts are imperative and fragile: they lack dependency tracking, produce unreproducible builds, and require manual test environment setup. AI coding agents cannot parse or invoke Makefile targets, so builds remain a human-only workflow. No single tool manages the full build-test lifecycle -- from binary compilation through container packaging, test environment provisioning, test execution, and artifact tracking -- in a way that is both declarative and machine-accessible.

Forge solves this with an MCP-first architecture. Every component speaks JSON-RPC 2.0 over stdio. A single `forge.yaml` file declares builds, tests, and engine aliases. AI agents invoke builds, create test environments, and run test suites through the same protocol that human operators use via the CLI.

## Tenets

1. **AI-native over human-convenient.** When in conflict, optimize for machine readability. Every operation is an MCP tool call.
2. **Declarative over imperative.** `forge.yaml` replaces shell scripts. Configuration declares intent; engines handle execution.
3. **Composable over monolithic.** MCP servers compose via stdio. Each engine is independent, testable, and replaceable.
4. **Convention over configuration.** Sensible defaults reduce boilerplate. Override when needed.
5. **Dogfooding.** Forge builds and tests itself using its own engines and test infrastructure.
6. **Fail fast and loud.** Surface errors immediately. Never silently skip a failing stage.

## Requirements

1. Define builds and tests in a single YAML file.
2. Build Go binaries and container images with automatic dependency tracking.
3. Create and teardown test environments (Kind clusters, TLS registries, Helm charts) in one command.
4. Run all test stages sequentially with fail-fast behavior.
5. Track build artifacts with git SHAs and timestamps.
6. Allow AI agents to invoke all operations via MCP protocol.
7. Support parallel builds and parallel test execution.
8. Support custom build engines and test runners via generic wrappers.

## Out of Scope

- Multi-language package managers (npm, pip) -- `generic-builder` handles these but no native engines exist.
- Remote/distributed builds -- all engines run locally.
- GUI or web dashboard.
- Container orchestration beyond Kind (no EKS, GKE provisioning).

## Success Criteria

- `forge test-all` validates the entire system end-to-end.
- AI agent (Claude Code) builds, tests, and manages environments using only MCP protocol.
- New engine can be scaffolded with `forge-dev` in under 30 minutes.
- Lazy rebuild skips unchanged artifacts, verified by timestamp comparison.

## Proposed Design

### High-Level Architecture

```
+-------------------+
|  AI Agent / User  |
+---------+---------+
          |
     forge.yaml
          |
+---------+---------+
|    forge CLI      |  MCP server + orchestrator
|                   |
| +---------------+ |
| | Config Parser | |  Reads forge.yaml
| | Engine Mgr    | |  Resolves go:// and alias:// URIs
| | Artifact Store| |  Tracks builds in .forge/artifact-store.yaml
| | Test Orchestr | |  Manages test lifecycle
| +---------------+ |
+---------+---------+
          | MCP over stdio (JSON-RPC 2.0)
          |
    +-----+------+----------+--------------+
    |            |           |              |
+---+----+ +----+----+ +----+-----+ +------+------+
| Build  | | Test    | | TestEnv  | | Code Gen    |
| Engines| | Runners | | Managers | | Tools       |
+--------+ +---------+ +----+-----+ +-------------+
                             |
                       +-----+------+
                       |            |
                  +----+---+  +----+----+
                  |testenv |  |testenv  |
                  |kind    |  |lcr      |
                  +--------+  +---------+
```

### Engine Resolution

Forge resolves engine URIs to executable MCP server processes:

- `go://engine-name` resolves to `go run github.com/alexandremahdhaoui/forge/cmd/engine-name@version --mcp`
- `alias://name` looks up the `engines` section in `forge.yaml`, then resolves each entry to `go://` engines
- Local development mode (`FORGE_RUN_LOCAL_ENABLED=true`) resolves to `go run ./cmd/engine-name --mcp`

### Testenv Chain Composition

```
forge test integration create
  |
  v
testenv (orchestrator)
  |
  +--> testenv-kind (create Kind cluster)
  |      |
  |      +--> sets KUBECONFIG env var
  |
  +--> testenv-lcr (create TLS registry)
  |      |
  |      +--> reads KUBECONFIG from env
  |      +--> sets TESTENV_LCR_FQDN env var
  |
  +--> testenv-helm-install (install charts)
         |
         +--> reads KUBECONFIG from env
         +--> uses {{.Env.TESTENV_LCR_FQDN}} in values
```

Sub-engines run sequentially. Each propagates environment variables to the next via `envPropagation` config. Template expansion (`{{.Env.VAR}}`) enables dynamic configuration between stages.

### Lazy Rebuild

```
forge build <name>
  |
  v
shouldRebuild(artifact)?
  |
  +-- No previous build?              --> YES, rebuild
  +-- Artifact file missing?           --> YES, rebuild
  +-- --force flag?                    --> YES, rebuild
  +-- File dependency changed (mtime)? --> YES, rebuild
  +-- External package version changed?--> YES, rebuild
  +-- None of the above?              --> SKIP

After build:
  go-dependency-detector scans Go AST
  Records file paths + mtimes, package versions
  Stores in artifact-store.yaml
```

### Parallel Execution

- **parallel-builder**: Wraps N build engines, runs concurrently, collects results.
- **parallel-test-runner**: Wraps N test runners, runs concurrently, merges TestReports.
  - `primaryCoverageRunner` selects which runner provides coverage data.
  - `TestStats` are summed across all runners.
  - Any failure produces overall failure.

## Technical Design

### Data Model

Key types from `pkg/forge/`:

```go
// BuildSpec represents a single artifact to build
type BuildSpec struct {
    Name   string                 `json:"name"`
    Src    string                 `json:"src"`
    Dest   string                 `json:"dest,omitempty"`
    Engine string                 `json:"engine"`
    Spec   map[string]interface{} `json:"spec,omitempty"`
}

// TestSpec defines a test stage configuration
type TestSpec struct {
    Name           string                 `json:"name"`
    Testenv        string                 `json:"testenv,omitempty"`
    Runner         string                 `json:"runner"`
    Spec           map[string]interface{} `json:"spec,omitempty"`
    EnvPropagation *EnvPropagation        `json:"envPropagation,omitempty"`
}

// TestReport represents a test execution report
type TestReport struct {
    ID           string    `json:"id"`
    Stage        string    `json:"stage"`
    Status       string    `json:"status"`
    StartTime    time.Time `json:"startTime"`
    Duration     float64   `json:"duration"`
    TestStats    TestStats `json:"testStats"`
    Coverage     Coverage  `json:"coverage"`
    ErrorMessage string    `json:"errorMessage,omitempty"`
}

// Artifact tracks a built artifact with dependency information
type Artifact struct {
    Name                     string                 `json:"name"`
    Type                     string                 `json:"type"`
    Location                 string                 `json:"location"`
    Timestamp                string                 `json:"timestamp"`
    Version                  string                 `json:"version"`
    Dependencies             []ArtifactDependency   `json:"dependencies,omitempty"`
    DependencyDetectorEngine string                 `json:"dependencyDetectorEngine,omitempty"`
    DependencyDetectorSpec   map[string]interface{} `json:"dependencyDetectorSpec,omitempty"`
}

// TestEnvironment represents a test environment instance
type TestEnvironment struct {
    ID               string            `json:"id"`
    Name             string            `json:"name"`
    Status           string            `json:"status"`
    CreatedAt        time.Time         `json:"createdAt"`
    UpdatedAt        time.Time         `json:"updatedAt"`
    TmpDir           string            `json:"tmpDir,omitempty"`
    Files            map[string]string `json:"files,omitempty"`
    ManagedResources []string          `json:"managedResources"`
    Metadata         map[string]string `json:"metadata,omitempty"`
    Env              map[string]string `json:"env,omitempty"`
}

// ArtifactStore is the top-level storage structure
type ArtifactStore struct {
    Version          string                      `json:"version"`
    LastUpdated      time.Time                   `json:"lastUpdated"`
    Artifacts        []Artifact                  `json:"artifacts"`
    TestEnvironments map[string]*TestEnvironment `json:"testEnvironments,omitempty"`
    TestReports      map[string]*TestReport      `json:"testReports,omitempty"`
}
```

### MCP Protocol

All engines communicate via JSON-RPC 2.0 over stdio. Forge spawns each engine as a child process with `--mcp` flag, then sends tool calls over stdin and reads responses from stdout.

**Request:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "build",
    "arguments": {
      "name": "forge",
      "src": "./cmd/forge",
      "dest": "./build/bin",
      "engine": "go://go-build"
    }
  }
}
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"name\":\"forge\",\"type\":\"binary\",\"location\":\"./build/bin/forge\"}"
      }
    ]
  }
}
```

**Tool names by engine category:**

| Category | MCP Tool Names |
|----------|---------------|
| Build engines | `build` |
| Test runners | `run` |
| Testenv managers | `create`, `get`, `list`, `delete` |
| Dependency detectors | `detectDependencies` |
| Test report management | `get`, `list`, `delete` |

### Component Catalog

28 CLI tools in `cmd/`, built with Go 1.25.0:

| Name | Location | Category | MCP Tools |
|------|----------|----------|-----------|
| forge | `cmd/forge` | Orchestration | `build`, `test-all`, `test-create`, `test-run`, `test-get`, `test-list`, `test-delete`, `config-validate` |
| forge-dev | `cmd/forge-dev` | Orchestration | `build` |
| forge-e2e | `cmd/forge-e2e` | Orchestration | `run` |
| ci-orchestrator | `cmd/ci-orchestrator` | Orchestration | (planned) |
| go-build | `cmd/go-build` | Build Engine | `build` |
| container-build | `cmd/container-build` | Build Engine | `build` |
| generic-builder | `cmd/generic-builder` | Build Engine | `build` |
| parallel-builder | `cmd/parallel-builder` | Build Engine | `build` |
| go-dependency-detector | `cmd/go-dependency-detector` | Dependency Detection | `detectDependencies` |
| go-gen-mocks-dep-detector | `cmd/go-gen-mocks-dep-detector` | Dependency Detection | `detectDependencies` |
| go-gen-openapi-dep-detector | `cmd/go-gen-openapi-dep-detector` | Dependency Detection | `detectDependencies` |
| testenv | `cmd/testenv` | Test Environment | `create`, `get`, `list`, `delete` |
| testenv-kind | `cmd/testenv-kind` | Test Environment | `create`, `get`, `list`, `delete` |
| testenv-lcr | `cmd/testenv-lcr` | Test Environment | `create`, `get`, `list`, `delete` |
| testenv-helm-install | `cmd/testenv-helm-install` | Test Environment | `create`, `get`, `list`, `delete` |
| testenv-stub | `cmd/testenv-stub` | Test Environment | `create`, `get`, `list`, `delete` |
| go-test | `cmd/go-test` | Test Runner | `run` |
| go-lint-tags | `cmd/go-lint-tags` | Test Runner | `run` |
| generic-test-runner | `cmd/generic-test-runner` | Test Runner | `run` |
| parallel-test-runner | `cmd/parallel-test-runner` | Test Runner | `run` |
| test-report | `cmd/test-report` | Test Management | `get`, `list`, `delete` |
| go-format | `cmd/go-format` | Code Quality | `build` |
| go-lint | `cmd/go-lint` | Code Quality | `run` |
| go-lint-licenses | `cmd/go-lint-licenses` | Code Quality | `run` |
| go-gen-mocks | `cmd/go-gen-mocks` | Code Generation | `build` |
| go-gen-openapi | `cmd/go-gen-openapi` | Code Generation | `build` |
| go-gen-protobuf | `cmd/go-gen-protobuf` | Code Generation | `build` |
| go-gen-bpf | `cmd/go-gen-bpf` | Code Generation | `build` |

### Public Package Catalog

13 packages in `pkg/`:

| Package | Location | Purpose |
|---------|----------|---------|
| enginecli | `pkg/enginecli` | Common CLI bootstrapping for forge engine binaries |
| enginedocs | `pkg/enginedocs` | Distributed documentation management across engines |
| engineframework | `pkg/engineframework` | MCP tool registration utilities for type-safe engines |
| engineversion | `pkg/engineversion` | Engine version management and reporting |
| eventualconfig | `pkg/eventualconfig` | Channel-based eventual consistency for async config coordination |
| flaterrors | `pkg/flaterrors` | Error tree flattening compatible with errors.Is/As |
| forge | `pkg/forge` | Core types: Spec, BuildSpec, TestSpec, Artifact, ArtifactStore, TestEnvironment, TestReport |
| mcpserver | `pkg/mcpserver` | MCP server framework for stdio-based JSON-RPC 2.0 |
| mcptypes | `pkg/mcptypes` | MCP wire types: BuildInput, RunInput, DetectDependenciesInput |
| mcputil | `pkg/mcputil` | MCP utilities: validation, batch handling, result formatting |
| portalloc | `pkg/portalloc` | Dynamic port allocation for test environments |
| templateutil | `pkg/templateutil` | Template expansion for environment variable interpolation |
| testenvutil | `pkg/testenvutil` | Test environment utilities: environment variable merging |

### Internal Package Catalog

10 packages in `internal/`:

| Package | Location | Purpose |
|---------|----------|---------|
| cmdutil | `internal/cmdutil` | Command execution utilities |
| engineresolver | `internal/engineresolver` | Engine URI resolution (go://, alias://) |
| enginetest | `internal/enginetest` | Test helpers for engine development |
| forgepath | `internal/forgepath` | Forge path resolution and directory utilities |
| gitutil | `internal/gitutil` | Git operations: commit SHA, version, dirty state |
| integration | `internal/integration` | Integration test utilities and helpers |
| mcpcaller | `internal/mcpcaller` | MCP client caller for engine invocation |
| orchestrate | `internal/orchestrate` | Build and test orchestration logic |
| testutil | `internal/testutil` | Test utilities and assertions |
| util | `internal/util` | General-purpose utilities |

## Design Patterns

1. **MCP-First.** Every engine is an MCP server communicating via JSON-RPC 2.0 over stdio. This makes all tooling directly accessible to AI agents without adapters or wrappers.

2. **Dogfooding.** Forge builds and tests itself. The `forge.yaml` in the repository root defines 28 build targets and 7 test stages that exercise every engine.

3. **Adapter Pattern.** testenv-lcr uses four adapters -- K8s (namespace), TLS (certificates), Credentials (htpasswd), Registry (deployment) -- coordinated via eventualconfig. Each adapter manages one concern.

4. **Eventual Consistency.** The `eventualconfig` package enables concurrent setup phases that depend on each other's outputs. Producers set values; consumers block until values are available. This eliminates polling and race conditions.

5. **Error Aggregation.** `flaterrors.Join` collects errors from multi-step operations (e.g., teardown) instead of failing on the first error. All errors surface to the caller.

6. **Engine URI Convention.** `go://name` references built-in engines. `alias://name` references user-defined engine chains in `forge.yaml`. This two-tier scheme separates distribution from composition.

7. **Code Generation.** `forge-dev` scaffolds new engines from OpenAPI specs using `engineframework` for type-safe MCP tool registration. Generated code follows `zz_generated` naming convention.

## Alternatives Considered

1. **Do nothing (keep Makefiles and shell scripts).** Rejected: Makefiles are imperative, fragile, and invisible to AI agents. No dependency tracking, no artifact versioning, no reproducible test environments.
2. **Makefile-based orchestration with AI wrappers.** Rejected: wrapping Makefiles in MCP adapters preserves the underlying fragility. Dependency tracking and artifact management still require custom tooling.
3. **Bazel/Buck.** Rejected: heavyweight, poor AI integration, steep learning curve for Go-centric teams.
4. **REST API instead of MCP/stdio.** Rejected: stdio requires no network stack, no server lifecycle management, and provides direct AI agent compatibility via MCP protocol.
5. **Single monolithic binary.** Rejected: composability requires independent engines. Each engine can be developed, tested, and versioned independently.

## Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| MCP protocol immaturity | Keep JSON-RPC 2.0 core simple; avoid protocol extensions |
| Go-only ecosystem perception | generic-builder and generic-test-runner wrap any CLI command |
| Local-only execution | Acceptable for current scope; ci-orchestrator planned for remote execution |

## Testing Strategy

`forge test-all` runs all stages sequentially with fail-fast behavior:

| Stage | Tool | Purpose |
|-------|------|---------|
| lint-tags | go-lint-tags | Verify all test files have build tags |
| lint-license | go-lint-licenses | Verify license headers |
| lint | go-lint | golangci-lint |
| unit | go-test | Fast Go tests, no external dependencies |
| integration | go-test + testenv | Kind cluster + TLS registry + Helm charts |
| e2e | forge-e2e | Full system validation |
| e2e-stub | go-test + testenv-stub | Lightweight testenv create/list/get/delete workflow |

## FAQ

**Why MCP over gRPC?**
MCP uses stdio transport. No server lifecycle management, no port allocation, no TLS configuration. AI coding agents (Claude Code, Cursor) speak MCP natively. gRPC would require a separate integration layer.

**Why sequential testenv sub-engines?**
Environment variables propagate between stages. testenv-kind sets KUBECONFIG; testenv-lcr reads it and sets TESTENV_LCR_FQDN; testenv-helm-install reads both. Parallel execution would break this dependency chain.

**Why not content-addressable caching?**
Timestamp comparison is simpler and sufficient for local builds. Content hashing adds complexity (large binary hashing, cache invalidation) without proportional benefit for the local-only use case.

## Appendix

### forge.yaml example (forge self-build, abbreviated)

```yaml
name: forge
envFile: .envrc
artifactStorePath: .forge/artifact-store.yaml

engines:
  - alias: setup-e2e-stub
    type: testenv
    testenv:
      - engine: "go://testenv-stub"

  - alias: setup-integration
    type: testenv
    testenv:
      - engine: "go://testenv-kind"
      - engine: "go://testenv-lcr"
        spec:
          enabled: true
          images:
            - name: local://for-testing-purposes:latest
          imagePullSecretNamespaces:
            - default
            - test-podinfo
      - engine: "go://testenv-helm-install"
        spec:
          charts:
            - name: podinfo-release
              sourceType: helm-repo
              url: https://stefanprodan.github.io/podinfo
              chartName: podinfo
              namespace: test-podinfo
              releaseName: test-podinfo
              createNamespace: true

test:
  - name: lint-tags
    runner: "go://go-lint-tags"
  - name: lint-license
    runner: "go://go-lint-licenses"
  - name: lint
    runner: "go://go-lint"
  - name: unit
    runner: "go://go-test"
  - name: integration
    runner: "go://go-test"
    testenv: "alias://setup-integration"
  - name: e2e
    runner: "go://forge-e2e"
  - name: e2e-stub
    runner: "go://go-test"
    testenv: "alias://setup-e2e-stub"

build:
  - name: forge
    src: ./cmd/forge
    dest: ./build/bin
    engine: go://go-build
  - name: go-build
    src: ./cmd/go-build
    dest: ./build/bin
    engine: go://go-build
  - name: container-build
    src: ./cmd/container-build
    dest: ./build/bin
    engine: go://go-build
  - name: testenv
    src: ./cmd/testenv
    dest: ./build/bin
    engine: go://go-build
  - name: forge-dev
    src: ./cmd/forge-dev
    dest: ./build/bin
    engine: go://go-build
    spec:
      ldflags: "-X main.Version={{.GitVersion}}"
  # ... 23 more build targets
```
