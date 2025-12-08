# Go Lint Configuration Schema

## Overview

This document describes the configuration options for `go-lint` in `forge.yaml`. The go-lint engine runs golangci-lint on Go code with auto-fix enabled.

## Basic Configuration

```yaml
test:
  - name: lint
    runner: go://go-lint
```

## Configuration Options

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | The name of the lint stage. |
| `runner` | string | Must be `go://go-lint` to use this linter. |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `stage` | string | Same as `name` | The test stage name. |

## Examples

### Minimal Configuration

```yaml
test:
  - name: lint
    runner: go://go-lint
```

### With Custom Stage Name

```yaml
test:
  - name: code-quality
    stage: lint
    runner: go://go-lint
```

### Combined with Other Linters

```yaml
test:
  - name: lint
    runner: go://go-lint

  - name: verify-tags
    runner: go://go-lint-tags

  - name: verify-licenses
    runner: go://go-lint-licenses
```

### In Parallel Test Runner

```yaml
test:
  - name: validation
    runner: go://parallel-test-runner
    spec:
      runners:
        - name: lint
          engine: go://go-lint
        - name: verify-tags
          engine: go://go-lint-tags
```

## Environment Variables

Configure the linter version via environment variable:

```bash
# Use a specific version
export GOLANGCI_LINT_VERSION=v2.5.0

# Then run tests
forge test run lint
```

## Generated Artifacts

Each lint run creates a TestReport in the artifact store:

```yaml
testReports:
  - id: "test-lint-lint-20250106-abc123"
    stage: "lint"
    status: "passed"
    startTime: "2025-01-06T10:00:00Z"
    duration: 5.432
    testStats:
      total: 0       # No issues found
      passed: 1
      failed: 0
      skipped: 0
    coverage:
      enabled: false
      percentage: 0
```

## golangci-lint Configuration

Create a `.golangci.yml` in your project root to customize linting:

```yaml
linters:
  enable:
    - gofmt
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - ineffassign

run:
  timeout: 5m
  tests: true

issues:
  exclude-use-default: false
```

## Default Behavior

When running go-lint:

- Uses golangci-lint version v2.6.0 (configurable via env)
- Applies `--fix` flag automatically
- Reads `.golangci.yml` if present
- Reports status based on exit code

## See Also

- [Go Lint Usage Guide](usage.md)
- [golangci-lint Documentation](https://golangci-lint.run/)
