# Test Report Usage Guide

## Purpose

`test-report` is a forge engine for managing test reports. It provides operations to get, list, and delete test reports stored in the artifact store. Test reports are created by test runners (go-test, generic-test-runner, etc.) and this engine manages their lifecycle.

## Invocation

### CLI Mode

Use as a standalone command:

```bash
# List all test reports
test-report list

# List reports for a specific stage
test-report list --stage=unit

# Get details about a specific report
test-report get <REPORT-ID>

# Delete a test report and its artifacts
test-report delete <REPORT-ID>

# Show version
test-report version
```

### MCP Mode

Run as an MCP server:

```bash
test-report --mcp
```

## Available MCP Tools

### `create`

No-op operation for interface compatibility. Test reports are created by test runners, not by this engine.

**Input Schema:**
```json
{
  "stage": "string (required)"
}
```

**Output:**
```json
{
  "message": "No-op: test reports for stage 'unit' are created by test runners during execution"
}
```

### `get`

Get test report details by ID.

**Input Schema:**
```json
{
  "reportID": "string (required)"
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
    "total": 42,
    "passed": 40,
    "failed": 2,
    "skipped": 0
  },
  "coverage": {
    "enabled": true,
    "percentage": 85.3
  },
  "artifactFiles": ["junit.xml", "coverage.out"],
  "errorMessage": "string"
}
```

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "get",
    "arguments": {
      "reportID": "test-unit-unit-20250106-abc123"
    }
  }
}
```

### `list`

List test reports, optionally filtered by stage.

**Input Schema:**
```json
{
  "stage": "string (optional)"
}
```

**Output:**
```json
[
  {
    "id": "string",
    "stage": "string",
    "status": "passed|failed",
    "startTime": "2025-01-06T10:00:00Z",
    "duration": 5.432,
    "testStats": {...}
  }
]
```

**Example - List all:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "list",
    "arguments": {}
  }
}
```

**Example - Filter by stage:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "list",
    "arguments": {
      "stage": "unit"
    }
  }
}
```

### `delete`

Delete a test report and its artifact files.

**Input Schema:**
```json
{
  "reportID": "string (required)"
}
```

**Output:**
```json
{
  "success": true,
  "message": "Deleted test report: test-unit-unit-20250106-abc123"
}
```

**What It Deletes:**
- TestReport entry from artifact store
- Associated artifact files (junit.xml, coverage.out, etc.)
- Temporary directory if empty

### `docs-list`

List all available documentation for test-report.

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

## CLI Examples

### List All Reports

```bash
test-report list
```

Output:
```
ID                                  STAGE       STATUS   DURATION  TOTAL  PASSED  FAILED
test-unit-unit-20250106-abc123      unit        passed   5.43s     42     42      0
test-lint-lint-20250106-def456      lint        passed   12.1s     0      1       0
test-e2e-e2e-20250106-ghi789        e2e         failed   45.2s     15     13      2
```

### List Reports by Stage

```bash
test-report list --stage=unit
```

### Get Report Details

```bash
test-report get test-unit-unit-20250106-abc123
```

Output:
```yaml
id: test-unit-unit-20250106-abc123
stage: unit
status: passed
startTime: 2025-01-06T10:00:00Z
duration: 5.432
testStats:
  total: 42
  passed: 42
  failed: 0
  skipped: 0
coverage:
  enabled: true
  percentage: 85.3
files:
  - junit.xml
  - coverage.out
tmpDir: .forge/tmp/test-unit-20250106-abc123
```

### Delete a Report

```bash
test-report delete test-unit-unit-20250106-abc123
```

## Storage Location

Reports are stored in the artifact store defined in forge.yaml:

```yaml
artifactStorePath: .forge/artifact-store.yaml
```

## Implementation Details

- Reads/writes artifact store directly
- Deletes artifact files from filesystem on delete
- Returns TestReport objects as defined in pkg/forge
- No test execution - only management operations

## See Also

- [Test Report Configuration Schema](schema.md)
- [go-test MCP Server](../../go-test/docs/usage.md)
- [generic-test-runner MCP Server](../../generic-test-runner/docs/usage.md)
