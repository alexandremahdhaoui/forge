# Go Lint Tags Usage Guide

## Purpose

`go-lint-tags` is a forge engine for verifying that all Go test files have valid build tags. It scans the repository for test files and ensures each has one of the required build tags: `unit`, `integration`, or `e2e`. This prevents tests from being silently skipped due to missing tags.

## Invocation

### MCP Mode

Run as an MCP server:

```bash
go-lint-tags --mcp
```

Forge invokes this automatically when using:

```yaml
runner: go://go-lint-tags
```

## Available MCP Tools

### `run`

Verify all test files have valid build tags.

**Input Schema:**
```json
{
  "stage": "string (required)",
  "name": "string (optional)",
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
  "duration": 0.123,
  "testStats": {
    "total": 42,
    "passed": 42,
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
      "stage": "verify-tags",
      "rootDir": "."
    }
  }
}
```

### `docs-list`

List all available documentation for go-lint-tags.

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

### Pre-commit Check

Run as a pre-commit hook to ensure all test files have proper tags:

```yaml
test:
  - name: verify-tags
    stage: verify-tags
    runner: go://go-lint-tags
```

### CI Validation

Include in CI pipeline to enforce build tag conventions:

```bash
forge test run verify-tags
```

## Validation Rules

### Valid Build Tags

The following build tags are accepted:
- `//go:build unit`
- `//go:build integration`
- `//go:build e2e`

### Files Checked

- All `*_test.go` files
- Recursively scans rootDir
- Skips `vendor`, `.git`, `.tmp`, and `node_modules` directories

### Pass Criteria

- All test files have one of the valid build tags
- Tag must appear in the first 5 lines of the file (before `package` declaration)

### Fail Criteria

- Any test file missing a build tag
- Error message lists all files without tags

## Error Message Format

On failure:
```
Found 3 test file(s) without build tags out of 45 total files

Files missing build tags:
  - pkg/myapp/handler_test.go
  - pkg/utils/helper_test.go
  - cmd/server/main_test.go

Test files must have one of these build tags:
  //go:build unit
  //go:build integration
  //go:build e2e
```

## Adding Build Tags

To fix files without build tags, add the appropriate tag at the top of the file:

```go
//go:build unit

package myapp_test

import "testing"

func TestMyFunction(t *testing.T) {
    // ...
}
```

## Implementation Details

- Walks directory tree recursively
- Parses Go files to check for build tags
- Returns detailed error with file list on failure
- Fast execution (no test compilation)
- No coverage tracking

## See Also

- [Go Lint Tags Configuration Schema](schema.md)
- [go-test MCP Server](../../go-test/docs/usage.md)
- [go-lint MCP Server](../../go-lint/docs/usage.md)
