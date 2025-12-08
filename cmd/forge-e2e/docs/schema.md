# Forge E2E Configuration Schema

## Overview

This document describes the configuration options for `forge-e2e` in `forge.yaml`. The forge-e2e engine runs comprehensive end-to-end tests for the forge system.

## Basic Configuration

```yaml
test:
  - name: e2e
    runner: go://forge-e2e
```

## Configuration Options

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | The name of the e2e test stage. |
| `runner` | string | Must be `go://forge-e2e` to use this runner. |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `stage` | string | Same as `name` | The test stage name. |
| `tags` | []string | `[]` | Build tags for test selection. |

## Examples

### Minimal Configuration

```yaml
test:
  - name: e2e
    runner: go://forge-e2e
```

### With Tags

```yaml
test:
  - name: e2e
    runner: go://forge-e2e
    tags: ["e2e"]
```

## Environment Variables

Control test execution via environment variables:

### Category Filter

Run only tests in a specific category:

```bash
TEST_CATEGORY=build forge test run e2e
```

### Name Pattern Filter

Run tests matching a pattern (case-insensitive):

```bash
TEST_NAME_PATTERN=environment forge test run e2e
```

### Skip Cleanup

Keep test resources for debugging:

```bash
SKIP_CLEANUP=1 forge test run e2e
```

### Prerequisites

```bash
# Required for testenv tests
export KIND_BINARY=kind

# Required for container tests
export CONTAINER_ENGINE=docker
```

## Test Categories

### build

Build system tests:
- `forge build` - Build all artifacts
- `forge build specific artifact` - Build named artifact
- `forge build container` - Build container images
- `forge build format` - Code formatting
- `incremental build` - Unchanged rebuild

### testenv

Test environment lifecycle:
- `test environment create` - Create environment
- `test environment list` - List environments
- `test environment get` - Get environment details
- `test environment get JSON` - Get as JSON
- `test environment delete` - Delete environment

### test-runner

Test runner integration:
- `forge test unit run` - Run unit tests
- `forge test integration run` - Run with testenv
- `forge test lint run` - Run linter
- `forge test verify-tags run` - Verify build tags

### system

System commands:
- `forge version` - Show version info
- `forge help` - Show help text
- `forge no args` - No arguments error

### mcp

MCP integration:
- `MCP server mode` - Start MCP server
- `MCP run tool call` - Call run tool
- `MCP error propagation` - Error handling

## Test Execution Order

Tests run by category in this order:
1. build
2. testenv
3. test-runner
4. prompt
5. artifact-store
6. system
7. error-handling
8. cleanup
9. mcp
10. performance

Within each category:
1. Sequential tests first
2. Parallel tests concurrently

## Generated Artifacts

Each e2e run creates a TestReport:

```yaml
testReports:
  - id: "test-e2e-e2e-20250106-abc123"
    stage: "e2e"
    status: "passed"
    startTime: "2025-01-06T10:00:00Z"
    duration: 123.45
    testStats:
      total: 42
      passed: 40
      failed: 2
      skipped: 0
    coverage:
      enabled: false
      percentage: 0
```

## Default Behavior

When running forge-e2e:

- Runs all registered tests
- Creates shared test environment if needed
- Executes tests by category
- Cleans up resources on completion
- Returns detailed test report

## See Also

- [Forge E2E Usage Guide](usage.md)
- [Forge Test Documentation](../../docs/forge-test-usage.md)
