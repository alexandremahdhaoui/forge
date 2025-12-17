# testenv-stub

**A no-op testenv subengine for fast infrastructure testing.**

> "I needed to test my testenv orchestration logic without waiting 2 minutes for Kind clusters to spin up. testenv-stub gives me a mock environment in milliseconds - perfect for CI validation and unit testing the testenv workflow itself."

## What problem does testenv-stub solve?

Testing testenv infrastructure (create/list/get/delete workflows) requires spinning up real resources, which is slow. testenv-stub provides a lightweight mock that returns realistic metadata without provisioning actual resources, enabling fast iteration on testenv logic.

## How do I use testenv-stub?

Replace real subengines with testenv-stub in forge.yaml:

```yaml
engines:
  - alias: fast-testenv
    type: testenv
    testenv:
      - engine: go://testenv-stub

test:
  - name: e2e
    testenv: alias://fast-testenv
    runner: go://go-test
```

## When should I use testenv-stub?

| Scenario | Use testenv-stub? |
|----------|-------------------|
| Testing testenv orchestration | Yes |
| Fast unit tests | Yes |
| CI pipeline validation | Yes |
| Tests needing real Kubernetes | No - use testenv-kind |
| Tests needing container registry | No - use testenv-lcr |

## What does testenv-stub provide?

| Output | Value |
|--------|-------|
| `TESTENV_STUB_ACTIVE` env var | `true` |
| Marker file | `stub-marker.txt` in tmpDir |
| Metadata | Timestamps and test identifiers |

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../MCP.md) - MCP tool documentation
- [testenv-kind](../../testenv-kind/docs/usage.md) - Real Kubernetes clusters
- [testenv](../../testenv/docs/usage.md) - Testenv orchestrator
