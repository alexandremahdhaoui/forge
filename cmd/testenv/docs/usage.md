# testenv

**Orchestrate complete test environments with a single command.**

> "I was spending hours manually setting up Kind clusters, registries, and Helm charts for each test run. With testenv, I define my environment once in forge.yaml and it handles everything - creation, coordination between components, and cleanup."

## What problem does testenv solve?

Test environments require multiple coordinated components: Kubernetes clusters, container registries, and pre-installed applications. testenv orchestrates these subengines (testenv-kind, testenv-lcr, testenv-helm-install) so you can create, manage, and destroy complete environments with simple commands.

## How do I use testenv?

Add testenv to your forge.yaml:

```yaml
test:
  - name: integration
    stage: integration
    testenv: go://testenv
    runner: go://go-test
```

Run with:

```bash
forge test create integration     # Create environment
forge test run integration        # Run tests
forge test delete integration     # Cleanup
```

## How do I customize my test environment?

Define an engine alias with specific subengines and configuration:

```yaml
engines:
  - alias: my-testenv
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-lcr
        spec:
          enabled: true
          imagePullSecretNamespaces: [default, my-app]
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

test:
  - name: integration
    testenv: alias://my-testenv
    runner: go://go-test
```

## What is the environment lifecycle?

| Operation | Command | What happens |
|-----------|---------|--------------|
| Create | `forge test create <stage>` | Generate testID, create tmpDir, execute subengines in order |
| List | `forge test list <stage>` | Show all environments for a stage |
| Get | `forge test get <stage> <id>` | Show environment details |
| Delete | `forge test delete <stage> <id>` | Execute subengines in reverse, cleanup tmpDir |

## How does testenv coordinate subengines?

Subengines execute in order during create, reverse order during delete. Each subengine receives:
- `testID`: Unique identifier (`test-{stage}-{date}-{random}`)
- `tmpDir`: Temporary directory for test files
- `metadata`: Accumulated from previous subengines
- `env`: Environment variables from previous subengines

This allows testenv-lcr to use the kubeconfig from testenv-kind, and testenv-helm-install to use both.

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../MCP.md) - MCP tool documentation
- [testenv-kind](../../testenv-kind/docs/usage.md) - Kind cluster subengine
- [testenv-lcr](../../testenv-lcr/docs/usage.md) - Local container registry subengine
- [testenv-helm-install](../../testenv-helm-install/docs/usage.md) - Helm chart installer subengine
