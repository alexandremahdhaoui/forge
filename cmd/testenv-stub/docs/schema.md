# Testenv-Stub Configuration Schema

## Overview

This document describes the configuration options for `testenv-stub` in `forge.yaml`. The testenv-stub engine is a lightweight no-op subengine used for testing the testenv infrastructure without creating real resources.

## Basic Configuration

```yaml
engines:
  - alias: stub-testenv
    type: testenv
    testenv:
      - engine: go://testenv-stub
```

## Configuration Options

### Engine Entry Fields

| Field | Type | Description |
|-------|------|-------------|
| `engine` | string | Must be `go://testenv-stub`. |
| `spec` | object | Engine-specific configuration (currently unused). |

**Note:** testenv-stub does not require any spec configuration. It operates as a pure no-op subengine.

## Examples

### Minimal Configuration

```yaml
engines:
  - alias: stub-env
    type: testenv
    testenv:
      - engine: go://testenv-stub
```

### For E2E Testing

```yaml
engines:
  - alias: e2e-stub-env
    type: testenv
    testenv:
      - engine: go://testenv-stub

test:
  - name: e2e
    stage: e2e
    testenv: e2e-stub-env
    runner: go://go-test
```

### Fast Unit Test Environment

```yaml
engines:
  - alias: fast-unit-env
    type: testenv
    testenv:
      - engine: go://testenv-stub

test:
  - name: unit
    stage: unit
    testenv: fast-unit-env
    runner: go://go-test
```

## Output Artifacts

### Files

| Key | Relative Path | Description |
|-----|---------------|-------------|
| `testenv-stub.marker` | `stub-marker.txt` | Marker file indicating stub creation |

### Metadata

| Key | Description | Example |
|-----|-------------|---------|
| `testenv-stub.createdAt` | Creation timestamp | `2025-01-06T10:00:00Z` |
| `testenv-stub.testID` | Test environment ID | `test-unit-20250106-abc123` |
| `testenv-stub.stage` | Test stage name | `unit` |

### Environment Variables

| Variable | Value | Description |
|----------|-------|-------------|
| `TESTENV_STUB_ACTIVE` | `true` | Indicates stub environment is active |

## Behavior

### Create

1. Generates a marker file in tmpDir: `stub-marker.txt`
2. Returns mock metadata with creation timestamp
3. Sets `TESTENV_STUB_ACTIVE=true` in environment
4. Completes in milliseconds (no external resources)

### Delete

1. Logs deletion message (no actual cleanup)
2. tmpDir cleanup handled by testenv orchestrator
3. Completes instantly

## Use Cases

### Testing Testenv Orchestration

testenv-stub allows testing the full testenv lifecycle without provisioning real infrastructure:

```bash
# Create stub environment
forge test create e2e
# Expected: Fast creation with mock metadata

# List environments
forge test list e2e
# Expected: Shows stub environment

# Get environment details
forge test get e2e <testID>
# Expected: Shows stub metadata

# Delete environment
forge test delete e2e <testID>
# Expected: Fast deletion
```

### CI Pipeline Validation

Use in CI to validate forge configuration without resource overhead:

```yaml
# .github/workflows/ci.yml
- name: Validate forge test workflow
  run: |
    forge test create unit
    forge test list unit
    forge test delete unit $(forge test list unit -o json | jq -r '.[0].id')
```

## Comparison with Other Subengines

| Subengine | Resources Created | Time | Use Case |
|-----------|-------------------|------|----------|
| testenv-stub | None | ~1ms | Testing infrastructure |
| testenv-kind | Kind cluster | ~30s | Kubernetes testing |
| testenv-lcr | Container registry | ~60s | Image push testing |
| testenv-helm-install | Helm releases | ~varies | Chart deployment testing |

## Notes

- No external dependencies required
- No cleanup necessary (orchestrator handles tmpDir)
- Safe to run in parallel (isolated tmpDir per environment)
- Useful for validating forge configuration syntax

## See Also

- [Testenv-Stub Usage Guide](usage.md)
- [testenv Configuration](../../testenv/docs/schema.md)
- [testenv-kind Configuration](../../testenv-kind/docs/schema.md)
