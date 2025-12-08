# Testenv Usage Guide

## Purpose

`testenv` is a forge engine for orchestrating test environments. It coordinates testenv subengines (testenv-kind, testenv-lcr, testenv-helm-install) to create complete test environments with Kind clusters, local container registries, and Helm charts.

## Invocation

### CLI Mode

Run directly as a standalone command:

```bash
testenv create <STAGE>
testenv delete <TEST-ID>
```

Examples:
```bash
testenv create integration
testenv delete test-integration-20250106-abc123
```

### MCP Mode

Run as an MCP server:

```bash
testenv --mcp
```

Forge invokes this automatically when using:

```yaml
engine: go://testenv
```

## Available MCP Tools

### `create`

Create a complete test environment.

**Input Schema:**
```json
{
  "stage": "string (required)"
}
```

**Output:**
```json
{
  "testID": "string"
}
```

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "create",
    "arguments": {
      "stage": "integration"
    }
  }
}
```

### `delete`

Delete a test environment by ID.

**Input Schema:**
```json
{
  "testID": "string (required)"
}
```

**Output:**
```json
{
  "success": true,
  "message": "Deleted test environment: test-integration-20250106-abc123"
}
```

### `docs-list`

List all available documentation for testenv.

### `docs-get`

Get a specific documentation by name.

**Input Schema:**
```json
{
  "name": "string (required)"
}
```

### `docs-validate`

Validate documentation completeness.

## Common Use Cases

### Basic Test Environment

Create a test environment with Kind cluster and local container registry:

```yaml
test:
  - name: integration
    stage: integration
    testenv: go://testenv
    runner: go://go-test
```

Run with:

```bash
forge test create integration
forge test run integration
forge test delete integration
```

### Custom Testenv Configuration

Use the engines configuration for more control:

```yaml
engines:
  - alias: my-testenv
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-lcr
        spec:
          enabled: true
          namespace: testenv-lcr
          imagePullSecretNamespaces:
            - default
            - my-app
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
```

### Environment with Pre-loaded Images

Configure testenv-lcr to push images to the local registry:

```yaml
engines:
  - alias: integration-testenv
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-lcr
        spec:
          enabled: true
          images:
            - name: local://myapp:latest
            - name: quay.io/example/img:v1.2.3
```

## Implementation Details

- Generates unique test IDs: `test-{stage}-{date}-{random}`
- Creates temporary directory: `.forge/tmp/{testID}`
- Coordinates subengine execution in order
- Stores TestEnvironment metadata in artifact store
- Cleans up resources in reverse order on delete

## Test Environment Lifecycle

1. **Create**: `testenv create <stage>`
   - Generate unique testID
   - Create tmpDir for test files
   - Execute subengines (kind, lcr, helm-install)
   - Store TestEnvironment in artifact store

2. **List**: `forge test list <stage>`
   - Read artifact store directly (not via MCP)

3. **Get**: `forge test get <stage> <testID>`
   - Read artifact store directly (not via MCP)

4. **Delete**: `testenv delete <testID>`
   - Execute subengines in reverse order
   - Clean up tmpDir
   - Remove from artifact store

## See Also

- [Testenv Configuration Schema](schema.md)
- [testenv-kind MCP Server](../../testenv-kind/docs/usage.md)
- [testenv-lcr MCP Server](../../testenv-lcr/docs/usage.md)
- [testenv-helm-install MCP Server](../../testenv-helm-install/docs/usage.md)
