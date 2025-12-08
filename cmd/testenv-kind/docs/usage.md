# Testenv-Kind Usage Guide

## Purpose

`testenv-kind` is a forge engine for creating Kind (Kubernetes in Docker) clusters as part of test environments. It generates unique clusters per test environment with isolated kubeconfig files.

## Invocation

### MCP Mode

Run as an MCP server:

```bash
testenv-kind --mcp
```

This is typically called automatically by the testenv orchestrator.

Forge invokes this automatically when configured in testenv:

```yaml
engines:
  - alias: my-testenv
    type: testenv
    testenv:
      - engine: go://testenv-kind
```

## Available MCP Tools

### `create`

Create a Kind cluster for a test environment.

**Input Schema:**
```json
{
  "testID": "string (required)",
  "stage": "string (required)",
  "tmpDir": "string (required)",
  "rootDir": "string (optional)"
}
```

**Output:**
```json
{
  "testID": "string",
  "files": {
    "testenv-kind.kubeconfig": "kubeconfig"
  },
  "metadata": {
    "testenv-kind.clusterName": "forge-test-integration-20250106-abc123",
    "testenv-kind.kubeconfigPath": "/abs/path/to/tmpDir/kubeconfig"
  },
  "env": {
    "KUBECONFIG": "/abs/path/to/tmpDir/kubeconfig"
  },
  "managedResources": [
    "/abs/path/to/tmpDir/kubeconfig"
  ]
}
```

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "create",
    "arguments": {
      "testID": "test-integration-20250106-abc123",
      "stage": "integration",
      "tmpDir": ".forge/tmp/test-integration-20250106-abc123"
    }
  }
}
```

### `delete`

Delete a Kind cluster.

**Input Schema:**
```json
{
  "testID": "string (required)",
  "metadata": {
    "testenv-kind.clusterName": "string (optional)"
  }
}
```

**Output:**
```json
{
  "success": true,
  "message": "Deleted kind cluster: forge-test-integration-20250106-abc123"
}
```

### `docs-list`

List all available documentation for testenv-kind.

### `docs-get`

Get a specific documentation by name.

**Input Schema:**
```json
{
  "name": "string (required)"
}
```

### `docs-validate`

Validate documentation completeness.

## Common Use Cases

### Basic Kind Cluster

Create a test environment with a Kind cluster:

```yaml
engines:
  - alias: k8s-testenv
    type: testenv
    testenv:
      - engine: go://testenv-kind
```

### Kind with Local Container Registry

Combine with testenv-lcr for image pushing:

```yaml
engines:
  - alias: integration-env
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-lcr
        spec:
          enabled: true
```

### Kind with Helm Charts

Add pre-installed charts:

```yaml
engines:
  - alias: full-testenv
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-lcr
        spec:
          enabled: true
      - engine: go://testenv-helm-install
        spec:
          charts:
            - name: nginx-ingress
              sourceType: helm-repo
              url: https://kubernetes.github.io/ingress-nginx
              chartName: ingress-nginx
              namespace: ingress-nginx
              createNamespace: true
```

## Implementation Details

- Uses `kind create cluster` command
- Cluster name format: `{projectName}-{testID}`
- Kubeconfig written to tmpDir for isolation
- Each test environment gets its own cluster
- Kubeconfig path stored in metadata for other subengines

## Cluster Naming

Cluster names are generated as: `{projectName}-{testID}`

Example: `forge-test-integration-20250106-abc123`

This ensures:
- Unique clusters per test environment
- Easy identification of test clusters
- Automatic cleanup by testID

## Environment Variables

The following environment variables are exported for use by subsequent subengines and test runners:

| Variable | Description | Example |
|----------|-------------|---------|
| `KUBECONFIG` | Path to cluster kubeconfig | `.forge/tmp/.../kubeconfig` |

## Requirements

- Kind CLI must be installed and available in PATH
- Docker must be running

## See Also

- [Testenv-Kind Configuration Schema](schema.md)
- [testenv MCP Server](../../testenv/docs/usage.md)
- [testenv-lcr MCP Server](../../testenv-lcr/docs/usage.md)
