# CI-Orchestrator Configuration Schema

## Overview

This document describes the configuration options for `ci-orchestrator` in `forge.yaml`.

**Status: PLANNED - NOT YET IMPLEMENTED**

The configuration schema is not yet defined. This document outlines the planned structure.

## Current Status

ci-orchestrator is a placeholder for future CI/CD pipeline orchestration. The configuration schema will be defined during implementation.

## Planned Configuration

### Pipeline Definition (Planned)

```yaml
# Planned configuration structure
pipelines:
  - name: build-and-test
    description: "Build and test pipeline"
    stages:
      - name: build
        steps:
          - command: forge build all
            timeout: 10m
      - name: test
        steps:
          - command: forge test run unit
          - command: forge test run integration
```

### Planned Fields

#### Pipeline

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Pipeline identifier. |
| `description` | string | Human-readable description. |
| `stages` | []Stage | Ordered list of stages. |
| `triggers` | []Trigger | Event triggers (planned). |

#### Stage

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Stage identifier. |
| `steps` | []Step | Steps to execute. |
| `parallel` | []string | Parallel commands (alternative to steps). |
| `dependsOn` | []string | Stage dependencies. |
| `condition` | string | Execution condition. |
| `environment` | string | Target environment. |
| `approval` | string | Approval requirements. |

#### Step

| Field | Type | Description |
|-------|------|-------------|
| `command` | string | Command to execute. |
| `timeout` | duration | Execution timeout. |
| `retry` | RetryConfig | Retry configuration. |
| `env` | map[string]string | Environment variables. |

## Examples (Planned)

### Simple Build Pipeline

```yaml
pipelines:
  - name: build
    stages:
      - name: build
        steps:
          - command: forge build all
```

### Full CI Pipeline

```yaml
pipelines:
  - name: ci
    stages:
      - name: build
        steps:
          - command: forge build all
            timeout: 10m
      - name: lint
        dependsOn: [build]
        steps:
          - command: forge lint all
      - name: test
        dependsOn: [build]
        parallel:
          - forge test run unit
          - forge test run integration
      - name: e2e
        dependsOn: [test]
        steps:
          - command: forge test run e2e
```

### Deployment Pipeline

```yaml
pipelines:
  - name: deploy
    stages:
      - name: build
        steps:
          - command: forge build all
      - name: test
        dependsOn: [build]
        steps:
          - command: forge test run all
      - name: staging
        dependsOn: [test]
        environment: staging
        steps:
          - command: kubectl apply -f manifests/
      - name: production
        dependsOn: [staging]
        environment: production
        approval: required
        steps:
          - command: kubectl apply -f manifests/
```

## Integration with Forge

ci-orchestrator is planned to integrate with forge components:

### Build Integration

```yaml
stages:
  - name: build
    steps:
      - command: forge build all
      # Uses forge's build artifacts
```

### Test Integration

```yaml
stages:
  - name: test
    steps:
      - command: forge test create integration
      - command: forge test run integration
      - command: forge test delete integration
```

### Artifact Tracking

```yaml
stages:
  - name: build
    steps:
      - command: forge build my-app
    outputs:
      - artifact: my-app
        path: build/bin/my-app
```

## Output Artifacts (Planned)

### Metadata

| Key | Description |
|-----|-------------|
| `ci-orchestrator.pipelineName` | Pipeline that was executed |
| `ci-orchestrator.status` | Overall pipeline status |
| `ci-orchestrator.duration` | Total execution time |
| `ci-orchestrator.startedAt` | Start timestamp |
| `ci-orchestrator.completedAt` | Completion timestamp |

### Stage Metadata

| Key | Description |
|-----|-------------|
| `ci-orchestrator.stage.N.name` | Stage name |
| `ci-orchestrator.stage.N.status` | Stage status |
| `ci-orchestrator.stage.N.duration` | Stage duration |

## Notes

- This schema is planned and subject to change
- Implementation will follow forge's existing patterns
- See testenv for orchestration examples

## See Also

- [CI-Orchestrator Usage Guide](usage.md)
- [testenv Configuration](../../testenv/docs/schema.md) - Orchestration example
- [Forge Design Document](../../DESIGN.md)
