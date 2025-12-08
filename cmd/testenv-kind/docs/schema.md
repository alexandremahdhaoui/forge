# Testenv-Kind Configuration Schema

## Overview

This document describes the configuration options for `testenv-kind` in `forge.yaml`. The testenv-kind engine creates Kind (Kubernetes in Docker) clusters for test environments.

## Basic Configuration

```yaml
engines:
  - alias: my-testenv
    type: testenv
    testenv:
      - engine: go://testenv-kind
```

## Configuration Options

### Engine Entry Fields

| Field | Type | Description |
|-------|------|-------------|
| `engine` | string | Must be `go://testenv-kind`. |
| `spec` | object | Engine-specific configuration (currently unused). |

## Global Kind Configuration

testenv-kind reads from the `kindenv` section in `forge.yaml`:

```yaml
kindenv:
  kubeconfigPath: .forge/kubeconfig  # Ignored in MCP mode, uses tmpDir
```

**Note:** The `kubeconfigPath` is ignored in MCP mode. Each test environment uses its own tmpDir for kubeconfig isolation.

## Examples

### Minimal Configuration

```yaml
engines:
  - alias: k8s-env
    type: testenv
    testenv:
      - engine: go://testenv-kind
```

### With Other Subengines

```yaml
engines:
  - alias: integration-env
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-lcr
        spec:
          enabled: true
      - engine: go://testenv-helm-install
        spec:
          charts:
            - name: cert-manager
              sourceType: helm-repo
              url: https://charts.jetstack.io
              chartName: cert-manager
              version: v1.13.0
              namespace: cert-manager
              createNamespace: true
```

## Output Artifacts

### Files

| Key | Relative Path | Description |
|-----|---------------|-------------|
| `testenv-kind.kubeconfig` | `kubeconfig` | Kubernetes configuration file |

### Metadata

| Key | Description | Example |
|-----|-------------|---------|
| `testenv-kind.clusterName` | Name of the Kind cluster | `forge-test-integration-20250106-abc123` |
| `testenv-kind.kubeconfigPath` | Absolute path to kubeconfig | `/abs/path/to/tmpDir/kubeconfig` |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `KUBECONFIG` | Path to the kubeconfig file |

## Cluster Configuration

### Default Kind Config

testenv-kind creates clusters with the following default configuration:

- Single control-plane node
- Default Kind networking
- Ephemeral storage (destroyed on delete)

### Cluster Naming Convention

Clusters are named: `{projectName}-{testID}`

Where:
- `projectName` is from forge.yaml `name` field
- `testID` is the unique test environment ID

Example: `my-project-test-integration-20250106-abc123`

## Integration with Other Subengines

testenv-kind provides metadata used by other subengines:

```
testenv-kind --> KUBECONFIG --> testenv-lcr
            --> KUBECONFIG --> testenv-helm-install
```

### Example Flow

1. testenv-kind creates cluster and exports KUBECONFIG
2. testenv-lcr uses KUBECONFIG to deploy registry
3. testenv-helm-install uses KUBECONFIG to install charts

## Notes

- Clusters are ephemeral and deleted with the test environment
- Each test environment has an isolated cluster
- Kind CLI must be available in PATH
- Docker must be running

## See Also

- [Testenv-Kind Usage Guide](usage.md)
- [testenv Configuration](../../testenv/docs/schema.md)
- [testenv-lcr Configuration](../../testenv-lcr/docs/schema.md)
