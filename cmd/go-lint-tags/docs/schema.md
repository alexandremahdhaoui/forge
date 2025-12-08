# Go Lint Tags Configuration Schema

## Overview

This document describes the configuration options for `go-lint-tags` in `forge.yaml`. The go-lint-tags engine verifies that all Go test files have valid build tags.

## Basic Configuration

```yaml
test:
  - name: verify-tags
    runner: go://go-lint-tags
```

## Configuration Options

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | The name of the verification stage. |
| `runner` | string | Must be `go://go-lint-tags` to use this verifier. |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `stage` | string | Same as `name` | The test stage name. |
| `rootDir` | string | `.` | Root directory to scan for test files. |

## Examples

### Minimal Configuration

```yaml
test:
  - name: verify-tags
    runner: go://go-lint-tags
```

### With Custom Root Directory

```yaml
test:
  - name: verify-tags
    runner: go://go-lint-tags
    rootDir: ./src
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
  - name: code-quality
    runner: go://parallel-test-runner
    spec:
      runners:
        - name: lint
          engine: go://go-lint
        - name: verify-tags
          engine: go://go-lint-tags
        - name: verify-licenses
          engine: go://go-lint-licenses
```

## Accepted Build Tags

The verifier accepts the following build tags:

| Tag | Purpose |
|-----|---------|
| `//go:build unit` | Unit tests (isolated, fast) |
| `//go:build integration` | Integration tests (require dependencies) |
| `//go:build e2e` | End-to-end tests (full system) |

## Test File Requirements

Each test file must have a build tag in the first 5 lines:

```go
//go:build unit

package myapp_test
```

## Skipped Directories

The following directories are automatically skipped:

- `vendor/`
- `.git/`
- `.tmp/`
- `node_modules/`

## Generated Artifacts

Each verification run creates a TestReport in the artifact store:

```yaml
testReports:
  - id: "test-verify-tags-verify-tags-20250106-abc123"
    stage: "verify-tags"
    status: "passed"
    startTime: "2025-01-06T10:00:00Z"
    duration: 0.123
    testStats:
      total: 45      # Total test files found
      passed: 45     # Files with valid tags
      failed: 0      # Files without tags
      skipped: 0
    coverage:
      enabled: false
      percentage: 0
```

## Default Behavior

When running go-lint-tags:

- Scans from current directory (or specified rootDir)
- Checks all `*_test.go` files
- Reports missing or invalid build tags
- Exits with failure if any files lack tags

## See Also

- [Go Lint Tags Usage Guide](usage.md)
- [Go Build Tags Documentation](https://pkg.go.dev/cmd/go#hdr-Build_constraints)
