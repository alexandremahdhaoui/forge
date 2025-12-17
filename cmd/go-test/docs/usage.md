# go-test

**Run Go tests with JUnit XML reports and coverage tracking.**

> "I needed test reports that integrate with CI dashboards and coverage tracking that actually works. go-test gives me structured output I can parse and track over time."

## What problem does go-test solve?

Go's built-in test runner outputs plain text that's hard to parse for CI systems. go-test wraps `go test` to produce JUnit XML reports, calculate coverage percentages, and return structured TestReport data for artifact storage.

## How do I use go-test?

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

## What configuration options are available?

| Option | Description |
|--------|-------------|
| `stage` | Test stage (unit, integration, e2e) - maps to Go build tag |
| `testenvEnv` | Environment variables from test environment |
| `envPropagation.disabled` | Disable all testenv environment propagation |
| `envPropagation.whitelist` | Only propagate these variables from testenv |
| `envPropagation.blacklist` | Propagate all except these variables from testenv |

## How do build tags work?

The stage maps directly to Go build tags:

- `stage: unit` runs tests with `//go:build unit`
- `stage: integration` runs tests with `//go:build integration`
- `stage: e2e` runs tests with `//go:build e2e`

Your test files need the corresponding tag:

```go
//go:build unit

package myapp_test
```

## What output does it produce?

```json
{
  "stage": "unit",
  "status": "passed",
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
  }
}
```

Artifacts generated:
- `junit.xml` - JUnit XML report for CI integration
- `coverage.out` - Go coverage profile

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../../MCP.md) - MCP tool documentation
