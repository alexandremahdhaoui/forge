# Migrating from Makefile

**Replace your Makefile with declarative YAML.**

> "I converted our 200-line Makefile to a 50-line forge.yaml in an afternoon. The team loves the simpler syntax."

## Why migrate from Makefile?

| Makefile | Forge |
|----------|-------|
| Imperative shell scripts | Declarative YAML |
| Tab-sensitivity causes errors | Standard YAML syntax |
| Manual dependency tracking | Automatic lazy rebuild |
| No artifact history | Built-in artifact store |
| Shell-specific portability issues | Cross-platform engines |

## How do I map Makefile targets to forge?

| Makefile Target | Forge Equivalent |
|-----------------|------------------|
| `make build` | `forge build` |
| `make test` | `forge test unit run` |
| `make lint` | `forge test lint run` |
| `make fmt` | Build step with formatter engine |
| `make clean` | `rm -rf build/` |
| `make all` | `forge build` or `forge test-all` |

## Before and after

**Makefile:**
```makefile
.PHONY: all build test lint fmt

build:
	go build -o bin/myapp ./cmd/myapp

test:
	go test -v ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -l -w .
```

**forge.yaml:**
```yaml
name: myproject
artifactStorePath: .forge/artifacts.yaml

engines:
  - alias: formatter
    type: builder
    builder:
      - engine: go://generic-builder
        spec:
          command: "gofmt"
          args: ["-l", "-w", "."]

  - alias: linter
    type: test-runner
    testRunner:
      - engine: go://generic-test-runner
        spec:
          command: "golangci-lint"
          args: ["run", "./..."]

build:
  - name: format-code
    src: .
    engine: alias://formatter

  - name: myapp
    src: ./cmd/myapp
    dest: ./bin
    engine: go://go-build

test:
  - name: unit
    runner: go://go-test

  - name: lint
    runner: alias://linter
```

## How do I handle environment variables?

**Makefile:**
```makefile
build:
	CGO_ENABLED=0 GOOS=linux go build -o bin/myapp ./cmd/myapp
```

**forge.yaml:**
```yaml
engines:
  - alias: linux-builder
    type: builder
    builder:
      - engine: go://generic-builder
        spec:
          command: "go"
          args: ["build", "-o", "bin/myapp", "./cmd/myapp"]
          env:
            CGO_ENABLED: "0"
            GOOS: "linux"

build:
  - name: myapp
    src: ./cmd/myapp
    engine: alias://linux-builder
```

## How do I handle complex shell logic?

Extract to a script and wrap with generic-builder:

```yaml
engines:
  - alias: complex-build
    type: builder
    builder:
      - engine: go://generic-builder
        spec:
          command: "./scripts/complex-build.sh"
          args: ["--env=prod"]
```

## Migration checklist

- [ ] Map build targets to `build:` section
- [ ] Map test targets to `test:` section
- [ ] Convert environment variables to `env:` or `.envrc`
- [ ] Extract complex scripts and wrap with generic-builder
- [ ] Test incrementally: `forge build`, `forge test <stage> run`
- [ ] Update CI/CD to use forge commands
- [ ] Delete Makefile (or keep thin wrapper during transition)

## Transition wrapper

Keep `make` commands during migration:

```makefile
.PHONY: build test lint
build:
	forge build
test:
	forge test unit run
lint:
	forge test lint run
```

## What's next?

- [generic-builder](./generic-builder.md) - Wrap CLI tools
- [generic-test-runner](./generic-test-runner.md) - Wrap test tools
- [Getting Started](./getting-started.md) - Basic setup
