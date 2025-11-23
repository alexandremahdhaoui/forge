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

### Git Repository Specifics (sourceType="git")

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `chartPath` | string | Yes* | Relative path to chart directory within the Git repository (e.g., "charts/app") |
| `gitBranch` | string | Conditional** | Git branch to checkout (e.g., "main", "develop") |
| `gitTag` | string | Conditional** | Git tag to checkout (takes precedence over branch, e.g., "v1.0.0") |
| `gitCommit` | string | Conditional** | Exact Git commit SHA - minimum 7 characters (takes precedence over tag and branch) |
| `gitSemVer` | string | Conditional** | Semantic version range to match against Git tags (e.g., "^1.0.0", ">=1.0.0 <2.0.0") |
| `ignorePaths` | []string | No | .gitignore-style patterns to exclude from artifact (optimization placeholder - logs warning if used) |

*Required when `sourceType` is `"git"`

**At least one Git reference field is required. Precedence order: `gitCommit` > `gitTag` > `gitSemVer` > `gitBranch`

### OCI Repository Specifics (sourceType="oci")

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | string | No | Chart version (can also be specified in URL with `:tag` or `@sha256:digest`) |
| `authSecretName` | string | No | Name of Kubernetes Secret (type: `kubernetes.io/dockerconfigjson`) for private registries |
| `ociProvider` | string | No | Signature verification provider: `"cosign"`, `"notation"` (logs warning, cryptographic verification reserved for future) |

**Note**: Requires Helm 3.8+ for OCI support. The chart name is embedded in the OCI URL (e.g., `oci://ghcr.io/org/charts/mychart`), so `chartName` field should not be set.

### S3 Bucket Specifics (sourceType="s3")

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `s3BucketName` | string | Yes* | Name of the S3-compatible bucket containing the chart |
| `chartPath` | string | Yes* | Path to chart tarball within bucket (must end with `.tgz` or `.tar.gz`) |
| `s3BucketRegion` | string | No | AWS region for the bucket. Defaults to "us-east-1" |
| `authSecretName` | string | No | Name of Kubernetes Secret containing S3 credentials (accessKeyID, secretAccessKey, sessionToken) |

*Required when `sourceType` is `"s3"`

**Note**: The chart tarball is downloaded from S3 before installation. Git, OCI, and `chartName` fields should not be set for S3 sources.

### Authentication & Security

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `authSecretName` | string | No | Name of a Kubernetes Secret containing credentials |
| `passCredentials` | bool | No | Pass credentials to chart tarball download (critical for private repos). Defaults to false |
| `insecureSkipVerify` | bool | No | Disable TLS verification (development only). Defaults to false |

### Values Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `values` | map[string]interface{} | No | Inline Helm values - supports nested maps, arrays, and all YAML types |
| `valuesFiles` | []string | No | Paths to values files within the source artifact |
| `valueReferences` | []ValueReference | No | References to Kubernetes ConfigMaps/Secrets containing values |

#### ValueReference Structure

Each `ValueReference` has the following fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `kind` | string | Yes | Resource type: `"ConfigMap"` or `"Secret"` |
| `name` | string | Yes | Name of the ConfigMap or Secret |
| `valuesKey` | string | No | Specific key to extract from the resource's data. If empty, all keys are merged |
| `targetPath` | string | No | Dot-notation path where values should be merged (e.g., `"server.config"`). If empty, merges at root level |
| `optional` | bool | No | Whether the reference is optional. If `true` and resource not found, continues without error. Defaults to `false` |

**Value Composition Order** (lowest to highest precedence):
1. `valuesFiles` - Values files from the source artifact
2. `valueReferences` - Values from ConfigMaps/Secrets (in order)
3. `values` - Inline values (highest precedence)

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

### Git Source with Branch

```yaml
charts:
  - name: podinfo-from-git
    sourceType: git
    url: https://github.com/stefanprodan/podinfo
    gitBranch: master
    chartPath: charts/podinfo
    namespace: test-git
    releaseName: podinfo-git
    createNamespace: true
    timeout: 5m
```

### Git Source with Tag

```yaml
charts:
  - name: podinfo-v6
    sourceType: git
    url: https://github.com/stefanprodan/podinfo
    gitTag: "6.0.0"
    chartPath: charts/podinfo
    namespace: test-git-tag
    createNamespace: true
```

### Git Source with Specific Commit

```yaml
charts:
  - name: podinfo-commit
    sourceType: git
    url: https://github.com/stefanprodan/podinfo
    gitCommit: abc1234def5678
    chartPath: charts/podinfo
    namespace: test-git-commit
    createNamespace: true
```

### Git Source with SemVer Constraint

```yaml
charts:
  - name: podinfo-semver
    sourceType: git
    url: https://github.com/stefanprodan/podinfo
    gitSemVer: "^6.0.0"
    chartPath: charts/podinfo
    namespace: test-git-semver
    createNamespace: true
    values:
      replicaCount: 2
```

### Git Source with SSH (Private Repository)

```yaml
charts:
  - name: private-chart
    sourceType: git
    url: git@github.com:organization/private-repo.git
    gitBranch: main
    chartPath: deploy/charts/myapp
    namespace: production
    createNamespace: true
```

### OCI Registry (Public)

```yaml
charts:
  - name: podinfo-oci
    sourceType: oci
    url: oci://ghcr.io/stefanprodan/charts/podinfo
    version: "6.0.0"
    namespace: test-oci
    releaseName: podinfo-oci
    createNamespace: true
```

### OCI Registry (Private with Authentication)

```yaml
charts:
  - name: private-chart
    sourceType: oci
    url: oci://ghcr.io/myorg/charts/myapp:1.2.3
    authSecretName: oci-registry-creds
    namespace: production
    createNamespace: true
    values:
      replicas: 3
      image:
        tag: v1.2.3
```

**Note**: Create the auth secret beforehand:
```bash
kubectl create secret docker-registry oci-registry-creds \
  --docker-server=ghcr.io \
  --docker-username=myusername \
  --docker-password=mytoken \
  --namespace=production
```

### OCI Registry with Digest

```yaml
charts:
  - name: pinned-chart
    sourceType: oci
    url: oci://ghcr.io/stefanprodan/charts/podinfo@sha256:abc123def456789...
    namespace: test-oci-digest
    createNamespace: true
```

### S3 Bucket (MinIO)

```yaml
charts:
  - name: myapp-from-s3
    sourceType: s3
    url: http://localhost:9000
    s3BucketName: helm-charts
    chartPath: production/myapp-1.2.3.tgz
    s3BucketRegion: us-east-1
    authSecretName: s3-creds
    namespace: myapp
    createNamespace: true
```

**Note**: Create the S3 credentials secret:
```bash
kubectl create secret generic s3-creds \
  --from-literal=accessKeyID=AKIAIOSFODNN7EXAMPLE \
  --from-literal=secretAccessKey=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY \
  --namespace=myapp
```

### S3 Bucket (AWS S3)

```yaml
charts:
  - name: production-app
    sourceType: s3
    url: https://s3.amazonaws.com
    s3BucketName: my-helm-charts
    chartPath: apps/myapp-2.0.0.tgz
    s3BucketRegion: eu-west-1
    authSecretName: aws-s3-creds
    namespace: production
    createNamespace: true
    values:
      environment: production
```

### ValueReferences (ConfigMap)

```yaml
charts:
  - name: myapp
    sourceType: helm-repo
    url: https://charts.example.com
    chartName: myapp
    namespace: production
    createNamespace: true
    valueReferences:
      - kind: ConfigMap
        name: myapp-config
        # All keys from myapp-config merged at root level
```

### ValueReferences (Secret with Target Path)

```yaml
charts:
  - name: myapp
    sourceType: helm-repo
    url: https://charts.example.com
    chartName: myapp
    namespace: production
    createNamespace: true
    valueReferences:
      - kind: Secret
        name: database-credentials
        valuesKey: connection-string
        targetPath: database.connectionString
        # Merges the 'connection-string' value to database.connectionString
```

### ValueReferences (Multiple with Precedence)

```yaml
charts:
  - name: myapp
    sourceType: helm-repo
    url: https://charts.example.com
    chartName: myapp
    namespace: production
    createNamespace: true
    valueReferences:
      - kind: ConfigMap
        name: base-config
        # First: merge all keys from base-config
      - kind: ConfigMap
        name: environment-overrides
        # Second: merge all keys (overrides base-config)
      - kind: Secret
        name: secrets
        targetPath: secure
        # Third: merge secrets to secure.* path
    values:
      # Highest precedence - overrides everything above
      replicaCount: 3
      server:
        port: 8080
```

### Nested Values Structure

```yaml
charts:
  - name: complex-app
    sourceType: helm-repo
    url: https://charts.example.com
    chartName: myapp
    namespace: production
    createNamespace: true
    values:
      server:
        replicas: 3
        config:
          database:
            host: db.example.com
            port: 5432
          cache:
            enabled: true
            ttl: 300
        env:
          - name: LOG_LEVEL
            value: info
          - name: DEBUG
            value: false
      resources:
        limits:
          cpu: 200m
          memory: 256Mi
        requests:
          cpu: 100m
          memory: 128Mi
```

## Current Limitations

All core features are now implemented. The following advanced features are reserved for future enhancement:

### OCI Signature Verification
- The `ociProvider` field (`"cosign"` or `"notation"`) is defined but does not perform cryptographic verification
- Currently logs a warning when set
- Reserved for future implementation of signature validation

### Git Ignore Patterns
- The `ignorePaths` field for Git sources logs a warning when used
- Pattern filtering is not yet applied (optimization placeholder)
- All files from the Git repository are currently cloned

### Implemented Features
All source types and value composition features are fully implemented:
- ✅ Helm repositories (HTTP/S) - `helm-repo`
- ✅ Git repositories (HTTP/S and SSH) - `git` with branch/tag/commit/semver support
- ✅ OCI registries (Helm 3.8+) - `oci` with authentication
- ✅ S3 buckets (AWS S3 and compatible) - `s3` with authentication
- ✅ ValueReferences - ConfigMap/Secret value sources with optional and targetPath support
- ✅ Nested values - Full YAML structure support in `values` field

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

- [testenv-helm-install MCP Documentation](../cmd/testenv-helm-install/MCP.md) - MCP server details and examples
- [Migration Guide](./testenv-helm-install-migration.md) - Upgrade from old format
- [Testing Guide](./testenv-helm-install-testing.md) - Test infrastructure setup
- [Built-in Tools Reference](./built-in-tools.md)
- [FluxCD HelmRelease API](https://fluxcd.io/flux/components/helm/helmreleases/) - Design inspiration
