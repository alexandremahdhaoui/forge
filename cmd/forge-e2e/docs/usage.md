# forge-e2e

**Run comprehensive end-to-end tests for forge build and test orchestration.**

> "I needed confidence that forge's build system, test environments, and MCP integration all work together. forge-e2e runs categorized tests across the entire system and gives me structured reports showing exactly what passed or failed."

## What problem does forge-e2e solve?

Validating forge requires testing builds, test environments, artifact stores, and MCP integration together. forge-e2e provides a structured test suite with categories, parallel execution, and detailed reporting.

## How do I use forge-e2e?

Add to `forge.yaml`:

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

### Filtering tests

```bash
# Run only build tests
TEST_CATEGORY=build forge test run e2e

# Run tests matching a pattern
TEST_NAME_PATTERN=environment forge test run e2e
```

## What test categories exist?

| Category | Tests |
|----------|-------|
| `build` | Build system commands |
| `testenv` | Test environment lifecycle (create, list, get, delete) |
| `test-runner` | Test runner integration |
| `prompt` | Prompt system |
| `artifact-store` | Artifact store validation |
| `system` | Version, help commands |
| `error-handling` | Error scenarios |
| `cleanup` | Resource cleanup |
| `mcp` | MCP integration |
| `performance` | Performance benchmarks |

## What environment variables are available?

| Variable | Description |
|----------|-------------|
| `TEST_CATEGORY` | Filter by category |
| `TEST_NAME_PATTERN` | Filter by name (case-insensitive) |
| `KIND_BINARY` | Path to kind binary (for testenv tests) |
| `CONTAINER_ENGINE` | Container runtime (docker/podman) |
| `SKIP_CLEANUP` | Keep test resources for debugging |

## What output do I get?

Test progress on stderr:

```
=== Forge E2E Test Suite ===
Running 25 tests across 8 categories

=== Category: build (5 tests) ===
  forge build                        PASSED (1.23s)
  forge build specific artifact      PASSED (0.89s)

=== Test Summary ===
Status: passed
Total: 25, Passed: 25, Failed: 0
Duration: 45.67s
```

Structured JSON test report on stdout (in MCP mode).

Exit codes: `0` = all passed, `1` = failures.

## How does parallel execution work?

Tests marked `Parallel: true` run concurrently (read-only operations, isolated resources). Tests marked `Parallel: false` run sequentially (shared state, resource creation/destruction).

Some tests share a test environment created during setup and cleaned up during teardown.

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../MCP.md) - MCP tool documentation
- [forge-test-usage.md](../../../docs/forge-test-usage.md) - Forge test system
