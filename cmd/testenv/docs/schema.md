# Testenv Configuration Schema

## Overview

This document describes the configuration options for `testenv` in `forge.yaml`. The testenv engine orchestrates test environment creation by coordinating testenv subengines.

## Basic Configuration

```yaml
test:
  - name: integration
    stage: integration
    testenv: forge://testenv
    runner: forge://go-test
```

## Configuration Options

### Test Stage Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | The name of the test stage. |
| `stage` | string | Stage identifier (e.g., "unit", "integration", "e2e"). |
| `testenv` | string | Engine URL for test environment orchestration. Must be `forge://testenv`. |
| `runner` | string | Engine URL for running tests (e.g., `forge://go-test`). |

## Engines Configuration

For advanced control, use the `engines` section to configure testenv subengines:

```yaml
engines:
  - alias: my-testenv
    type: testenv
    testenv:
      - engine: forge://testenv-kind
      - engine: forge://testenv-lcr
        spec:
          enabled: true
      - engine: forge://testenv-helm-install
        spec:
          charts:
            - name: cert-manager
              sourceType: helm-repo
              url: https://charts.jetstack.io
              chartName: cert-manager
```

### Engine Entry Fields

| Field | Type | Description |
|-------|------|-------------|
| `engine` | string | Engine URL (e.g., `forge://testenv-kind`). |
| `spec` | object | Engine-specific configuration. |

### Available Subengines

| Engine | Purpose |
|--------|---------|
| `forge://testenv-kind` | Creates Kind (Kubernetes in Docker) clusters |
| `forge://testenv-lcr` | Deploys local container registry with TLS |
| `forge://testenv-helm-install` | Installs Helm charts into the cluster |
| `forge://testenv-stub` | Lightweight no-op subengine for testing |

## Examples

### Minimal Configuration

```yaml
test:
  - name: integration
    stage: integration
    testenv: forge://testenv
    runner: forge://go-test
```

### Full Configuration with Custom Subengines

```yaml
engines:
  - alias: integration-env
    type: testenv
    testenv:
      - engine: forge://testenv-kind
      - engine: forge://testenv-lcr
        spec:
          enabled: true
          namespace: testenv-lcr
          imagePullSecretNamespaces:
            - default
            - my-app
          images:
            - name: local://myapp:latest
      - engine: forge://testenv-helm-install
        spec:
          charts:
            - name: nginx
              sourceType: helm-repo
              url: https://kubernetes.github.io/ingress-nginx
              chartName: ingress-nginx
              namespace: ingress-nginx
              createNamespace: true

test:
  - name: integration
    stage: integration
    testenv: integration-env
    runner: forge://go-test
```

### Multiple Test Stages

```yaml
test:
  - name: unit
    stage: unit
    runner: forge://go-test
    # No testenv needed for unit tests

  - name: integration
    stage: integration
    testenv: forge://testenv
    runner: forge://go-test

  - name: e2e
    stage: e2e
    testenv: forge://testenv
    runner: forge://go-test
```

## TestEnvironment Structure

When a test environment is created, the following structure is stored in the artifact store:

```yaml
testEnvironments:
  - id: "test-integration-20250106-abc123"
    name: "integration"
    stage: "integration"
    status: "created"
    createdAt: "2025-01-06T10:00:00Z"
    updatedAt: "2025-01-06T10:00:00Z"
    tmpDir: ".forge/tmp/test-integration-20250106-abc123"
    files:
      testenv-kind.kubeconfig: "kubeconfig"
      testenv-lcr.ca.crt: "ca.crt"
      testenv-lcr.credentials.yaml: "registry-credentials.yaml"
    metadata:
      testenv-kind.clusterName: "forge-test-integration-20250106-abc123"
      testenv-kind.kubeconfigPath: ".forge/tmp/.../kubeconfig"
      testenv-lcr.registryFQDN: "testenv-lcr.testenv-lcr.svc.cluster.local:31906"
    env:
      KUBECONFIG: ".forge/tmp/.../kubeconfig"
      TESTENV_LCR_FQDN: "testenv-lcr.testenv-lcr.svc.cluster.local:31906"
    managedResources:
      - ".forge/tmp/.../kubeconfig"
      - ".forge/tmp/.../ca.crt"
```

## Environment Variables

Test environments export environment variables for use by test runners:

| Variable | Description | Source |
|----------|-------------|--------|
| `KUBECONFIG` | Path to kubeconfig file | testenv-kind |
| `TESTENV_LCR_FQDN` | Local container registry FQDN | testenv-lcr |
| `TESTENV_LCR_HOST` | Registry hostname (without port) | testenv-lcr |
| `TESTENV_LCR_PORT` | Registry port number | testenv-lcr |
| `TESTENV_LCR_NAMESPACE` | Registry Kubernetes namespace | testenv-lcr |
| `TESTENV_LCR_CA_CERT` | Path to CA certificate | testenv-lcr |

## Notes

- Subengines execute in order during create
- Subengines execute in reverse order during delete
- Each test environment has a unique tmpDir
- Test environments are isolated from each other

## See Also

- [Testenv Usage Guide](usage.md)
- [testenv-kind Configuration](../../testenv-kind/docs/schema.md)
- [testenv-lcr Configuration](../../testenv-lcr/docs/schema.md)
- [testenv-helm-install Configuration](../../testenv-helm-install/docs/schema.md)
