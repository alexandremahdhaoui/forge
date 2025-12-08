# Generic Test Runner Usage Guide

## Purpose

`generic-test-runner` is a forge engine for executing arbitrary commands as test runners. It provides structured TestReport output with pass/fail status based on exit code, making it ideal for integrating linters, security scanners, and custom test frameworks.

## Invocation

### MCP Mode

Run as an MCP server:

```bash
generic-test-runner --mcp
```

Forge invokes this automatically when using:

```yaml
runner: go://generic-test-runner
```

## Available MCP Tools

### `run`

Execute a command as a test and generate TestReport.

**Input Schema:**
```json
{
  "stage": "string (required)",
  "name": "string (required)",
  "command": "string (required)",
  "args": ["string"],
  "env": {"key": "value"},
  "envFile": "string",
  "workDir": "string",
  "tmpDir": "string",
  "buildDir": "string",
  "rootDir": "string"
}
```

**Output:**
```json
{
  "stage": "string",
  "status": "passed|failed",
  "startTime": "2025-01-06T10:00:00Z",
  "duration": 1.234,
  "testStats": {
    "total": 1,
    "passed": 1,
    "failed": 0,
    "skipped": 0
  },
  "errorMessage": "string"
}
```

**Example - Run linter:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "run",
    "arguments": {
      "stage": "lint",
      "name": "golangci-lint",
      "command": "golangci-lint",
      "args": ["run", "./..."]
    }
  }
}
```

**Example - Run security scanner:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "run",
    "arguments": {
      "stage": "security",
      "name": "gosec",
      "command": "gosec",
      "args": ["-fmt=json", "./..."]
    }
  }
}
```

### `docs-list`

List all available documentation for generic-test-runner.

### `docs-get`

Get a specific documentation by name.

**Input Schema:**
```json
{
  "name": "string (required)"
}
```

### `docs-validate`

Validate documentation completeness.

## Common Use Cases

### Linters

Run golangci-lint as a test stage:

```yaml
test:
  - name: lint
    stage: lint
    runner: go://generic-test-runner
    command: golangci-lint
    args: ["run", "./..."]
```

### Security Scanners

Run gosec security scanner:

```yaml
test:
  - name: security
    stage: security
    runner: go://generic-test-runner
    command: gosec
    args: ["-fmt=json", "./..."]
```

### Custom Test Frameworks

Run any custom test framework:

```yaml
test:
  - name: custom
    stage: custom
    runner: go://generic-test-runner
    command: ./run-tests.sh
    args: ["--verbose"]
```

### Compliance Checkers

Run compliance validation:

```yaml
test:
  - name: compliance
    stage: compliance
    runner: go://generic-test-runner
    command: compliance-checker
    workDir: ./compliance
```

## Status Determination

- **Exit code 0** -> status: "passed"
- **Exit code != 0** -> status: "failed"

TestReport.errorMessage contains stdout/stderr on failure.

## Environment Variables

### From Environment File

Load environment variables from a file:

```yaml
test:
  - name: test
    runner: go://generic-test-runner
    command: ./test.sh
    envFile: .env.test
```

### Inline Environment Variables

Set environment variables directly:

```yaml
test:
  - name: test
    runner: go://generic-test-runner
    command: ./test.sh
    env:
      DEBUG: "true"
      LOG_LEVEL: "verbose"
```

## Implementation Details

- Executes command via exec.Command
- Captures stdout, stderr, exit code
- Measures execution duration
- Generates UUID for report ID
- Returns TestReport regardless of pass/fail
- Supports working directory configuration
- Supports environment file loading

## See Also

- [Generic Test Runner Configuration Schema](schema.md)
- [go-test MCP Server](../../go-test/docs/usage.md)
- [go-lint MCP Server](../../go-lint/docs/usage.md)
