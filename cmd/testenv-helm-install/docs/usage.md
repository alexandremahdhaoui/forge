# testenv-helm-install

**Pre-install Helm charts into your test environment.**

> "My integration tests needed cert-manager and ingress-nginx running before they could start. With testenv-helm-install, I declare the charts once and they're installed automatically - from Helm repos, Git, OCI, or local paths."

## What problem does testenv-helm-install solve?

Integration tests often depend on infrastructure components (cert-manager, ingress controllers, databases) that must be installed before tests run. testenv-helm-install installs Helm charts in order during environment creation and uninstalls them in reverse during cleanup.

## How do I use testenv-helm-install?

Add it to your testenv configuration after testenv-kind:

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

## What source types are supported?

| Source | Required Fields | Example |
|--------|-----------------|---------|
| `helm-repo` | `url`, `chartName` | Helm repository (charts.jetstack.io) |
| `local` | `path` | Local directory (./charts/my-app) |
| `git` | `url`, `chartPath` | Git repository with chart path |
| `oci` | `url` | OCI registry (oci://ghcr.io/...) |
| `s3` | `url`, `s3BucketName`, `chartPath` | S3 bucket |

## How do I configure chart values?

**Inline values:**
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
```

**Values files:**
```yaml
charts:
  - name: my-release
    sourceType: helm-repo
    url: https://charts.example.com
    chartName: my-chart
    valuesFiles:
      - values-production.yaml
```

## How do I use the local registry in chart values?

Reference testenv-lcr's registry via template expansion:

```yaml
- engine: go://testenv-lcr
  spec:
    enabled: true
- engine: go://testenv-helm-install
  spec:
    charts:
      - name: my-app
        sourceType: local
        path: ./charts/my-app
        values:
          image:
            repository: "{{.Env.TESTENV_LCR_FQDN}}/my-app"
```

## What are the requirements?

- Helm CLI installed and in PATH
- Kubeconfig provided by testenv-kind
- Charts accessible (public repos or configured auth)

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../MCP.md) - MCP tool documentation
- [testenv-kind](../../testenv-kind/docs/usage.md) - Required: provides the Kubernetes cluster
- [testenv-lcr](../../testenv-lcr/docs/usage.md) - Optional: local container registry
