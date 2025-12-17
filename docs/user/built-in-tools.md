# Built-in Tools Reference

This document provides a comprehensive reference for all built-in forge tools/engines. These tools are available out-of-the-box and don't require any configuration beyond specifying their URI in your `forge.yaml`.

## Table of Contents

- [Overview](#overview)
- [Build Engines](#build-engines)
- [Test Runners](#test-runners)
- [Test Environments](#test-environments)
- [Utility Tools](#utility-tools)
- [Quick Reference Table](#quick-reference-table)

## Overview

Forge includes 16 built-in tools/engines organized into categories:
- **4 Build Engines** - For building binaries and containers
- **4 Test Runners** - For executing tests
- **4 Test Environment Tools** - For managing test infrastructure
- **4 Utility Tools** - For code quality, generation, and management

All tools are MCP servers and can be used directly via their `go://` URI or wrapped in engine aliases for customization.

**Note:** This document covers the 16 built-in engines that forge orchestrates. The forge CLI itself (the 17th tool) is the orchestrator and is documented separately in [forge-cli.md](./forge-cli.md) and [cmd/forge/MCP.md](../../cmd/forge/MCP.md).

## Build Engines

### go-build

**Purpose:** Build Go binaries with automatic version injection from git

**URI:** `go://go-build`

**Features:**
- Automatic version metadata injection via ldflags
- Git-based versioning (commit SHA, tags, dirty flag)
- Custom build arguments support
- Custom environment variables support
- Cross-compilation support (GOOS/GOARCH)
- Static binary builds (CGO_ENABLED=0 by default)
- Parallel build support
- Artifact tracking in artifact store

**Basic Usage:**
```yaml
build:
  - name: myapp
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
```

**Advanced Usage with Custom Args and Environment Variables:**
```yaml
# Static binary with build tags
build:
  - name: static-binary
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
    spec:
      args:
        - "-tags=netgo"
        - "-ldflags=-w -s"
      env:
        CGO_ENABLED: "0"

# Cross-compilation for Linux AMD64
  - name: myapp-linux-amd64
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
    spec:
      env:
        GOOS: "linux"
        GOARCH: "amd64"
        CGO_ENABLED: "0"
```

**Configuration Options:**
- `spec.args` - Array of additional arguments passed to `go build` (e.g., `["-tags=netgo", "-ldflags=-w -s"]`)
- `spec.env` - Map of environment variables for the build (e.g., `{"GOOS": "linux", "GOARCH": "amd64"}`)

**Environment Variables:**
- `GO_BUILD_LDFLAGS` - Linker flags to pass to `go build` command (optional)
- `CGO_ENABLED` - Set to "0" by default for static binaries (can be overridden via `spec.env`)

**Version Injection:**
Automatically injects:
- `Version` - Git tag or commit SHA
- `CommitSHA` - Full commit hash
- `BuildTimestamp` - RFC3339 timestamp

**When to use:** For all Go binary builds. This is the preferred way to build Go applications.

---

### container-build

**Purpose:** Build container images with support for docker, kaniko, or podman

**URI:** `go://container-build`

**Features:**
- Multi-mode support: docker (native), kaniko (rootless), or podman (rootless)
- Supports both Dockerfile and Containerfile
- Automatic image tagging with git versions
- Build arguments support (--build-arg)
- Build caching (native for docker/podman, directory-based for kaniko)
- Multi-stage build support
- Automatic version and latest tags

**Basic Usage:**
```yaml
build:
  - name: myapp-image
    src: ./Containerfile
    engine: go://container-build
```

**Advanced Usage with Build Args:**
```yaml
build:
  - name: myapp-image
    src: ./Containerfile
    engine: go://container-build
```

Then set environment variable:
```bash
export BUILD_ARGS="GO_BUILD_LDFLAGS=-X main.Version=1.0.0 BASE_IMAGE=alpine:3.18"
forge build
```

**Environment Variables:**
- `CONTAINER_BUILD_ENGINE` - Build mode: docker, kaniko, or podman (required)
- `BUILD_ARGS` - Space-separated build arguments to pass to the build engine (optional, e.g., "KEY1=value1 KEY2=value2")
- `KANIKO_CACHE_DIR` - Cache directory for kaniko mode (optional, default: ~/.kaniko-cache, supports ~ expansion)

**Build Modes:**
- **docker**: Native Docker builds (fast, requires Docker daemon, not rootless)
- **kaniko**: Rootless builds using Kaniko executor (runs in container via docker, secure, layer caching to disk)
- **podman**: Native Podman builds (rootless, requires Podman)

**Image Tagging:**
Automatically creates two tags for each build:
- `<name>:<git-commit-sha>` - Version tag with full commit hash
- `<name>:latest` - Latest tag

**When to use:** For building container images from Dockerfiles/Containerfiles with flexible backend selection.

---

### generic-builder

**Purpose:** Execute arbitrary commands as build steps

**URI:** `go://generic-builder`

**Features:**
- Run any CLI tool as a build engine
- Template support for arguments ({{ .Name }}, {{ .Src }}, {{ .Dest }}, {{ .Version }})
- Environment variable support
- Working directory control
- envFile support for secrets
- Structured artifact output

**Basic Usage:**
```yaml
engines:
  - alias: protoc
    type: builder
    builder:
      - engine: go://generic-builder
        spec:
          command: "protoc"
          args: ["--go_out=.", "api/service.proto"]
          workDir: "."

build:
  - name: generate-proto
    src: ./api
    engine: alias://protoc
```

**Advanced Usage with Templates:**
```yaml
engines:
  - alias: protoc-advanced
    type: builder
    builder:
      - engine: go://generic-builder
        spec:
          command: "protoc"
          args:
            - "--go_out={{ .Dest }}"
            - "--go_opt=paths=source_relative"
            - "{{ .Src }}/api.proto"
          env:
            PROTO_VERSION: "3"

build:
  - name: generate-proto
    src: ./proto
    dest: ./pkg/api
    engine: alias://protoc-advanced
```

**Configuration Options:**
- `spec.command` - Command to execute (required)
- `spec.args` - Array of arguments (supports Go templates)
- `spec.env` - Map of environment variables
- `spec.envFile` - Path to environment file for secrets
- `spec.workDir` - Working directory (defaults to current directory)

**Template Variables:**
Arguments support Go template syntax with these fields:
- `{{ .Name }}` - Build name from spec
- `{{ .Src }}` - Source directory from spec
- `{{ .Dest }}` - Destination directory from spec
- `{{ .Version }}` - Version string (typically git commit)

**Error Handling:**
- Exit code 0: Success, artifact is tracked
- Exit code != 0: Failure, error returned with stdout/stderr

**When to use:** When no built-in builder exists for your tool (protoc, npm, custom scripts, code formatters, code generators, etc.)

**See Also:** [Generic Builder Guide](./generic-builder.md)

---

### go-format

**Purpose:** Format Go code using gofumpt

**URI:** `go://go-format`

**Features:**
- Runs gofumpt (stricter superset of gofmt)
- Formats all .go files recursively
- Simplifies code where possible
- Writes changes directly to files
- Configurable gofumpt version
- Can be used as a build step

**Usage:**
```yaml
build:
  - name: format-code
    src: .
    engine: go://go-format
```

**Environment Variables:**
- `GOFUMPT_VERSION` - Version of gofumpt to use (optional, default: v0.6.0)

**Formatting Tool:**
Uses `gofumpt` which is a stricter formatter than `gofmt`. It:
- Applies all standard gofmt rules
- Adds extra formatting rules for consistency
- Removes unnecessary whitespace
- Enforces stricter import grouping
- Can be run via `go run mvdan.cc/gofumpt@{version} -w {path}`

**When to use:** To ensure consistent Go code formatting before builds. Prefer this over running gofmt manually as it provides stricter, more opinionated formatting.

---

## Test Runners

### go-test

**Purpose:** Run Go tests with coverage and reporting

**URI:** `go://go-test`

**Features:**
- Uses gotestsum (v1.13.0) for better test output and formatting
- Generates JUnit XML reports
- Generates coverage profiles (atomic mode)
- Supports build tags for test isolation
- Race detector enabled (-race)
- Test caching disabled (-count=1)
- UUID-based test report tracking
- Automatic artifact storage
- Parses test statistics from JUnit XML
- Calculates coverage metrics

**Usage:**
```yaml
test:
  - name: unit
    runner: go://go-test

  - name: integration
    testenv: "alias://my-testenv"
    runner: go://go-test
```

**Test Command:**
Runs the following command:
```bash
go run gotest.tools/gotestsum@v1.13.0 \
  --format pkgname-and-test-fails \
  --format-hide-empty-pkg \
  --junitfile {tmpDir}/test-{stage}-{name}.xml \
  -- \
  -tags {stage} \
  -race \
  -count=1 \
  -cover \
  -coverprofile {tmpDir}/test-{stage}-{name}-coverage.out \
  ./...
```

**Build Tags:**
Automatically uses `-tags=<stage-name>` (e.g., `-tags=unit`, `-tags=integration`, `-tags=e2e`). Tests must have corresponding build tags:
```go
//go:build unit

package myapp_test
```

**Output Files:**
- `test-{stage}-{name}.xml` - JUnit XML report
- `test-{stage}-{name}-coverage.out` - Coverage profile

**Environment Variables Passed to Tests:**
All environment variables from testenv are passed to the test process, including:
- `FORGE_TESTENV_TMPDIR` - Test environment temporary directory
- `FORGE_ARTIFACT_*` - Artifact file paths from testenv
- `FORGE_METADATA_*` - Metadata from testenv

**Environment Variables (Configuration):**
- `FORGE_ARTIFACT_STORE_PATH` - Path to artifact store (optional, defaults to .forge/artifacts.yaml)

**Test Report:**
Returns structured TestReport with:
- ID (UUID)
- Stage and name
- Status (passed/failed)
- Test statistics (total, passed, failed, skipped)
- Coverage metrics (percentage, covered lines, total lines)
- Duration
- Artifact file paths

**When to use:** For all Go test execution. This is the standard test runner.

---

### go-lint-tags

**Purpose:** Verify all test files have proper build tags

**URI:** `go://go-lint-tags`

**Features:**
- Scans all *_test.go files recursively
- Ensures each has a `//go:build` tag
- Validates tags are one of: unit, integration, or e2e
- Skips vendor, .git, .tmp, and node_modules directories
- Returns detailed table of violations with file paths
- Prevents tests from running in wrong stages

**Usage:**
```yaml
test:
  - name: verify-tags
    runner: go://go-lint-tags
```

**Valid Build Tags:**
The tool accepts these build tags:
```go
//go:build unit
//go:build integration
//go:build e2e
```

**Output:**
If violations are found, displays a formatted table:
```
FILE PATH                                                                MISSING TAG
--------------------------------------------------------------------------------  ------------
path/to/test_file.go                                                             X
```

**Exit Codes:**
- Exit 0: All test files have valid build tags
- Exit 1: One or more test files missing build tags

**When to use:** As a pre-test validation step to ensure test isolation.

---

### generic-test-runner

**Purpose:** Execute arbitrary commands as test runners

**URI:** `go://generic-test-runner`

**Features:**
- Run any command as a test
- Pass/fail based on exit code (0 = pass, non-zero = fail)
- Generates structured TestReport
- Environment variable support
- envFile support for secrets
- Working directory control
- Captures stdout and stderr
- No coverage parsing (returns 0%)

**Usage:**
```yaml
engines:
  - alias: shellcheck
    type: test-runner
    testRunner:
      - engine: go://generic-test-runner
        spec:
          command: "shellcheck"
          args: ["scripts/*.sh"]
          workDir: "."

test:
  - name: shell-lint
    runner: alias://shellcheck
```

**Advanced Usage with Environment Variables:**
```yaml
engines:
  - alias: custom-validator
    type: test-runner
    testRunner:
      - engine: go://generic-test-runner
        spec:
          command: "my-validator"
          args: ["--strict", "./config"]
          env:
            VALIDATOR_MODE: "strict"
            LOG_LEVEL: "debug"
          envFile: ".secrets.env"
          workDir: "."

test:
  - name: validate-config
    runner: alias://custom-validator
```

**Configuration Options:**
- `spec.command` - Command to execute (required)
- `spec.args` - Array of arguments
- `spec.env` - Map of environment variables
- `spec.envFile` - Path to environment file (supports shell exports and KEY=value format)
- `spec.workDir` - Working directory (defaults to current directory)

**Environment File Format:**
The envFile supports:
```bash
# Comments are ignored
export KEY1="value1"
KEY2=value2
KEY3='value3'
```

**Test Report:**
Returns structured TestReport with:
- Status (passed/failed based on exit code)
- Timestamp
- Test statistics (total: 1, passed: 0 or 1, failed: 0 or 1)
- Coverage (always 0% - generic runner doesn't parse coverage)

**When to use:** When no built-in runner exists for your test tool (shellcheck, custom validators, Python test runners, etc.)

**See Also:** [Generic Test Runner Guide](./generic-test-runner.md)

---

### go-lint

**Purpose:** Run golangci-lint with auto-fix

**URI:** `go://go-lint`

**Features:**
- Runs golangci-lint with --fix flag
- Automatically fixes issues where possible
- Returns pass/fail as test report
- Works with your .golangci.yml config
- Configurable golangci-lint version
- Structured TestReport output

**Usage:**
```yaml
test:
  - name: lint
    runner: go://go-lint
```

**Command Executed:**
```bash
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@{version} run --fix
```

**Environment Variables:**
- `GOLANGCI_LINT_VERSION` - Version of golangci-lint to use (optional, default: v2.6.0)

**Example with Custom Version:**
```bash
GOLANGCI_LINT_VERSION=v2.7.0 forge test lint run
```

**Test Report:**
Returns structured TestReport with:
- Status (passed/failed based on exit code)
- Duration (in seconds)
- Error message if lint failed
- Test statistics (passed: 1 if success, failed: 1 if errors found)

**Exit Codes:**
- Exit 0: No linting issues (or all auto-fixed)
- Exit non-zero: Linting issues found that couldn't be auto-fixed

**When to use:** For Go code linting. Prefer this over wrapping golangci-lint manually.

---

## Test Environments

### testenv

**Purpose:** Complete test environment orchestrator

**URI:** `go://testenv`

**Features:**
- Orchestrates multiple testenv sub-engines
- Creates Kind clusters via testenv-kind
- Sets up local registries via testenv-lcr
- Installs Helm charts via testenv-helm-install
- Manages environment lifecycle (create, get, list, delete)
- Tracks environments in artifact store

**Usage:**
```yaml
# Option 1: Use default (creates Kind + registry)
test:
  - name: integration
    testenv: "go://testenv"
    runner: "go://go-test"

# Option 2: Use custom alias
engines:
  - alias: my-testenv
    type: testenv
    testenv:
      - engine: "go://testenv-kind"
      - engine: "go://testenv-lcr"
        spec:
          enabled: true

test:
  - name: integration
    testenv: "alias://my-testenv"
    runner: "go://go-test"
```

**When to use:** For integration tests requiring Kubernetes clusters and container registries.

---

### testenv-kind

**Purpose:** Create and manage Kind (Kubernetes in Docker) clusters

**URI:** `go://testenv-kind`

**Features:**
- Creates isolated Kind clusters
- Unique cluster names (forge-test-{stage}-{timestamp}-{random})
- Generates kubeconfig files
- Automatic cleanup on delete
- Stores cluster metadata

**Environment Variables Required:**
- `KIND_BINARY` - Path to kind binary (e.g., "kind")
- `KIND_BINARY_PREFIX` - Optional prefix (e.g., "sudo")

**Outputs:**
- `kubeconfig` file in testenv tmpDir
- Cluster name in metadata

**When to use:** When you need just a Kubernetes cluster (no registry).

---

### testenv-lcr

**Purpose:** Local Container Registry with TLS

**URI:** `go://testenv-lcr`

**Features:**
- Creates TLS-enabled container registry in Kind
- Generates CA certificates
- Auto-pushes images from artifact store
- Stores credentials and certs in testenv tmpDir

**Configuration:**
```yaml
engines:
  - alias: my-testenv
    type: testenv
    testenv:
      - engine: "go://testenv-kind"
      - engine: "go://testenv-lcr"
        spec:
          enabled: true
          autoPushImages: true
          namespace: "local-container-registry"
```

**Outputs:**
- Registry credentials
- CA certificate
- Registry endpoint

**When to use:** When tests need to push/pull container images.

---

### testenv-helm-install

The `testenv-helm-install` engine installs Helm charts into a Kubernetes test environment (typically a Kind cluster).

**Configuration:**

```yaml
- engine: "go://testenv-helm-install"
  spec:
    charts:
      - name: podinfo-release           # Internal identifier
        sourceType: helm-repo            # Required: "helm-repo", "git", "oci", "s3"
        url: https://stefanprodan.github.io/podinfo  # Repository URL
        chartName: podinfo               # Chart name in the repository
        version: "6.0.0"                # Optional: version constraint
        namespace: test-podinfo          # Kubernetes namespace
        releaseName: test-podinfo        # Helm release name
        createNamespace: true            # Create namespace if missing
        timeout: "5m"                    # Helm operation timeout
        disableWait: false               # Wait for resources to be ready
        values:                          # Inline Helm values
          replicaCount: 2
          service:
            type: ClusterIP
```

**Key Fields:**

- `name`: Internal identifier for the chart configuration
- `sourceType`: Must be `"helm-repo"` (currently only supported type)
- `url`: Helm repository URL
- `chartName`: Name of the chart in the repository
- `releaseName`: Helm release name (defaults to `name`)
- `namespace`: Target namespace (defaults to "default")
- `createNamespace`: Create namespace if it doesn't exist
- `timeout`: Helm operation timeout (default: "5m")
- `values`: Inline Helm values (flat key-value pairs supported)
- `valuesFiles`: List of values file paths

**Lifecycle Options:**

- `disableWait`: Skip waiting for resources (default: false)
- `forceUpgrade`: Use helm upgrade --force (default: false)
- `disableHooks`: Disable Helm hooks (default: false)
- `testEnable`: Run helm tests after install (default: false)

**Note:** Only `sourceType: helm-repo` is currently implemented. Git, OCI, and S3 sources are planned for future releases.

---

## Utility Tools

### test-report

**Purpose:** Manage test reports and artifacts

**URI:** `go://test-report`

**Features:**
- Query test reports from artifact store
- List reports by stage
- Get detailed report information
- Delete old reports and artifacts

**Commands:**
```bash
forge test report get <report-id>
forge test report list --stage=unit
forge test report delete <report-id>
```

**When to use:** For CI/CD pipelines to retrieve test results, or cleanup old reports.

---

### go-gen-mocks

**Purpose:** Generate Go mocks using mockery

**URI:** `go://go-gen-mocks`

**Features:**
- Generates mocks for Go interfaces
- Uses mockery under the hood
- Configurable output directories

**Usage:**
```yaml
build:
  - name: go-gen-mocks
    src: ./pkg
    dest: ./mocks
    engine: go://go-gen-mocks
```

**When to use:** For automated mock generation in Go projects.

---

### go-gen-openapi

**Purpose:** Generate Go client/server code from OpenAPI specs

**URI:** `go://go-gen-openapi`

**Features:**
- Generates Go code from OpenAPI 3.0 specs
- Creates both client and server stubs
- Version-aware generation

**Usage:** See [go-gen-openapi MCP documentation](../../cmd/go-gen-openapi/MCP.md) and [migration guide](../migration-go-gen-openapi.md)

**When to use:** For projects using OpenAPI/Swagger specifications.

---

### ci-orchestrator

**Purpose:** CI pipeline orchestration (placeholder)

**URI:** `go://ci-orchestrator`

**Status:** Not yet implemented - returns "not yet implemented" error

**Planned Features:**
- Orchestrate multi-stage CI pipelines
- Parallel job execution
- Dependency management

**When to use:** Reserved for future CI/CD orchestration features.

---

## Quick Reference Table

| Tool | Category | URI | Primary Use |
|------|----------|-----|-------------|
| go-build | Build | `go://go-build` | Build Go binaries |
| container-build | Build | `go://container-build` | Build container images |
| generic-builder | Build | `go://generic-builder` | Wrap custom build tools |
| go-format | Build | `go://go-format` | Format Go code |
| go-test | Test Runner | `go://go-test` | Run Go tests |
| go-lint-tags | Test Runner | `go://go-lint-tags` | Verify build tags |
| generic-test-runner | Test Runner | `go://generic-test-runner` | Wrap custom test tools |
| go-lint | Test Runner | `go://go-lint` | Run golangci-lint |
| testenv | Testenv | `go://testenv` | Full test environment |
| testenv-kind | Testenv | `go://testenv-kind` | Kind clusters |
| testenv-lcr | Testenv | `go://testenv-lcr` | Local container registry |
| testenv-helm-install | Testenv | `go://testenv-helm-install` | Helm chart installation |
| test-report | Utility | `go://test-report` | Test report management |
| go-gen-mocks | Utility | `go://go-gen-mocks` | Mock generation |
| go-gen-openapi | Utility | `go://go-gen-openapi` | OpenAPI code gen |
| ci-orchestrator | Utility | `go://ci-orchestrator` | CI orchestration (NYI) |

## Usage Patterns

### Standard Go Project

```yaml
build:
  - name: format-code
    src: .
    engine: go://go-format

  - name: myapp
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build

test:
  - name: verify-tags
    runner: go://go-lint-tags

  - name: unit
    runner: go://go-test

  - name: lint
    runner: go://go-lint

  - name: integration
    testenv: "go://testenv"
    runner: go://go-test
```

### With Container Builds

```yaml
build:
  - name: myapp
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build

  - name: myapp-image
    src: ./Containerfile
    engine: go://container-build

test:
  - name: integration
    testenv: "go://testenv"  # Includes registry
    runner: go://go-test
```

### Custom Tools Integration

```yaml
engines:
  - alias: npm-build
    type: builder
    builder:
      - engine: go://generic-builder
        spec:
          command: "npm"
          args: ["run", "build"]
          workDir: "./frontend"

  - alias: shellcheck
    type: test-runner
    testRunner:
      - engine: go://generic-test-runner
        spec:
          command: "shellcheck"
          args: ["scripts/*.sh"]

build:
  - name: frontend
    src: ./frontend
    engine: alias://npm-build

test:
  - name: shell-lint
    runner: alias://shellcheck
```

## Best Practices

1. **Prefer built-in tools over generic wrappers**
   - Use `go://go-build` instead of wrapping `go build`
   - Use `go://go-test` instead of wrapping `go test`

2. **Use generic-* tools for third-party integrations**
   - `generic-builder` for npm, protoc, custom scripts
   - `generic-test-runner` for shellcheck, custom validators

3. **Always verify build tags**
   - Add `verify-tags` as first test stage
   - Prevents tests running in wrong contexts

4. **Use testenv for integration tests**
   - Creates isolated, reproducible environments
   - Automatic cleanup
   - Consistent across developers and CI

5. **Format before building**
   - Add `go-format` as first build step
   - Ensures consistent code style

## Related Documentation

- [forge.yaml Schema](./forge-yaml-schema.md)
- [Generic Builder Guide](./generic-builder.md)
- [Generic Test Runner Guide](./generic-test-runner.md)
- [Test Environment Architecture](../architecture/testenv-architecture.md)
- [Forge CLI Usage Guide](./forge-cli.md)

## MCP Documentation

Each tool has detailed MCP documentation in its source directory:
- See `cmd/<tool-name>/MCP.md` for tool-specific MCP protocol documentation
- See `cmd/<tool-name>/README.md` for implementation details
