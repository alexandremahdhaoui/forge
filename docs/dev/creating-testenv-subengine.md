# Creating Testenv Subengines

**Provision specific resources for test environments.**

> "I needed PostgreSQL for integration tests. I created a subengine that starts a container and writes credentials to tmpDir. Now every test environment gets a fresh database."

## What problem does this solve?

A testenv subengine provisions one specific resource (database, cache, mock service). Subengines are composed by the testenv orchestrator to build complete environments.

## What is a testenv subengine?

A subengine:
- Creates a specific resource (database, cluster, mock API)
- Writes configuration files to the shared tmpDir
- Returns metadata for test runners to consume
- Cleans up resources on delete

**Built-in subengines:** testenv-kind, testenv-lcr, testenv-helm-install

## What MCP tools must it provide?

| Tool | Required | Description |
|------|----------|-------------|
| `create` | Yes | Create resource, return files/metadata |
| `delete` | Yes | Clean up resource (best-effort) |
| `config-validate` | Yes | Validate configuration |

**create input/output:**
```go
type CreateInput struct {
    TestID string `json:"testID"` // Unique test environment ID
    Stage  string `json:"stage"`  // Test stage name
    TmpDir string `json:"tmpDir"` // Shared temporary directory
}

type CreateOutput struct {
    TestID           string            `json:"testID"`
    Files            map[string]string `json:"files"`            // Relative paths in tmpDir
    Metadata         map[string]string `json:"metadata"`         // Key-value pairs
    ManagedResources []string          `json:"managedResources"` // For cleanup
}
```

## How does it integrate with testenv?

The orchestrator calls subengines in order during create, **reverse order** during delete.

**Naming convention:** Prefix files/metadata keys with engine name (e.g., `testenv-postgres.credentials`).

## How do I implement a subengine?

Use forge-dev. Create `forge-dev.yaml` and `spec.openapi.yaml` (see [forge-dev.md](./forge-dev.md)), then implement:

```go
func Create(ctx context.Context, input engineframework.CreateInput, spec *Spec) (*engineframework.TestEnvArtifact, error) {
    containerID := startPostgres(spec.Image, spec.Port)
    credsPath := filepath.Join(input.TmpDir, "db-credentials.yaml")
    writeCredentials(credsPath, containerID)

    return &engineframework.TestEnvArtifact{
        TestID: input.TestID,
        Files: map[string]string{"testenv-postgres.credentials": "db-credentials.yaml"},
        Metadata: map[string]string{"testenv-postgres.containerID": containerID},
        ManagedResources: []string{credsPath, containerID},
    }, nil
}

func Delete(ctx context.Context, input engineframework.DeleteInput, spec *Spec) error {
    stopContainer(input.TestID) // Best-effort - don't fail if already gone
    return nil
}
```

**Configure in forge.yaml:**
```yaml
test:
  - name: integration
    testenv:
      engine: go://testenv
      subengines:
        - go://testenv-kind
        - go://testenv-postgres
    spec:
      image: postgres:15
```
