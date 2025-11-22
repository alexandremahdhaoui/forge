# testenv-helm-install ChartSpec Reference

This document provides a comprehensive reference for the `ChartSpec` structure used by the `testenv-helm-install` engine.

## Overview

The `ChartSpec` is inspired by FluxCD's GitOps Toolkit, synthesizing capabilities from multiple FluxCD CRDs (HelmRelease, HelmRepository, GitRepository, OCIRepository, Bucket) into a single, unified configuration structure.

## ChartSpec Structure

### Core Identity & Location

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Internal identifier for this chart configuration |
| `releaseName` | string | No | Helm release name in the cluster. Defaults to `name` if not specified |
| `namespace` | string | No | Kubernetes namespace where the release will be installed. Defaults to "default" |

### Source Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `sourceType` | string | Yes | Artifact acquisition strategy. Valid values: `"helm-repo"`, `"git"`, `"oci"`, `"s3"` |
| `url` | string | Yes | Primary locator for the source (HTTP/S URL for helm-repo, Git URL, OCI registry, or S3 endpoint) |
| `interval` | string | No | Reconciliation frequency (e.g., "10m", "1h"). Currently not used |

### Helm Repository Specifics (sourceType="helm-repo")

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `chartName` | string | Yes* | Name of the chart to fetch from the Helm repository index |
| `version` | string | No | Chart version constraint (e.g., "6.0.0", "^1.0.0", "*"). Defaults to "*" (latest) |

*Required when `sourceType` is `"helm-repo"`

### Git Repository Specifics (not yet implemented)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `chartPath` | string | Conditional | Relative path to chart directory within the Git repository |
| `gitBranch` | string | No | Git branch to checkout |
| `gitTag` | string | No | Git tag to checkout (takes precedence over branch) |
| `gitCommit` | string | No | Exact Git commit SHA (takes precedence over tag and branch) |
| `gitSemVer` | string | No | Semantic version range to match against Git tags |
| `ignorePaths` | []string | No | .gitignore-style patterns to exclude from artifact |

### OCI Repository Specifics (not yet implemented)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `ociProvider` | string | No | Verification provider: `"cosign"`, `"notation"` |
| `ociLayerMediaType` | string | No | Media type of the layer to extract |

### S3 Bucket Specifics (not yet implemented)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `s3BucketName` | string | Conditional | Name of the S3/Minio bucket |
| `s3BucketRegion` | string | No | AWS region (defaults to "us-east-1") |

### Authentication & Security

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `authSecretName` | string | No | Name of a Kubernetes Secret containing credentials |
| `passCredentials` | bool | No | Pass credentials to chart tarball download (critical for private repos). Defaults to false |
| `insecureSkipVerify` | bool | No | Disable TLS verification (development only). Defaults to false |

### Values Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `values` | map[string]interface{} | No | Inline Helm values (currently supports flat key-value pairs only) |
| `valuesFiles` | []string | No | Paths to values files within the source artifact |
| `valueReferences` | []ValueReference | No | References to ConfigMaps/Secrets (not yet implemented) |

### Lifecycle & Remediation

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `timeout` | string | No | Helm operation timeout (e.g., "5m", "10m"). Defaults to "5m" |
| `createNamespace` | bool | No | Create target namespace if it doesn't exist. Defaults to false |
| `forceUpgrade` | bool | No | Use `helm upgrade --force` (recreates immutable resources). Defaults to false |
| `disableHooks` | bool | No | Prevent Helm hooks from running. Defaults to false |
| `disableWait` | bool | No | Skip waiting for resources to be ready. Defaults to false |
| `testEnable` | bool | No | Run helm tests after installation. Defaults to false |

## Examples

### Basic Helm Repository Chart

```yaml
charts:
  - name: podinfo-release
    sourceType: helm-repo
    url: https://stefanprodan.github.io/podinfo
    chartName: podinfo
    version: "6.0.0"
    namespace: test-podinfo
    releaseName: test-podinfo
    createNamespace: true
```

### Advanced with Lifecycle Options

```yaml
charts:
  - name: nginx-release
    sourceType: helm-repo
    url: https://charts.bitnami.com/bitnami
    chartName: nginx
    version: "^15.0.0"
    namespace: web
    releaseName: my-nginx
    createNamespace: true
    timeout: "10m"
    forceUpgrade: false
    disableWait: false
    disableHooks: false
    testEnable: true
    values:
      replicaCount: 3
      service:
        type: LoadBalancer
```

### With Values Files

```yaml
charts:
  - name: app-release
    sourceType: helm-repo
    url: https://charts.example.com
    chartName: myapp
    namespace: production
    releaseName: myapp-prod
    valuesFiles:
      - values-prod.yaml
      - secrets-prod.yaml
    values:
      environment: production
```

## Current Limitations

### Values Composition
- The `values` field currently supports **flat key-value pairs only**
- Nested structures in `values` are not yet fully supported
- Workaround: Use `valuesFiles` for complex value structures

### Source Types
Only `sourceType: helm-repo` is currently implemented:
- ✅ Helm repositories (HTTP/S)
- ❌ Git repositories (planned)
- ❌ OCI registries (planned)
- ❌ S3 buckets (planned)

### ValueReferences
- The `valueReferences` field is defined but not yet implemented
- A warning is logged if this field is specified
- Planned for future release

## Migration from Old Format

**Breaking Change:** The old format using `repo` and chart names like `"podinfo/podinfo"` is no longer supported.

**Old format (no longer works):**
```yaml
charts:
  - name: podinfo/podinfo
    repo: https://stefanprodan.github.io/podinfo
    namespace: test-podinfo
```

**New format (required):**
```yaml
charts:
  - name: podinfo-release
    sourceType: helm-repo
    url: https://stefanprodan.github.io/podinfo
    chartName: podinfo
    namespace: test-podinfo
```

## See Also

- [testenv-helm-install MCP Documentation](../cmd/testenv-helm-install/MCP.md)
- [Built-in Tools Reference](./built-in-tools.md)
- [FluxCD HelmRelease API](https://fluxcd.io/flux/components/helm/helmreleases/)
