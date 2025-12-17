# testenv-lcr

**Deploy a TLS-enabled container registry inside your Kind cluster.**

> "Pushing images to Docker Hub for every test was slow and cluttered my registry. testenv-lcr gives me a local, secure registry inside my test cluster - images push instantly and everything cleans up when I'm done."

## What problem does testenv-lcr solve?

Integration tests often need custom container images, but using external registries is slow and requires managing credentials. testenv-lcr deploys a secure, TLS-enabled registry inside your Kind cluster with auto-generated credentials and cert-manager integration.

## How do I use testenv-lcr?

Add it after testenv-kind in your testenv configuration:

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

## How do I create image pull secrets for my namespaces?

Specify namespaces that need to pull images from the registry:

```yaml
- engine: go://testenv-lcr
  spec:
    enabled: true
    imagePullSecretNamespaces:
      - default
      - my-app
      - system
```

## How do I pre-load images into the registry?

Push images during environment creation:

```yaml
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

## How do I reference the registry in Helm values?

Use template expansion with the `TESTENV_LCR_FQDN` environment variable:

```yaml
- engine: go://testenv-helm-install
  spec:
    charts:
      - name: my-app
        sourceType: local
        path: ./charts/my-app
        values:
          image:
            repository: "{{.Env.TESTENV_LCR_FQDN}}/my-app"
            tag: latest
```

## What environment variables does testenv-lcr provide?

| Variable | Example | Description |
|----------|---------|-------------|
| `TESTENV_LCR_FQDN` | `testenv-lcr.testenv-lcr.svc.cluster.local:31906` | Full registry address with port |
| `TESTENV_LCR_HOST` | `testenv-lcr.testenv-lcr.svc.cluster.local` | Registry hostname |
| `TESTENV_LCR_PORT` | `31906` | Dynamic port number |
| `TESTENV_LCR_NAMESPACE` | `testenv-lcr` | Kubernetes namespace |
| `TESTENV_LCR_CA_CERT` | `/path/to/ca.crt` | Path to CA certificate |

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../MCP.md) - MCP tool documentation
- [testenv-kind](../../testenv-kind/docs/usage.md) - Required: provides the Kubernetes cluster
- [testenv-helm-install](../../testenv-helm-install/docs/usage.md) - Install charts using registry images
