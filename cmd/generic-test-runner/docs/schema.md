# Generic Test Runner Configuration Schema

## Overview

This document describes the configuration options for `generic-test-runner` in `forge.yaml`. The generic-test-runner engine executes arbitrary commands as tests and reports pass/fail based on exit code.

## Basic Configuration

```yaml
test:
  - name: lint
    runner: go://generic-test-runner
    command: golangci-lint
    args: ["run", "./..."]
```

## Configuration Options

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | The name of the test stage. |
| `runner` | string | Must be `go://generic-test-runner` to use this test runner. |
| `command` | string | The command to execute. |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `stage` | string | Same as `name` | The test stage name. |
| `args` | []string | `[]` | Command-line arguments to pass to the command. |
| `env` | map[string]string | `{}` | Environment variables to set for the command. |
| `envFile` | string | - | Path to an environment file to load. |
| `workDir` | string | Current directory | Working directory for command execution. |
| `spec` | object | `{}` | Engine-specific configuration options. |

## Examples

### Minimal Configuration

```yaml
test:
  - name: lint
    runner: go://generic-test-runner
    command: golangci-lint
```

### With Arguments

```yaml
test:
  - name: lint
    runner: go://generic-test-runner
    command: golangci-lint
    args: ["run", "--fix", "./..."]
```

### With Environment Variables

```yaml
test:
  - name: test
    runner: go://generic-test-runner
    command: npm
    args: ["test"]
    env:
      CI: "true"
      NODE_ENV: "test"
```

### With Environment File

```yaml
test:
  - name: test
    runner: go://generic-test-runner
    command: ./run-tests.sh
    envFile: .env.test
```

### With Working Directory

```yaml
test:
  - name: frontend-test
    runner: go://generic-test-runner
    command: npm
    args: ["test"]
    workDir: ./frontend
```

### Security Scanner

```yaml
test:
  - name: security
    runner: go://generic-test-runner
    command: gosec
    args: ["-fmt=json", "-out=security-report.json", "./..."]
```

### Shell Script

```yaml
test:
  - name: integration
    runner: go://generic-test-runner
    command: bash
    args: ["-c", "./scripts/integration-tests.sh"]
```

### Multiple Runners

```yaml
test:
  - name: lint
    runner: go://generic-test-runner
    command: golangci-lint
    args: ["run"]

  - name: security
    runner: go://generic-test-runner
    command: gosec
    args: ["./..."]

  - name: typecheck
    runner: go://generic-test-runner
    command: go
    args: ["vet", "./..."]
```

## Environment File Format

Environment files support the following format:

```bash
# Comments are ignored
KEY=value
export EXPORTED_KEY=value
QUOTED_VALUE="value with spaces"
SINGLE_QUOTED='value with spaces'
```

## Generated Artifacts

Each test run creates a TestReport in the artifact store:

```yaml
testReports:
  - id: "test-lint-lint-20250106-abc123"
    stage: "lint"
    status: "passed"
    startTime: "2025-01-06T10:00:00Z"
    duration: 1.234
    testStats:
      total: 1
      passed: 1
      failed: 0
      skipped: 0
    errorMessage: ""
```

## Exit Code Mapping

| Exit Code | Status | Description |
|-----------|--------|-------------|
| 0 | passed | Command executed successfully |
| Non-zero | failed | Command failed or reported errors |

## Default Behavior

When no optional fields are provided:

- Uses current directory as working directory
- Inherits parent process environment
- No additional arguments passed
- Status determined solely by exit code

## See Also

- [Generic Test Runner Usage Guide](usage.md)
- [Forge Test Documentation](../../docs/forge-test-usage.md)
