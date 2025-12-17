# test-report

**Manage test reports stored in the artifact store.**

> "After running tests, I need to inspect results, compare runs, and clean up old reports. test-report gives me CLI and MCP access to all my test history."

## What problem does test-report solve?

Test runners create reports, but you need a way to retrieve, list, and delete them. test-report provides management operations for TestReport artifacts without running any tests.

## How do I use test-report?

List all reports:
```bash
test-report list
```

List reports for a stage:
```bash
test-report list --stage=unit
```

Get report details:
```bash
test-report get <REPORT-ID>
```

Delete a report:
```bash
test-report delete <REPORT-ID>
```

## What operations are available?

| Operation | Description |
|-----------|-------------|
| `list` | List all test reports, optionally filtered by stage |
| `get` | Get full details of a specific report by ID |
| `delete` | Delete a report and its artifact files |

## What does list output look like?

```
ID                                  STAGE   STATUS   DURATION  TOTAL  PASSED  FAILED
test-unit-unit-20250106-abc123      unit    passed   5.43s     42     42      0
test-lint-lint-20250106-def456      lint    passed   12.1s     1      1       0
test-e2e-e2e-20250106-ghi789        e2e     failed   45.2s     15     13      2
```

## What does get output look like?

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
```

## What does delete remove?

- TestReport entry from artifact store
- Associated artifact files (junit.xml, coverage.out, etc.)
- Empty temporary directory

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../../MCP.md) - MCP tool documentation
