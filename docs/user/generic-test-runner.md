# generic-test-runner

**Wrap any CLI command as a forge test runner.**

> "I wanted golangci-lint and shellcheck as test stages. Generic-test-runner made it a 5-line config change."

## What problem does generic-test-runner solve?

You have validation tools (linters, security scanners, compliance checks) that exit with code 0 on success and non-zero on failure. You want them as forge test stages without writing custom Go code. Generic-test-runner wraps any command as a test runner using YAML configuration.

## How do I configure generic-test-runner?

Define a test-runner alias in forge.yaml, then reference it in your test specs:

```yaml
engines:
  - alias: golangci
    type: test-runner
    testRunner:
      - engine: go://generic-test-runner
        spec:
          command: "golangci-lint"
          args: ["run", "./..."]

test:
  - name: lint
    runner: alias://golangci
```

Run with: `forge test lint run`

## What configuration options are available?

| Option | Required | Description |
|--------|----------|-------------|
| `command` | Yes | Executable to run (in PATH or full path) |
| `args` | No | Array of command arguments |
| `env` | No | Environment variables as key-value map |
| `envFile` | No | Path to .envrc file with environment variables |
| `workDir` | No | Working directory for execution |

**Exit code interpretation:**
- Exit 0 = test passed
- Exit non-zero = test failed

## When should I use a built-in runner instead?

Use built-in runners when available:
- **Go tests**: `go://go-test`
- **Go linting**: `go://go-lint`
- **Build tag verification**: `go://go-lint-tags`

Use generic-test-runner when no built-in exists for your tool.

## Quick examples

**Security scanner:**
```yaml
engines:
  - alias: gosec
    type: test-runner
    testRunner:
      - engine: go://generic-test-runner
        spec:
          command: "gosec"
          args: ["-quiet", "./..."]

test:
  - name: security
    runner: alias://gosec
```

**Shell script linter:**
```yaml
engines:
  - alias: shellcheck
    type: test-runner
    testRunner:
      - engine: go://generic-test-runner
        spec:
          command: "shellcheck"
          args: ["scripts/*.sh"]

test:
  - name: shell-lint
    runner: alias://shellcheck
```

**Python tests:**
```yaml
engines:
  - alias: pytest
    type: test-runner
    testRunner:
      - engine: go://generic-test-runner
        spec:
          command: "pytest"
          args: ["--verbose", "tests/"]
          workDir: "./python-service"

test:
  - name: python-tests
    runner: alias://pytest
```

## How is this different from generic-builder?

| Aspect | generic-builder | generic-test-runner |
|--------|----------------|---------------------|
| Used in | `build:` section | `test:` section |
| Output | Artifact | TestReport |
| Exit non-zero | Build fails | Test fails (valid report) |

## How do I debug a failing test?

1. Run the command manually to see full output
2. Check exit code: `echo $?`
3. Add verbose flags to args for more detail

## What's next?

- [generic-builder](./generic-builder.md) - Wrap CLI tools as build engines
- [Schema Reference](./forge-yaml-schema.md) - Full forge.yaml options
