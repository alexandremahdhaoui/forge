# parallel-test-runner

**Run multiple test runners concurrently with aggregated results.**

> "My unit tests, linting, and tag verification are independent. parallel-test-runner runs them all at once, cutting my CI time in half while giving me a single aggregated report."

## What problem does parallel-test-runner solve?

Running independent tests sequentially wastes time. parallel-test-runner executes multiple test engines concurrently and aggregates their results into a single TestReport with combined statistics.

## How do I use parallel-test-runner?

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

Run with:

```bash
forge test run unit
```

## What configuration options are available?

| Option | Description |
|--------|-------------|
| `primaryCoverageRunner` | Name of runner to use for coverage percentage |
| `runners` | Array of test runners to execute in parallel |
| `runners[].name` | Unique name for the runner |
| `runners[].engine` | Engine URI (e.g., `go://go-test`) |
| `runners[].spec` | Engine-specific configuration |

## How are results aggregated?

**Test statistics** are summed:
- `total` = sum of all runners' total
- `passed` = sum of all runners' passed
- `failed` = sum of all runners' failed

**Coverage** comes from `primaryCoverageRunner` only (not averaged).

**Status**:
- Any failure = overall "failed"
- All passed = overall "passed"

## What output does it produce?

```json
{
  "stage": "unit",
  "status": "passed",
  "duration": 8.5,
  "testStats": {
    "total": 50,
    "passed": 50,
    "failed": 0,
    "skipped": 0
  },
  "coverage": {
    "enabled": true,
    "percentage": 85.3
  }
}
```

Duration is wall-clock time, not sum of individual runner times.

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../../MCP.md) - MCP tool documentation
