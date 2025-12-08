# Go Test Configuration Schema

## Overview

This document describes the configuration options for `go-test` in `forge.yaml`. The go-test engine runs Go tests with coverage reporting and JUnit XML output.

## Basic Configuration

```yaml
test:
  - name: unit
    stage: unit
    runner: go://go-test
```

## Configuration Options

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | The name of the test stage (e.g., "unit", "integration"). |
| `runner` | string | Must be `go://go-test` to use this test runner. |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `stage` | string | Same as `name` | The test stage, used for build tag selection. |
| `testenv` | string | - | Test environment engine (e.g., `go://testenv`). |
| `spec` | object | `{}` | Engine-specific configuration options (see below). |

## Spec Options

The `spec` field contains engine-specific configuration:

### `env` (map of string to string)

Environment variables to set for test execution.

```yaml
spec:
  env:
    DATABASE_URL: "postgres://localhost:5432/testdb"
    LOG_LEVEL: "debug"
```

### `envPropagation` (object)

Control how test environment variables are propagated to tests.

```yaml
spec:
  envPropagation:
    disabled: false
    whitelist: ["KUBECONFIG", "DATABASE_URL"]
    blacklist: ["SECRET_TOKEN"]
```

**Fields:**
- `disabled` (bool): If true, no testenv variables are propagated
- `whitelist` ([]string): Only propagate these specific variables
- `blacklist` ([]string): Propagate all except these variables

**Note:** Whitelist and blacklist are mutually exclusive.

## Examples

### Minimal Configuration

```yaml
test:
  - name: unit
    runner: go://go-test
```

### With Test Environment

```yaml
test:
  - name: integration
    stage: integration
    testenv: go://testenv
    runner: go://go-test
```

### With Environment Variables

```yaml
test:
  - name: integration
    stage: integration
    testenv: go://testenv
    runner: go://go-test
    spec:
      env:
        TEST_TIMEOUT: "60s"
        VERBOSE: "true"
```

### With Environment Propagation Whitelist

```yaml
test:
  - name: integration
    stage: integration
    testenv: go://testenv
    runner: go://go-test
    spec:
      envPropagation:
        whitelist:
          - KUBECONFIG
          - DATABASE_URL
```

### With Environment Propagation Blacklist

```yaml
test:
  - name: integration
    stage: integration
    testenv: go://testenv
    runner: go://go-test
    spec:
      envPropagation:
        blacklist:
          - SECRET_TOKEN
          - API_KEY
```

### Full Configuration

```yaml
test:
  - name: integration
    stage: integration
    testenv: go://testenv
    runner: go://go-test
    spec:
      env:
        TEST_TIMEOUT: "120s"
        VERBOSE: "true"
      envPropagation:
        whitelist:
          - KUBECONFIG
```

## Build Tags

The `stage` field is used to select Go build tags:

| Stage | Build Tag |
|-------|-----------|
| `unit` | `-tags=unit` |
| `integration` | `-tags=integration` |
| `e2e` | `-tags=e2e` |

## Generated Artifacts

Each test run creates a TestReport in the artifact store:

```yaml
testReports:
  - id: "test-unit-unit-20250106-abc123"
    stage: "unit"
    status: "passed"
    startTime: "2025-01-06T10:00:00Z"
    duration: 5.432
    testStats:
      total: 42
      passed: 42
      failed: 0
      skipped: 0
    coverage:
      enabled: true
      percentage: 85.3
```

## Default Behavior

When no `spec` is provided, go-test uses these defaults:

- Runs tests with `-tags={stage}`
- Generates coverage profile
- Produces JUnit XML report
- Propagates all test environment variables

## See Also

- [Go Test Usage Guide](usage.md)
- [Forge Test Documentation](../../docs/forge-test-usage.md)
