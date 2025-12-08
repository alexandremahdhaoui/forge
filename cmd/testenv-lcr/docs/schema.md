# Testenv-LCR Configuration Schema

## Overview

This document describes the configuration options for `testenv-lcr` in `forge.yaml`. The testenv-lcr engine deploys a TLS-enabled local container registry inside Kind clusters.

## Basic Configuration

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

## Configuration Options

### Spec Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable/disable the registry. |
| `namespace` | string | `testenv-lcr` | Kubernetes namespace for deployment. |
| `imagePullSecretNamespaces` | []string | `[]` | Namespaces where image pull secrets are created. |
| `imagePullSecretName` | string | `local-container-registry-credentials` | Name of the image pull secret. |
| `images` | []ImageSource | `[]` | Images to push to the registry. |

### ImageSource Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Image reference (see formats below). |
| `basicAuth` | BasicAuth | No | Credentials for private registries. |

### BasicAuth Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `username` | ValueFrom | Yes | Username credential. |
| `password` | ValueFrom | Yes | Password credential. |

### ValueFrom Fields

| Field | Type | Description |
|-------|------|-------------|
| `envName` | string | Environment variable name. |
| `literal` | string | Direct literal value. |

**Note:** `envName` and `literal` are mutually exclusive.

## Image Name Formats

| Format | Description |
|--------|-------------|
| `local://name:tag` | Image exists in local Docker daemon. |
| `registry/path:tag` | Image pulled from remote registry. |

**Note:** Tags are mandatory. No implicit `:latest`.

## Global Configuration

testenv-lcr also reads from the root-level `localContainerRegistry` section:

```yaml
localContainerRegistry:
  enabled: true
  namespace: testenv-lcr
  credentialPath: .forge/registry-credentials.yaml
  caCrtPath: .forge/ca.crt
  imagePullSecretNamespaces:
    - default
  imagePullSecretName: local-container-registry-credentials
```

**Note:** Spec values override global configuration.

## Examples

### Minimal Configuration

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

### Full Configuration

```yaml
engines:
  - alias: full-registry
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-lcr
        spec:
          enabled: true
          namespace: my-registry
          imagePullSecretNamespaces:
            - default
            - my-app
            - system
          imagePullSecretName: my-registry-credentials
          images:
            - name: local://myapp:v1.0.0
            - name: quay.io/external/image:v2.0.0
              basicAuth:
                username:
                  envName: QUAY_USER
                password:
                  envName: QUAY_PASS
```

### Using Environment Variables for Credentials

```yaml
engines:
  - alias: with-private-images
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-lcr
        spec:
          enabled: true
          images:
            - name: ghcr.io/myorg/myimage:v1.0.0
              basicAuth:
                username:
                  envName: GHCR_USER
                password:
                  envName: GHCR_TOKEN
```

### Using Literal Credentials

```yaml
engines:
  - alias: with-literal-creds
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-lcr
        spec:
          enabled: true
          images:
            - name: docker.io/myimage:v1.0.0
              basicAuth:
                username:
                  literal: myuser
                password:
                  literal: mypassword
```

## Output Artifacts

### Files

| Key | Relative Path | Description |
|-----|---------------|-------------|
| `testenv-lcr.ca.crt` | `ca.crt` | CA certificate for TLS verification. |
| `testenv-lcr.credentials.yaml` | `registry-credentials.yaml` | Registry credentials (username/password). |

### Metadata

| Key | Description | Example |
|-----|-------------|---------|
| `testenv-lcr.registryFQDN` | Registry FQDN with port | `testenv-lcr.testenv-lcr.svc.cluster.local:31906` |
| `testenv-lcr.namespace` | Kubernetes namespace | `testenv-lcr` |
| `testenv-lcr.port` | Dynamic port number | `31906` |
| `testenv-lcr.caCrtPath` | Absolute path to CA cert | `/abs/path/to/ca.crt` |
| `testenv-lcr.credentialPath` | Absolute path to credentials | `/abs/path/to/credentials.yaml` |
| `testenv-lcr.enabled` | Whether registry was enabled | `true` |
| `testenv-lcr.imagePullSecretCount` | Number of secrets created | `2` |
| `testenv-lcr.imagePullSecret.N.namespace` | Namespace of Nth secret | `default` |
| `testenv-lcr.imagePullSecret.N.secretName` | Name of Nth secret | `local-container-registry-credentials` |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `TESTENV_LCR_FQDN` | Full registry address with port. |
| `TESTENV_LCR_HOST` | Registry hostname (without port). |
| `TESTENV_LCR_PORT` | Dynamic port number. |
| `TESTENV_LCR_NAMESPACE` | Kubernetes namespace. |
| `TESTENV_LCR_CA_CERT` | Path to CA certificate. |

## Credential File Format

The credentials file is YAML:

```yaml
username: <random-32-chars>
password: <random-32-chars>
```

## Registry Architecture

```
+-------------------+     +-------------------+
| Host Machine      |     | Kind Cluster      |
|                   |     |                   |
| Port-Forward      |---->| NodePort Service  |
| localhost:<port>  |     | <port>            |
|                   |     |                   |
+-------------------+     +-------+-----------+
                                  |
                          +-------v-----------+
                          | Registry Pod      |
                          | registry:2        |
                          | TLS + htpasswd    |
                          +-------------------+
```

## Notes

- Requires testenv-kind to run first
- Uses KUBECONFIG from environment or metadata
- Dynamic port allocation prevents conflicts
- Containerd trust configured on Kind nodes
- Best-effort cleanup on delete

## See Also

- [Testenv-LCR Usage Guide](usage.md)
- [testenv Configuration](../../testenv/docs/schema.md)
- [testenv-kind Configuration](../../testenv-kind/docs/schema.md)
