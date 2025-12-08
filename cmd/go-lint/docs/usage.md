# Go Lint Usage Guide

## Purpose

`go-lint` is a forge engine for running golangci-lint on Go code. It provides structured test reports with pass/fail status and automatic fix application.

## Invocation

### MCP Mode

Run as an MCP server:

```bash
go-lint --mcp
```

Forge invokes this automatically when using:

```yaml
runner: go://go-lint
```

## Available MCP Tools

### `run`

Run golangci-lint and generate TestReport.

**Input Schema:**
```json
{
  "id": "string (optional)",
  "stage": "string (required)",
  "name": "string (required)",
  "tmpDir": "string (optional)",
  "buildDir": "string (optional)",
  "rootDir": "string (optional)"
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
    "total": 1,
    "passed": 1,
    "failed": 0,
    "skipped": 0
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
      "stage": "lint",
      "name": "lint-check",
      "id": "lint-20250106-abc123"
    }
  }
}
```

### `docs-list`

List all available documentation for go-lint.

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

### Basic Lint Check

```yaml
test:
  - name: lint
    runner: go://go-lint
```

Run with:

```bash
forge test run lint
```

### As Part of test-all

```bash
forge test-all
```

## Lint Execution

The engine runs:
```bash
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@{version} run --fix
```

### Default Version

Uses `v2.6.0` by default. Override with `GOLANGCI_LINT_VERSION` environment variable.

### Auto-Fix

The `--fix` flag is applied automatically, so fixable issues are corrected in place.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GOLANGCI_LINT_VERSION` | Version of golangci-lint to use | `v2.6.0` |

## Lint Configuration

Uses `.golangci.yml` in project root if present. Falls back to golangci-lint defaults.

Example `.golangci.yml`:
```yaml
linters:
  enable:
    - gofmt
    - govet
    - errcheck
    - staticcheck
    - unused
  disable:
    - depguard

linters-settings:
  govet:
    check-shadowing: true
  gofmt:
    simplify: true

issues:
  exclude-use-default: false
  max-issues-per-linter: 0
  max-same-issues: 0
```

## Exit Behavior

- **Exit code 0**: Linting passed (no issues)
- **Exit code 1**: Linting failed (issues found)

## Implementation Details

- Runs golangci-lint via `go run` (no global install required)
- Applies fixes automatically where possible
- Returns structured test report for artifact store integration
- Lint output written to stderr
- No coverage tracking (Coverage.Percentage = 0)

## See Also

- [Go Lint Configuration Schema](schema.md)
- [go-test MCP Server](../../go-test/docs/usage.md)
- [go-lint-tags MCP Server](../../go-lint-tags/docs/usage.md)
