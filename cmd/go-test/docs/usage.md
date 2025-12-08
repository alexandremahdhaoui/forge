# Go Test Usage Guide

## Purpose

`go-test` is a forge engine for running Go tests with JUnit XML and coverage reporting. It provides structured test output, coverage calculation, and integration with the forge artifact store.

## Invocation

### MCP Mode

Run as an MCP server:

```bash
go-test --mcp
```

Forge invokes this automatically when using:

```yaml
runner: go://go-test
```

## Available MCP Tools

### `run`

Run Go tests and generate a TestReport.

**Input Schema:**
```json
{
  "stage": "string (required)",
  "name": "string (required)",
  "tmpDir": "string (optional)",
  "testenvEnv": {"key": "value"},
  "envPropagation": {
    "disabled": false,
    "whitelist": ["KUBECONFIG"],
    "blacklist": ["SECRET_VAR"]
  }
}
```

**Output:**
```json
{
  "stage": "string",
  "status": "passed|failed",
  "startTime": "2025-01-06T10:00:00Z",
  "duration": 5.432,
  "testStats": {
    "total": 42,
    "passed": 40,
    "failed": 2,
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
      "name": "unit-tests",
      "tmpDir": ".forge/tmp/test-unit-20250106-abc123"
    }
  }
}
```

### `docs-list`

List all available documentation for go-test.

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

### Unit Tests

Run unit tests for your Go project:

```yaml
test:
  - name: unit
    stage: unit
    runner: go://go-test
```

Run with:

```bash
forge test run unit
```

### Integration Tests with Test Environment

Run integration tests with a Kubernetes test environment:

```yaml
test:
  - name: integration
    stage: integration
    testenv: go://testenv
    runner: go://go-test
```

Run with:

```bash
forge test run integration
```

### E2E Tests

Run end-to-end tests:

```yaml
test:
  - name: e2e
    stage: e2e
    testenv: go://testenv
    runner: go://go-test
```

## Build Tags

Maps stage to Go build tag:
- `stage: unit` -> `-tags=unit`
- `stage: integration` -> `-tags=integration`
- `stage: e2e` -> `-tags=e2e`

Tests must have corresponding build tags:
```go
//go:build unit

package myapp_test
```

## Go Test Execution

Runs:
```bash
go test -tags={stage} \
  -v \
  -coverprofile={tmpDir}/coverage.out \
  -covermode=atomic \
  ./...
```

Generates:
- JUnit XML using go-junit-report
- Coverage profile in {tmpDir}/coverage.out

## Coverage Calculation

Parses coverage.out to compute:
- Percentage coverage
- Covered lines count
- Total lines count

## Environment Propagation

`go-test` supports environment variable propagation from test environments:

### Automatic Propagation

By default, all environment variables from the test environment are propagated to test execution.

### Whitelist Mode

Only propagate specific variables:

```yaml
spec:
  envPropagation:
    whitelist:
      - KUBECONFIG
      - DATABASE_URL
```

### Blacklist Mode

Propagate all except specific variables:

```yaml
spec:
  envPropagation:
    blacklist:
      - SECRET_TOKEN
      - API_KEY
```

### Disable Propagation

Disable all test environment variable propagation:

```yaml
spec:
  envPropagation:
    disabled: true
```

## Artifact Storage

TestReport is automatically stored in artifact store at:
- Path: Defined in forge.yaml `artifactStorePath`
- Files: junit.xml and coverage.out in tmpDir

## Implementation Details

- Uses gotestsum for better output formatting
- Parses JUnit XML to extract test statistics
- Stores report with artifact files for later retrieval
- Returns test report even if tests fail (Status="failed")

## See Also

- [Go Test Configuration Schema](schema.md)
- [go-lint MCP Server](../../go-lint/docs/usage.md)
- [generic-test-runner MCP Server](../../generic-test-runner/docs/usage.md)
