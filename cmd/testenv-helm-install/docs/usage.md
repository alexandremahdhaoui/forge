# Testenv-Helm-Install Usage Guide

## Purpose

`testenv-helm-install` is a forge engine for installing Helm charts into Kubernetes clusters as part of test environments. It supports multiple source types including Helm repositories, Git, OCI registries, and S3 buckets.

## Invocation

### MCP Mode

Run as an MCP server:

```bash
testenv-helm-install --mcp
```

This is typically called automatically by the testenv orchestrator.

Forge invokes this automatically when configured in testenv:

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

## Available MCP Tools

### `create`

Install Helm charts into a Kubernetes cluster.

**Input Schema:**
```json
{
  "testID": "string (required)",
  "stage": "string (required)",
  "tmpDir": "string (required)",
  "rootDir": "string (optional)",
  "metadata": {
    "testenv-kind.kubeconfigPath": "string"
  },
  "env": {
    "KUBECONFIG": "string"
  },
  "spec": {
    "charts": [...]
  }
}
```

**Output:**
```json
{
  "testID": "string",
  "files": {},
  "metadata": {
    "testenv-helm-install.chartCount": "2",
    "testenv-helm-install.chart.0.name": "nginx",
    "testenv-helm-install.chart.0.releaseName": "nginx",
    "testenv-helm-install.chart.0.namespace": "ingress-nginx"
  },
  "managedResources": []
}
```

### `delete`

Uninstall Helm charts from a Kubernetes cluster.

**Input Schema:**
```json
{
  "testID": "string (required)",
  "metadata": {
    "testenv-helm-install.chartCount": "string",
    "testenv-helm-install.chart.N.releaseName": "string",
    "testenv-kind.kubeconfigPath": "string"
  }
}
```

### `docs-list`

List all available documentation for testenv-helm-install.

### `docs-get`

Get a specific documentation by name.

### `docs-validate`

Validate documentation completeness.

## Source Types

### Helm Repository

```yaml
charts:
  - name: nginx-release
    sourceType: helm-repo
    url: https://kubernetes.github.io/ingress-nginx
    chartName: ingress-nginx
    version: "4.0.0"
    namespace: ingress-nginx
    createNamespace: true
```

### Local Chart

```yaml
charts:
  - name: my-app
    sourceType: local
    path: ./charts/my-app
    namespace: default
```

### Git Repository

```yaml
charts:
  - name: podinfo-from-git
    sourceType: git
    url: https://github.com/stefanprodan/podinfo
    gitBranch: master
    chartPath: charts/podinfo
    namespace: default
    createNamespace: true
```

### OCI Registry

```yaml
charts:
  - name: podinfo-oci
    sourceType: oci
    url: oci://ghcr.io/stefanprodan/charts/podinfo
    version: "6.0.0"
    namespace: default
```

### S3 Bucket

```yaml
charts:
  - name: my-chart
    sourceType: s3
    url: http://localhost:9000
    s3BucketName: helm-charts
    chartPath: charts/my-chart-1.0.0.tgz
    namespace: default
```

## Common Use Cases

### Basic Helm Chart Installation

```yaml
engines:
  - alias: with-charts
    type: testenv
    testenv:
      - engine: go://testenv-kind
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
              values:
                installCRDs: true
```

### Using Registry FQDN from testenv-lcr

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

### Multiple Charts with Dependencies

```yaml
engines:
  - alias: full-stack
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-helm-install
        spec:
          charts:
            # Install CRDs first
            - name: cert-manager
              sourceType: helm-repo
              url: https://charts.jetstack.io
              chartName: cert-manager
              version: v1.13.0
              namespace: cert-manager
              createNamespace: true
              values:
                installCRDs: true

            # Then ingress controller
            - name: nginx-ingress
              sourceType: helm-repo
              url: https://kubernetes.github.io/ingress-nginx
              chartName: ingress-nginx
              namespace: ingress-nginx
              createNamespace: true

            # Finally the application
            - name: my-app
              sourceType: local
              path: ./charts/my-app
              namespace: default
```

## Values Configuration

### Inline Values

```yaml
charts:
  - name: my-release
    sourceType: helm-repo
    url: https://charts.example.com
    chartName: my-chart
    values:
      replicaCount: 3
      image:
        repository: nginx
        tag: latest
      service:
        type: LoadBalancer
```

### Value References (ConfigMap/Secret)

```yaml
charts:
  - name: my-release
    sourceType: helm-repo
    url: https://charts.example.com
    chartName: my-chart
    valueReferences:
      - kind: ConfigMap
        name: my-config
      - kind: Secret
        name: my-secrets
        targetPath: credentials
        optional: true
```

### Values Files

```yaml
charts:
  - name: my-release
    sourceType: helm-repo
    url: https://charts.example.com
    chartName: my-chart
    valuesFiles:
      - values-production.yaml
```

## Implementation Details

- Uses Helm CLI commands (requires helm in PATH)
- Charts installed sequentially in order
- Charts uninstalled in reverse order during cleanup
- Creates namespaces automatically if specified
- Supports custom release names
- Template expansion for environment variables

## Requirements

- Helm CLI must be installed and available in PATH
- Kubeconfig must be provided by testenv-kind
- Charts must be accessible (public repos or configured auth)

## See Also

- [Testenv-Helm-Install Configuration Schema](schema.md)
- [testenv MCP Server](../../testenv/docs/usage.md)
- [testenv-kind MCP Server](../../testenv-kind/docs/usage.md)
- [testenv-lcr MCP Server](../../testenv-lcr/docs/usage.md)
