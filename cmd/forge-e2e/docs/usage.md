# Forge E2E Usage Guide

## Purpose

`forge-e2e` is a forge engine for running comprehensive end-to-end tests of the forge build and test orchestration system. Tests are categorized by functionality and can be filtered and run in parallel.

## Invocation

### CLI Mode

Run directly as a standalone command:

```bash
# Run all e2e tests
forge-e2e e2e test-run

# Run with category filter
TEST_CATEGORY=build forge-e2e e2e test-run

# Run with name pattern filter
TEST_NAME_PATTERN=environment forge-e2e e2e test-run
```

### MCP Mode

Run as an MCP server:

```bash
forge-e2e --mcp
```

Forge invokes this via:

```yaml
runner: go://forge-e2e
```

## Available MCP Tools

### `run`

Execute end-to-end tests and generate structured TestReport.

**Input Schema:**
```json
{
  "stage": "string (required)",
  "name": "string (required)",
  "id": "string (optional)",
  "tmpDir": "string (optional)",
  "buildDir": "string (optional)",
  "rootDir": "string (optional)"
}
```

**Output:**
```json
{
  "status": "passed|failed",
  "errorMessage": "string",
  "duration": 123.45,
  "total": 42,
  "passed": 40,
  "failed": 2,
  "skipped": 0,
  "results": [
    {
      "name": "test name",
      "category": "build",
      "status": "passed",
      "duration": 1.23,
      "error": ""
    }
  ],
  "categories": {
    "build": {
      "total": 5,
      "passed": 5,
      "failed": 0,
      "skipped": 0,
      "duration": 10.5
    }
  }
}
```

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "run",
    "arguments": {
      "stage": "e2e",
      "name": "test-20250106"
    }
  }
}
```

### `docs-list`

List all available documentation for forge-e2e.

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

## Test Categories

Tests are organized into the following categories:

| Category | Description |
|----------|-------------|
| `build` | Build system tests (forge build commands) |
| `testenv` | Test environment lifecycle (create, list, get, delete) |
| `test-runner` | Test runner integration (unit, integration, lint) |
| `prompt` | Prompt system tests (list, get prompts) |
| `artifact-store` | Artifact store validation |
| `system` | System commands (version, help) |
| `error-handling` | Error handling tests |
| `cleanup` | Resource cleanup tests |
| `mcp` | MCP integration tests |
| `performance` | Performance tests |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `TEST_CATEGORY` | Filter tests by category |
| `TEST_NAME_PATTERN` | Filter tests by name (case-insensitive substring) |
| `KIND_BINARY` | Path to kind binary (required for testenv tests) |
| `CONTAINER_ENGINE` | Container runtime (docker/podman) |
| `SKIP_CLEANUP` | Keep test resources for debugging |

## Parallel Execution Strategy

### Parallel Tests

Tests marked `Parallel: true` run concurrently:
- Read-only operations
- Independent test suites
- Tests with isolated resources

### Sequential Tests

Tests marked `Parallel: false` run sequentially:
- Tests modifying shared state
- Tests using shared test environments
- Tests that create/destroy resources

## Shared Test Environments

Some tests use a shared test environment:
- Created once during test suite setup
- Uses `e2e-stub` stage (fast, no real resources)
- Reused across multiple tests
- Cleaned up during test suite teardown

## Test Output

### stderr
Test progress, status updates, and summary:
```
=== Forge E2E Test Suite ===
Running 25 tests across 8 categories

=== Category: build (5 tests) ===
  forge build                        PASSED (1.23s)
  forge build specific artifact      PASSED (0.89s)

=== Test Summary ===
Status: passed
Total: 25
Passed: 25
Failed: 0
Duration: 45.67s
```

### stdout
Structured JSON test report (in MCP mode).

### Exit Code
- `0`: All tests passed
- `1`: One or more tests failed

## Integration with Forge

```yaml
test:
  - name: e2e
    runner: go://forge-e2e
    tags: ["e2e"]
```

Run with:
```bash
forge test run e2e
```

## Implementation Details

- Tests are registered via `registerAllTests()`
- Each test is a function receiving `*TestSuite` context
- Test results collected via channel with WaitGroup
- Shared test environment managed by `suiteEnvironment`

## See Also

- [Forge E2E Configuration Schema](schema.md)
- [go-test MCP Server](../../go-test/docs/usage.md)
- [Forge Test Documentation](../../docs/forge-test-usage.md)
