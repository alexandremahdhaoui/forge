# CI-Orchestrator Usage Guide

## Purpose

`ci-orchestrator` is a forge engine for orchestrating CI/CD pipelines.

**Status: PLANNED - NOT YET IMPLEMENTED**

This is a placeholder for future CI/CD pipeline orchestration functionality. The tool structure is in place, but all operations currently return "not yet implemented" errors.

## Invocation

### CLI Mode

Show help and version:

```bash
ci-orchestrator help
ci-orchestrator version
```

### MCP Mode

Run as an MCP server:

```bash
ci-orchestrator --mcp
```

**Note:** All tool calls will return errors indicating the functionality is not yet implemented.

## Available MCP Tools

### `run` (Not Implemented)

Execute a CI pipeline.

**Current Status:** Returns error "ci-orchestrator: not yet implemented"

**Planned Input Schema:**
```json
{
  "pipeline": "string (required)"
}
```

**Planned Output:**
```json
{
  "status": "success|failed",
  "duration": 123.45,
  "steps": [
    {
      "name": "step name",
      "status": "success",
      "duration": 12.34
    }
  ]
}
```

**Example (will fail):**
```json
{
  "method": "tools/call",
  "params": {
    "name": "run",
    "arguments": {
      "pipeline": "build-and-test"
    }
  }
}
```

### `docs-list`

List all available documentation for ci-orchestrator.

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

## Planned Features

The ci-orchestrator is planned to provide:

- **Pipeline Execution**: Run multi-stage CI/CD pipelines
- **Stage Orchestration**: Coordinate build, test, and deployment stages
- **Artifact Management**: Track artifacts across pipeline stages
- **Integration with Forge**: Leverage forge's build and test infrastructure
- **MCP-Native**: Full MCP protocol support for AI agent integration
- **Parallel Execution**: Run independent stages concurrently
- **Failure Handling**: Configurable retry and rollback strategies

## Current Status

| Feature | Status |
|---------|--------|
| Binary scaffold | Complete |
| MCP server framework | Complete |
| Version command | Complete |
| Documentation | Complete |
| Pipeline execution | Not implemented |
| Configuration schema | Not defined |
| Integration with forge | Not implemented |

## Future Use Cases

### Basic Pipeline Execution (Planned)

```yaml
# Planned forge.yaml configuration
pipelines:
  - name: build-and-test
    stages:
      - name: build
        steps:
          - forge build all
      - name: test
        steps:
          - forge test run unit
          - forge test run integration
```

```bash
# Planned usage
ci-orchestrator run build-and-test
```

### Parallel Stage Execution (Planned)

```yaml
pipelines:
  - name: full-ci
    stages:
      - name: build
        steps:
          - forge build all
      - name: tests
        parallel:
          - forge test run unit
          - forge test run integration
          - forge lint all
```

### Deployment Pipeline (Planned)

```yaml
pipelines:
  - name: deploy
    stages:
      - name: build
        steps:
          - forge build all
      - name: test
        steps:
          - forge test run all
      - name: deploy
        environment: production
        approval: required
        steps:
          - kubectl apply -f manifests/
```

## Development

To contribute to ci-orchestrator implementation, see:

- [ARCHITECTURE.md](../../ARCHITECTURE.md) for design patterns
- [docs/prompts/](../../docs/prompts/) for engine creation guides
- `cmd/testenv/` for orchestration patterns

## See Also

- [CI-Orchestrator Configuration Schema](schema.md)
- [forge MCP Server](../../forge/MCP.md)
- [testenv MCP Server](../../testenv/docs/usage.md)
