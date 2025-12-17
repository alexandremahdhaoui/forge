# testenv-kind

**Create isolated Kubernetes clusters for each test environment.**

> "Running integration tests against a shared cluster was a nightmare - tests would interfere with each other. testenv-kind gives me a fresh Kind cluster per test environment, with its own kubeconfig, that gets cleaned up automatically."

## What problem does testenv-kind solve?

Integration tests need Kubernetes clusters, but shared clusters cause test interference and leave behind resources. testenv-kind creates isolated Kind clusters with unique names and kubeconfig files, then cleans them up completely on delete.

## How do I use testenv-kind?

Add it to a testenv configuration in forge.yaml:

```yaml
engines:
  - alias: k8s-testenv
    type: testenv
    testenv:
      - engine: go://testenv-kind

test:
  - name: integration
    testenv: alias://k8s-testenv
    runner: go://go-test
```

The cluster kubeconfig is automatically passed to subsequent subengines and test runners via the `KUBECONFIG` environment variable.

## What does testenv-kind provide to other subengines?

| Output | Description |
|--------|-------------|
| `KUBECONFIG` env var | Path to cluster kubeconfig |
| `testenv-kind.clusterName` metadata | Cluster name for identification |
| `testenv-kind.kubeconfigPath` metadata | Absolute path to kubeconfig file |

## How are clusters named?

Clusters follow the pattern: `{projectName}-{testID}`

Example: `forge-test-integration-20250106-abc123`

This ensures unique clusters per test environment and easy identification of test clusters.

## What are the requirements?

- Kind CLI installed and in PATH
- Docker running

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../MCP.md) - MCP tool documentation
- [testenv-lcr](../../testenv-lcr/docs/usage.md) - Add a container registry
- [testenv-helm-install](../../testenv-helm-install/docs/usage.md) - Pre-install Helm charts
