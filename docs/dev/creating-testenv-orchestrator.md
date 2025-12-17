# Creating Testenv Orchestrators

**Compose subengines into complete test environments.**

> "The default testenv works for most cases. I only needed a custom orchestrator when I required conditional subengine selection based on the test stage."

## What problem does this solve?

A testenv orchestrator composes multiple subengines to create complete test environments. The default `testenv` handles most cases. Create a custom orchestrator only for complex composition logic.

## Table of Contents

- [What is a testenv orchestrator?](#what-is-a-testenv-orchestrator)
- [What MCP tools must it provide?](#what-mcp-tools-must-it-provide)
- [What is the TestEnvironment type?](#what-is-the-testenvironment-type)
- [When to create a custom orchestrator?](#when-to-create-a-custom-orchestrator)

## What is a testenv orchestrator?

An orchestrator:
- Composes multiple testenv subengines
- Manages a shared tmpDir for file isolation
- Coordinates create/delete across subengines
- Aggregates metadata and files from all subengines

## What MCP tools must it provide?

| Tool | Required | Description |
|------|----------|-------------|
| `create` | Yes | Create environment, returns `{testID: string}` |
| `delete` | Yes | Delete environment by testID |
| `get` | Yes | Get environment details |
| `list` | Yes | List all environments for a stage |

**create flow:**
1. Generate unique testID: `test-{stage}-{date}-{random}`
2. Create tmpDir at `.forge/tmp/{testID}/`
3. Call subengines in order with (testID, stage, tmpDir)
4. Aggregate files, metadata, managedResources
5. Store TestEnvironment in artifact store

**delete flow:**
1. Read TestEnvironment from artifact store
2. Call subengines in **reverse order** for cleanup
3. Remove tmpDir
4. Delete from artifact store

## What is the TestEnvironment type?

```go
type TestEnvironment struct {
    ID               string            `json:"id"`               // test-stage-date-random
    Name             string            `json:"name"`             // Test stage name
    Status           string            `json:"status"`           // created, running, passed, failed
    CreatedAt        time.Time         `json:"createdAt"`
    UpdatedAt        time.Time         `json:"updatedAt"`
    TmpDir           string            `json:"tmpDir"`           // Shared directory
    Files            map[string]string `json:"files"`            // Logical name -> filename
    Metadata         map[string]string `json:"metadata"`         // Key-value pairs
    ManagedResources []string          `json:"managedResources"` // Resources to clean up
    Env              map[string]string `json:"env"`              // Environment variables
}
```

## When to create a custom orchestrator?

**Use default testenv when:**
- Subengines run in fixed order
- No conditional logic needed
- Standard cleanup is sufficient

**Create custom orchestrator when:**
- Complex conditional subengine selection
- Dynamic ordering based on configuration
- Advanced retry logic
- Multi-cloud orchestration

**Most users should create subengines, not orchestrators.** See [creating-testenv-subengine.md](./creating-testenv-subengine.md).
