# Parallel Test Runner Configuration Schema

## Overview

This document describes the configuration options for `parallel-test-runner` in `forge.yaml`. The parallel-test-runner engine executes multiple test runners concurrently and aggregates their results.

## Basic Configuration

```yaml
test:
  - name: unit
    runner: go://parallel-test-runner
    spec:
      runners:
        - name: go-test
          engine: go://go-test
        - name: lint
          engine: go://go-lint
```

## Configuration Options

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | The name of the test stage. |
| `runner` | string | Must be `go://parallel-test-runner` to use this runner. |
| `spec.runners` | []RunnerConfig | List of test runners to execute in parallel. |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `stage` | string | Same as `name` | The test stage name. |
| `spec.primaryCoverageRunner` | string | - | Name of the runner whose coverage is used in the result. |

## RunnerConfig Schema

Each runner in `spec.runners` has the following fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique identifier for the runner (used for primaryCoverageRunner). |
| `engine` | string | Yes | The engine URI (must be `go://` format, not alias). |
| `spec` | object | No | Engine-specific configuration passed to the sub-runner. |

## Examples

### Minimal Configuration

```yaml
test:
  - name: unit
    runner: go://parallel-test-runner
    spec:
      runners:
        - name: tests
          engine: go://go-test
```

### Tests with Coverage Selection

```yaml
test:
  - name: unit
    runner: go://parallel-test-runner
    spec:
      primaryCoverageRunner: go-test
      runners:
        - name: go-test
          engine: go://go-test
        - name: lint
          engine: go://go-lint
```

### Multiple Test Types

```yaml
test:
  - name: full-validation
    runner: go://parallel-test-runner
    spec:
      primaryCoverageRunner: unit-tests
      runners:
        - name: unit-tests
          engine: go://go-test
          spec:
            stage: unit
        - name: lint
          engine: go://go-lint
        - name: verify-tags
          engine: go://go-lint-tags
        - name: verify-licenses
          engine: go://go-lint-licenses
```

### With Generic Test Runner

```yaml
test:
  - name: validation
    runner: go://parallel-test-runner
    spec:
      runners:
        - name: go-test
          engine: go://go-test
        - name: security
          engine: go://generic-test-runner
          spec:
            command: gosec
            args: ["./..."]
        - name: typecheck
          engine: go://generic-test-runner
          spec:
            command: go
            args: ["vet", "./..."]
```

### Integration with Test Environment

```yaml
test:
  - name: integration
    testenv: go://testenv
    runner: go://parallel-test-runner
    spec:
      primaryCoverageRunner: integration-tests
      runners:
        - name: integration-tests
          engine: go://go-test
          spec:
            stage: integration
        - name: lint
          engine: go://go-lint
```

## Coverage Behavior

### With primaryCoverageRunner

```yaml
spec:
  primaryCoverageRunner: go-test
  runners:
    - name: go-test
      engine: go://go-test
    - name: lint
      engine: go://go-lint
```

Result:
```json
{
  "coverage": {
    "enabled": true,
    "percentage": 85.3  // From go-test only
  }
}
```

### Without primaryCoverageRunner

```yaml
spec:
  runners:
    - name: lint
      engine: go://go-lint
    - name: verify-tags
      engine: go://go-lint-tags
```

Result:
```json
{
  "coverage": {
    "enabled": false,
    "percentage": 0
  }
}
```

## Result Aggregation

Test statistics are summed across all runners:

```yaml
# If runner A has: total=50, passed=48, failed=2
# And runner B has: total=20, passed=20, failed=0
# Result: total=70, passed=68, failed=2
```

## Engine URI Requirements

The `engine` field must use the `go://` URI format:

```yaml
# Correct
engine: go://go-test
engine: go://go-lint
engine: go://generic-test-runner

# Incorrect (aliases not supported)
engine: alias://my-runner
```

## Generated Artifacts

Aggregated TestReport in artifact store:

```yaml
testReports:
  - id: "test-unit-unit-20250106-abc123"
    stage: "unit"
    status: "passed"
    startTime: "2025-01-06T10:00:00Z"
    duration: 5.432
    testStats:
      total: 70
      passed: 68
      failed: 2
      skipped: 0
    coverage:
      enabled: true
      percentage: 85.3
```

## See Also

- [Parallel Test Runner Usage Guide](usage.md)
- [go-test Configuration](../../go-test/docs/schema.md)
- [go-lint Configuration](../../go-lint/docs/schema.md)
