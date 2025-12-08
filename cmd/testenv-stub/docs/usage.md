# Testenv-Stub Usage Guide

## Purpose

`testenv-stub` is a lightweight no-op testenv subengine for fast e2e testing of the testenv infrastructure itself. It creates mock metadata without provisioning real resources like Kind clusters, allowing rapid testing of the testenv create/list/get/delete workflow.

## Invocation

### MCP Mode

Run as an MCP server:

```bash
testenv-stub --mcp
```

This is typically called automatically by the testenv orchestrator.

Forge invokes this automatically when configured in testenv:

```yaml
engines:
  - alias: fast-testenv
    type: testenv
    testenv:
      - engine: go://testenv-stub
```

## Available MCP Tools

### `create`

Create a stub test environment (no-op with mock metadata).

**Input Schema:**
```json
{
  "testID": "string (required)",
  "stage": "string (required)",
  "tmpDir": "string (required)"
}
```

**Output:**
```json
{
  "testID": "string",
  "files": {
    "testenv-stub.marker": "stub-marker.txt"
  },
  "metadata": {
    "testenv-stub.createdAt": "2025-01-06T10:00:00Z",
    "testenv-stub.testID": "test-unit-20250106-abc123",
    "testenv-stub.stage": "unit"
  },
  "env": {
    "TESTENV_STUB_ACTIVE": "true"
  },
  "managedResources": [
    "/abs/path/to/tmpDir/stub-marker.txt"
  ]
}
```

### `delete`

Delete a stub test environment (no-op).

**Input Schema:**
```json
{
  "testID": "string (required)"
}
```

**Output:**
```json
{
  "success": true,
  "message": "Deleted stub test environment: test-unit-20250106-abc123"
}
```

### `docs-list`

List all available documentation for testenv-stub.

### `docs-get`

Get a specific documentation by name.

### `docs-validate`

Validate documentation completeness.

## Common Use Cases

### Testing Testenv Infrastructure

Use testenv-stub to test the testenv create/list/get/delete workflow without the overhead of real resource creation:

```yaml
engines:
  - alias: stub-testenv
    type: testenv
    testenv:
      - engine: go://testenv-stub

test:
  - name: e2e
    stage: e2e
    testenv: stub-testenv
    runner: go://go-test
```

### Fast CI Testing

For CI pipelines where real Kind clusters are not needed:

```yaml
engines:
  - alias: fast-ci-env
    type: testenv
    testenv:
      - engine: go://testenv-stub

test:
  - name: unit
    stage: unit
    testenv: fast-ci-env
    runner: go://go-test
```

### Combined with Real Subengines

Use stub alongside real subengines for testing specific components:

```yaml
engines:
  - alias: partial-testenv
    type: testenv
    testenv:
      - engine: go://testenv-stub  # Instead of testenv-kind
      - engine: go://testenv-helm-install  # Will fail without real cluster
        spec:
          charts: []  # Empty charts for testing
```

## Implementation Details

- Creates a single marker file in tmpDir
- Returns mock metadata with timestamps
- Delete is a no-op (tmpDir cleanup handled by orchestrator)
- Executes in milliseconds (no external resources)

## Environment Variables

| Variable | Value | Description |
|----------|-------|-------------|
| `TESTENV_STUB_ACTIVE` | `true` | Indicates stub environment is active |

## When to Use

| Scenario | Use testenv-stub? |
|----------|-------------------|
| Testing testenv orchestration | Yes |
| Fast unit tests | Yes |
| Integration tests needing Kubernetes | No (use testenv-kind) |
| Tests needing container registry | No (use testenv-lcr) |
| CI pipeline validation | Yes |

## See Also

- [Testenv-Stub Configuration Schema](schema.md)
- [testenv MCP Server](../../testenv/docs/usage.md)
- [testenv-kind MCP Server](../../testenv-kind/docs/usage.md)
