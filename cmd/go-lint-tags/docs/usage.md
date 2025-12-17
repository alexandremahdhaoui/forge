# go-lint-tags

**Verify all Go test files have valid build tags.**

> "Tests were silently skipped because someone forgot to add a build tag. go-lint-tags catches this immediately and tells me exactly which files need fixing."

## What problem does go-lint-tags solve?

Go test files without build tags run in ALL test stages or get silently skipped. go-lint-tags scans your codebase and fails if any test file is missing a `unit`, `integration`, or `e2e` tag.

## How do I use go-lint-tags?

```yaml
test:
  - name: verify-tags
    stage: verify-tags
    runner: go://go-lint-tags
```

Run with:

```bash
forge test run verify-tags
```

## What tags are valid?

Test files must have one of these build tags in the first 5 lines:

```go
//go:build unit
//go:build integration
//go:build e2e
```

## What directories are skipped?

- `vendor/`
- `.git/`
- `.tmp/`
- `node_modules/`

## What output does it produce?

On success:
```json
{
  "stage": "verify-tags",
  "status": "passed",
  "testStats": {
    "total": 45,
    "passed": 45,
    "failed": 0
  }
}
```

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

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../../MCP.md) - MCP tool documentation
