# go-lint

**Run golangci-lint with automatic fixes and structured reporting.**

> "I wanted linting as part of my test pipeline with the same TestReport format as my unit tests. go-lint integrates golangci-lint seamlessly and even auto-fixes issues."

## What problem does go-lint solve?

Running golangci-lint separately from your test pipeline means inconsistent reporting and manual fix steps. go-lint runs golangci-lint with `--fix`, returns a structured TestReport, and integrates cleanly with `forge test-all`.

## How do I use go-lint?

```yaml
test:
  - name: lint
    runner: go://go-lint
```

Run with:

```bash
forge test run lint
```

## What configuration options are available?

| Option | Description |
|--------|-------------|
| `stage` | Test stage name (typically "lint") |
| `rootDir` | Root directory for linting (defaults to project root) |

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `GOLANGCI_LINT_VERSION` | Version of golangci-lint to use | `v2.6.0` |

## How do I configure linters?

Place a `.golangci.yml` in your project root:

```yaml
linters:
  enable:
    - gofmt
    - govet
    - errcheck
    - staticcheck
  disable:
    - depguard

issues:
  max-issues-per-linter: 0
  max-same-issues: 0
```

## What output does it produce?

```json
{
  "stage": "lint",
  "status": "passed",
  "duration": 12.1,
  "testStats": {
    "total": 1,
    "passed": 1,
    "failed": 0,
    "skipped": 0
  }
}
```

- Exit code 0 = passed (no issues or all fixed)
- Exit code 1 = failed (unfixable issues found)
- The `--fix` flag is always applied

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../../MCP.md) - MCP tool documentation
