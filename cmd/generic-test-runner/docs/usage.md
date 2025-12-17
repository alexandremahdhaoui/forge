# generic-test-runner

**Run any command as a test with structured reporting.**

> "I have security scanners, compliance checkers, and custom scripts that need to be part of my test pipeline. generic-test-runner wraps any command and gives me consistent TestReport output."

## What problem does generic-test-runner solve?

Not every test tool produces structured output. generic-test-runner executes any command and converts its exit code into a pass/fail TestReport, integrating custom tools into your forge test pipeline.

## How do I use generic-test-runner?

```yaml
test:
  - name: security
    stage: security
    runner: go://generic-test-runner
    spec:
      command: gosec
      args: ["./..."]
```

Run with:

```bash
forge test run security
```

## What configuration options are available?

| Option | Description |
|--------|-------------|
| `command` | Command to execute (required) |
| `args` | Command arguments as array |
| `env` | Environment variables as key-value pairs |
| `envFile` | Path to env file to load |
| `workDir` | Working directory for command execution |

## How is pass/fail determined?

- Exit code 0 = `status: "passed"`
- Exit code != 0 = `status: "failed"`

On failure, stdout/stderr is captured in `errorMessage`.

## What are common use cases?

Security scanner:
```yaml
test:
  - name: security
    runner: go://generic-test-runner
    spec:
      command: gosec
      args: ["-fmt=json", "./..."]
```

Custom test script:
```yaml
test:
  - name: custom
    runner: go://generic-test-runner
    spec:
      command: ./run-tests.sh
      args: ["--verbose"]
      env:
        DEBUG: "true"
```

Compliance checker:
```yaml
test:
  - name: compliance
    runner: go://generic-test-runner
    spec:
      command: compliance-checker
      workDir: ./compliance
```

## What output does it produce?

```json
{
  "stage": "security",
  "status": "passed",
  "duration": 3.2,
  "testStats": {
    "total": 1,
    "passed": 1,
    "failed": 0,
    "skipped": 0
  }
}
```

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../../MCP.md) - MCP tool documentation
