# Parallel Test Runner Usage Guide

## Purpose

`parallel-test-runner` is a forge engine that executes multiple test runners in parallel. It aggregates results from all sub-runners and provides a unified TestReport. This is useful for running independent test suites concurrently to reduce total test time.

## Invocation

### MCP Mode

Run as an MCP server:

```bash
parallel-test-runner --mcp
```

Forge invokes this automatically when using:

```yaml
runner: go://parallel-test-runner
```

## Available MCP Tools

### `run`

Execute multiple test runners in parallel and aggregate results.

**Input Schema:**
```json
{
  "stage": "string (required)",
  "name": "string (required)",
  "spec": {
    "primaryCoverageRunner": "string (optional)",
    "runners": [
      {
        "name": "string (required)",
        "engine": "string (required)",
        "spec": {}
      }
    ]
  },
  "tmpDir": "string (optional)",
  "buildDir": "string (optional)",
  "rootDir": "string (optional)"
}
```

**Output:**
```json
{
  "id": "string",
  "stage": "string",
  "status": "passed|failed",
  "startTime": "2025-01-06T10:00:00Z",
  "duration": 5.432,
  "testStats": {
    "total": 100,
    "passed": 95,
    "failed": 5,
    "skipped": 0
  },
  "coverage": {
    "enabled": true,
    "percentage": 85.3
  },
  "errorMessage": "string"
}
```

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "run",
    "arguments": {
      "stage": "unit",
      "name": "unit-parallel",
      "spec": {
        "primaryCoverageRunner": "go-test",
        "runners": [
          {
            "name": "go-test",
            "engine": "go://go-test"
          },
          {
            "name": "lint",
            "engine": "go://go-lint"
          }
        ]
      }
    }
  }
}
```

### `docs-list`

List all available documentation for parallel-test-runner.

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

### Run Tests and Linting in Parallel

```yaml
test:
  - name: unit
    stage: unit
    runner: go://parallel-test-runner
    spec:
      primaryCoverageRunner: go-test
      runners:
        - name: go-test
          engine: go://go-test
        - name: lint
          engine: go://go-lint
        - name: verify-tags
          engine: go://go-lint-tags
```

### Run Multiple Independent Test Suites

```yaml
test:
  - name: full-test
    stage: full
    runner: go://parallel-test-runner
    spec:
      primaryCoverageRunner: unit
      runners:
        - name: unit
          engine: go://go-test
          spec:
            stage: unit
        - name: security
          engine: go://generic-test-runner
          spec:
            command: gosec
            args: ["./..."]
```

## Result Aggregation

### Test Statistics

Test statistics are summed from all runners:

- `total` = sum of all runners' total
- `passed` = sum of all runners' passed
- `failed` = sum of all runners' failed
- `skipped` = sum of all runners' skipped

### Coverage

Coverage is selected from the `primaryCoverageRunner` only:

- If `primaryCoverageRunner` is specified and that runner exists, use its coverage
- If not specified or runner not found, `coverage.enabled` is false
- Coverage is NOT averaged across runners

### Status Determination

- **Any failure** = overall status is "failed"
- **All passed** = overall status is "passed"
- Runner execution errors also result in "failed" status

## Parallel Execution

All configured runners execute concurrently:

1. Each runner is launched in a separate goroutine
2. Results are collected as runners complete
3. Final aggregation happens after all runners finish
4. Total duration is the wall-clock time (not sum of runner durations)

## Error Handling

If any runner fails:
- Its results are still included in aggregation
- Overall status becomes "failed"
- Error message includes details of failing runners

If a runner fails to execute (MCP call error):
- Overall status becomes "failed"
- Error message includes "[runner-name] MCP call failed: ..."

## Implementation Details

- Uses MCP caller to invoke sub-runners
- Resolves engine URIs to commands (go:// -> binary path)
- Passes through testenv information (artifactFiles, metadata, env)
- Collects results via channel with WaitGroup synchronization

## See Also

- [Parallel Test Runner Configuration Schema](schema.md)
- [go-test MCP Server](../../go-test/docs/usage.md)
- [go-lint MCP Server](../../go-lint/docs/usage.md)
