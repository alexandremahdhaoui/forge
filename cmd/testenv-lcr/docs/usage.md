# Testenv-LCR Usage Guide

## Purpose

`testenv-lcr` (Local Container Registry) is a forge engine for deploying TLS-enabled container registries inside Kind clusters. It provides a secure, authenticated registry with cert-manager integration for test environments.

## Invocation

### MCP Mode

Run as an MCP server:

```bash
testenv-lcr --mcp
```

This is typically called automatically by the testenv orchestrator.

Forge invokes this automatically when configured in testenv:

```yaml
engines:
  - alias: my-testenv
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-lcr
        spec:
          enabled: true
```

## Available MCP Tools

### `create`

Create a local container registry in a Kind cluster.

**Input Schema:**
```json
{
  "testID": "string (required)",
  "stage": "string (required)",
  "tmpDir": "string (required)",
  "metadata": {
    "testenv-kind.kubeconfigPath": "string"
  },
  "env": {
    "KUBECONFIG": "string"
  },
  "spec": {
    "enabled": true,
    "namespace": "string",
    "imagePullSecretNamespaces": ["string"],
    "images": [...]
  }
}
```

**Output:**
```json
{
  "testID": "string",
  "files": {
    "testenv-lcr.ca.crt": "ca.crt",
    "testenv-lcr.credentials.yaml": "registry-credentials.yaml"
  },
  "metadata": {
    "testenv-lcr.registryFQDN": "testenv-lcr.testenv-lcr.svc.cluster.local:31906",
    "testenv-lcr.namespace": "testenv-lcr",
    "testenv-lcr.port": "31906",
    "testenv-lcr.caCrtPath": "/abs/path/to/ca.crt",
    "testenv-lcr.credentialPath": "/abs/path/to/credentials.yaml"
  },
  "env": {
    "TESTENV_LCR_FQDN": "testenv-lcr.testenv-lcr.svc.cluster.local:31906",
    "TESTENV_LCR_HOST": "testenv-lcr.testenv-lcr.svc.cluster.local",
    "TESTENV_LCR_PORT": "31906",
    "TESTENV_LCR_NAMESPACE": "testenv-lcr",
    "TESTENV_LCR_CA_CERT": "/abs/path/to/ca.crt"
  },
  "managedResources": [...]
}
```

### `delete`

Delete a local container registry.

**Input Schema:**
```json
{
  "testID": "string (required)",
  "metadata": {
    "testenv-kind.kubeconfigPath": "string",
    "testenv-kind.clusterName": "string"
  }
}
```

### `create-image-pull-secret`

Create an image pull secret in a specific namespace.

**Input Schema:**
```json
{
  "testID": "string (required)",
  "namespace": "string (required)",
  "secretName": "string (optional)",
  "metadata": {
    "testenv-lcr.registryFQDN": "string",
    "testenv-lcr.caCrtPath": "string",
    "testenv-lcr.credentialPath": "string"
  }
}
```

### `list-image-pull-secrets`

List all image pull secrets created by testenv-lcr.

**Input Schema:**
```json
{
  "testID": "string (required)",
  "namespace": "string (optional)",
  "metadata": {
    "testenv-kind.kubeconfigPath": "string"
  }
}
```

### `docs-list`

List all available documentation for testenv-lcr.

### `docs-get`

Get a specific documentation by name.

### `docs-validate`

Validate documentation completeness.

## Common Use Cases

### Basic Registry

Enable a local container registry:

```yaml
engines:
  - alias: with-registry
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-lcr
        spec:
          enabled: true
```

### Registry with Image Pull Secrets

Create image pull secrets in multiple namespaces:

```yaml
engines:
  - alias: with-secrets
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-lcr
        spec:
          enabled: true
          imagePullSecretNamespaces:
            - default
            - my-app
            - system
```

### Registry with Pre-loaded Images

Push images to the registry during creation:

```yaml
engines:
  - alias: with-images
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-lcr
        spec:
          enabled: true
          images:
            - name: local://myapp:latest
            - name: quay.io/example/img:v1.2.3
              basicAuth:
                username:
                  envName: QUAY_USER
                password:
                  envName: QUAY_PASS
```

### Using Registry FQDN in Helm Values

Reference the registry in subsequent subengines:

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
            - name: my-app
              sourceType: local
              path: ./charts/my-app
              namespace: default
              values:
                image:
                  repository: "{{.Env.TESTENV_LCR_FQDN}}/my-app"
                  tag: latest
```

## Registry Details

| Property | Value |
|----------|-------|
| Image | `registry:2` |
| Port | Dynamic (30000-32767 NodePort range) |
| FQDN | `testenv-lcr.testenv-lcr.svc.cluster.local:<port>` |
| Auth | htpasswd (random 32-char credentials) |
| TLS | Self-signed via cert-manager |
| Storage | emptyDir (ephemeral) |

## Environment Variables

testenv-lcr exports environment variables for template expansion:

| Variable | Description | Example |
|----------|-------------|---------|
| `TESTENV_LCR_FQDN` | Full registry address with port | `testenv-lcr.testenv-lcr.svc.cluster.local:31906` |
| `TESTENV_LCR_HOST` | Registry hostname (without port) | `testenv-lcr.testenv-lcr.svc.cluster.local` |
| `TESTENV_LCR_PORT` | Dynamic port number | `31906` |
| `TESTENV_LCR_NAMESPACE` | Kubernetes namespace | `testenv-lcr` |
| `TESTENV_LCR_CA_CERT` | Path to CA certificate | `/abs/path/to/ca.crt` |

## Implementation Details

- Deploys cert-manager for TLS certificate management
- Creates self-signed CA and certificates
- Configures containerd trust on Kind nodes
- Sets up htpasswd authentication
- Manages /etc/hosts entry for FQDN resolution
- Runs port-forward for host accessibility

## See Also

- [Testenv-LCR Configuration Schema](schema.md)
- [testenv MCP Server](../../testenv/docs/usage.md)
- [testenv-kind MCP Server](../../testenv-kind/docs/usage.md)
