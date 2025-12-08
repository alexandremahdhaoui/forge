# Test Report Configuration Schema

## Overview

This document describes the test report structure and artifact store configuration. `test-report` manages test reports stored by test runners in the artifact store.

## Artifact Store Location

Configure the artifact store path in `forge.yaml`:

```yaml
artifactStorePath: .forge/artifact-store.yaml
```

Or via environment variable:

```bash
export FORGE_ARTIFACT_STORE_PATH=.forge/artifact-store.yaml
```

## Test Report Schema

Each test report in the artifact store follows this structure:

```yaml
testReports:
  - id: "test-unit-unit-20250106-abc123"
    stage: "unit"
    status: "passed"
    startTime: "2025-01-06T10:00:00Z"
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
    tmpDir: ".forge/tmp/test-unit-20250106-abc123"
    errorMessage: ""
```

## Fields

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier for the test report. |
| `stage` | string | Test stage name (e.g., "unit", "integration"). |
| `status` | string | Test result: "passed" or "failed". |
| `startTime` | string | ISO 8601 timestamp of test start. |
| `duration` | float | Test duration in seconds. |

### Test Statistics

| Field | Type | Description |
|-------|------|-------------|
| `testStats.total` | int | Total number of test cases. |
| `testStats.passed` | int | Number of passed tests. |
| `testStats.failed` | int | Number of failed tests. |
| `testStats.skipped` | int | Number of skipped tests. |

### Coverage

| Field | Type | Description |
|-------|------|-------------|
| `coverage.enabled` | bool | Whether coverage was collected. |
| `coverage.percentage` | float | Code coverage percentage (0-100). |

### Artifact Files

| Field | Type | Description |
|-------|------|-------------|
| `files` | []string | List of artifact file names in tmpDir. |
| `tmpDir` | string | Directory containing artifact files. |
| `errorMessage` | string | Error details if status is "failed". |

## Artifact File Types

Common artifact files created by test runners:

| File | Description | Created By |
|------|-------------|------------|
| `junit.xml` | JUnit XML test report | go-test |
| `coverage.out` | Go coverage profile | go-test |
| `test-output.log` | Test output log | various |

## CLI Commands

### List Reports

```bash
# List all
test-report list

# Filter by stage
test-report list --stage=unit
```

### Get Report

```bash
test-report get <REPORT-ID>
```

### Delete Report

```bash
test-report delete <REPORT-ID>
```

Deletes:
- Report entry from artifact store
- All files in tmpDir
- tmpDir directory if empty

## Environment Variables

| Variable | Description |
|----------|-------------|
| `FORGE_ARTIFACT_STORE_PATH` | Override artifact store path |

## Example Artifact Store

```yaml
version: "1.0"
lastUpdated: "2025-01-06T12:00:00Z"
artifacts:
  - name: forge
    type: binary
    location: ./build/bin/forge
    version: abc123
    timestamp: "2025-01-06T10:00:00Z"

testReports:
  - id: test-unit-unit-20250106-abc123
    stage: unit
    status: passed
    startTime: "2025-01-06T10:00:00Z"
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

  - id: test-lint-lint-20250106-def456
    stage: lint
    status: passed
    startTime: "2025-01-06T10:05:00Z"
    duration: 12.1
    testStats:
      total: 0
      passed: 1
      failed: 0
      skipped: 0
    coverage:
      enabled: false
      percentage: 0
    files: []
    tmpDir: ""
```

## See Also

- [Test Report Usage Guide](usage.md)
- [Forge Artifact Store Documentation](../../docs/artifact-store.md)
