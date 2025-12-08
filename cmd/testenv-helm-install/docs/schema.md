# Testenv-Helm-Install Configuration Schema

## Overview

This document describes the configuration options for `testenv-helm-install` in `forge.yaml`. The testenv-helm-install engine installs Helm charts into Kubernetes clusters as part of test environments.

## Basic Configuration

```yaml
engines:
  - alias: my-testenv
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-helm-install
        spec:
          charts:
            - name: nginx
              sourceType: helm-repo
              url: https://kubernetes.github.io/ingress-nginx
              chartName: ingress-nginx
```

## ChartSpec Fields

### Core Fields (Required)

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Internal identifier for this chart configuration. |
| `sourceType` | string | Artifact acquisition strategy: `helm-repo`, `git`, `oci`, `s3`, `local`. |

### Source Configuration

| Field | Type | Required For | Description |
|-------|------|--------------|-------------|
| `url` | string | helm-repo, git, oci, s3 | Primary source locator. |
| `path` | string | local | Path to local chart directory. |
| `chartName` | string | helm-repo | Name of chart in repository. |
| `chartPath` | string | git, s3 | Path to chart within source. |

### Version Control

| Field | Type | Description |
|-------|------|-------------|
| `version` | string | Chart version constraint (e.g., "6.0.0", "^1.0.0"). |
| `gitBranch` | string | Git branch to checkout. |
| `gitTag` | string | Git tag to checkout. |
| `gitCommit` | string | Git commit SHA to checkout. |
| `gitSemVer` | string | SemVer constraint for Git tags. |

### Release Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `releaseName` | string | `name` | Helm release name in cluster. |
| `namespace` | string | `default` | Kubernetes namespace for release. |
| `createNamespace` | bool | `false` | Create namespace if missing. |

### Lifecycle Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `timeout` | string | `5m` | Time to wait for Helm operations. |
| `disableWait` | bool | `false` | Skip waiting for resources. |
| `forceUpgrade` | bool | `false` | Use `helm upgrade --force`. |
| `disableHooks` | bool | `false` | Disable Helm hooks. |
| `testEnable` | bool | `false` | Run Helm tests after install. |

### Values Configuration

| Field | Type | Description |
|-------|------|-------------|
| `values` | map | Inline Helm values (highest precedence). |
| `valuesFiles` | []string | Values files within source (lowest precedence). |
| `valueReferences` | []ValueReference | References to ConfigMaps/Secrets. |

### Authentication

| Field | Type | Description |
|-------|------|-------------|
| `authSecretName` | string | Kubernetes Secret with credentials. |
| `passCredentials` | bool | Pass credentials to chart download. |
| `insecureSkipVerify` | bool | Skip TLS verification. |

### S3-Specific Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `s3BucketName` | string | - | S3 bucket name. |
| `s3BucketRegion` | string | `us-east-1` | AWS region. |

## ValueReference Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `kind` | string | Yes | Resource type: `ConfigMap` or `Secret`. |
| `name` | string | Yes | Name of the resource. |
| `valuesKey` | string | No | Specific key to extract. |
| `targetPath` | string | No | Dot-notation path to merge values. |
| `optional` | bool | No | Whether reference is optional. |

## Examples

### Helm Repository Chart

```yaml
engines:
  - alias: with-helm
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-helm-install
        spec:
          charts:
            - name: nginx-release
              sourceType: helm-repo
              url: https://kubernetes.github.io/ingress-nginx
              chartName: ingress-nginx
              version: "4.0.0"
              namespace: ingress-nginx
              releaseName: my-nginx
              createNamespace: true
              timeout: 10m
              values:
                controller:
                  replicaCount: 2
```

### Local Chart

```yaml
engines:
  - alias: local-chart
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-helm-install
        spec:
          charts:
            - name: my-app
              sourceType: local
              path: ./charts/my-app
              namespace: default
              values:
                image:
                  repository: my-app
                  tag: latest
```

### Git Source with Branch

```yaml
engines:
  - alias: git-chart
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-helm-install
        spec:
          charts:
            - name: podinfo
              sourceType: git
              url: https://github.com/stefanprodan/podinfo
              gitBranch: master
              chartPath: charts/podinfo
              namespace: default
              createNamespace: true
```

### Git Source with SemVer

```yaml
engines:
  - alias: git-semver
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-helm-install
        spec:
          charts:
            - name: podinfo
              sourceType: git
              url: https://github.com/stefanprodan/podinfo
              gitSemVer: "^6.0.0"
              chartPath: charts/podinfo
              namespace: default
```

### OCI Registry

```yaml
engines:
  - alias: oci-chart
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-helm-install
        spec:
          charts:
            - name: podinfo-oci
              sourceType: oci
              url: oci://ghcr.io/stefanprodan/charts/podinfo
              version: "6.0.0"
              namespace: default
              createNamespace: true
```

### OCI with Authentication

```yaml
engines:
  - alias: private-oci
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-helm-install
        spec:
          charts:
            - name: private-chart
              sourceType: oci
              url: oci://ghcr.io/myorg/charts/myapp:1.0.0
              authSecretName: oci-registry-creds
              namespace: production
              createNamespace: true
```

### S3 Source

```yaml
engines:
  - alias: s3-chart
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-helm-install
        spec:
          charts:
            - name: my-chart
              sourceType: s3
              url: http://localhost:9000
              s3BucketName: helm-charts
              chartPath: charts/my-chart-1.0.0.tgz
              s3BucketRegion: us-east-1
              authSecretName: s3-creds
              namespace: default
```

### ValueReferences Example

```yaml
engines:
  - alias: with-value-refs
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-helm-install
        spec:
          charts:
            - name: my-app
              sourceType: helm-repo
              url: https://charts.example.com
              chartName: my-app
              namespace: production
              valueReferences:
                - kind: ConfigMap
                  name: base-config
                - kind: Secret
                  name: database-creds
                  targetPath: database
                - kind: ConfigMap
                  name: feature-flags
                  optional: true
              values:
                replicaCount: 3  # Overrides everything above
```

### Template Expansion with testenv-lcr

```yaml
engines:
  - alias: integration
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

## Output Artifacts

### Metadata

| Key | Description |
|-----|-------------|
| `testenv-helm-install.chartCount` | Number of installed charts. |
| `testenv-helm-install.chart.N.name` | Chart name at index N. |
| `testenv-helm-install.chart.N.releaseName` | Release name at index N. |
| `testenv-helm-install.chart.N.namespace` | Namespace at index N. |

## Values Precedence

From lowest to highest priority:

1. `valuesFiles` - Files within source artifact
2. `valueReferences` - ConfigMaps/Secrets
3. `values` - Inline values (highest)

## Notes

- Charts install sequentially in order
- Charts uninstall in reverse order
- Requires Helm CLI in PATH
- Requires kubeconfig from testenv-kind
- Git reference precedence: Commit > Tag > SemVer > Branch

## See Also

- [Testenv-Helm-Install Usage Guide](usage.md)
- [testenv Configuration](../../testenv/docs/schema.md)
- [testenv-kind Configuration](../../testenv-kind/docs/schema.md)
