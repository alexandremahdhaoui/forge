# Generic Builder Configuration Schema

## Overview

This document describes the configuration options for `generic-builder` in `forge.yaml`. The generic-builder engine executes arbitrary shell commands as build steps.

## Basic Configuration

```yaml
build:
  - name: my-command
    command: echo
    args: ["Hello, World!"]
    engine: go://generic-builder
```

## Configuration Options

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | The name of the build step. |
| `command` | string | The shell command to execute. |
| `engine` | string | Must be `go://generic-builder` to use this builder. |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `args` | array of strings | `[]` | Command arguments (supports Go templates). |
| `env` | map of string to string | `{}` | Environment variables to set for the command. |
| `envFile` | string | - | Path to an environment file to load. |
| `workDir` | string | `.` | Working directory for command execution. |
| `src` | string | - | Source directory (available in templates). |
| `dest` | string | - | Destination directory (available in templates). |
| `version` | string | - | Version string (available in templates). |

## Template Variables

Arguments support Go template syntax:

| Variable | Description |
|----------|-------------|
| `{{ .Name }}` | Build step name |
| `{{ .Src }}` | Source directory |
| `{{ .Dest }}` | Destination directory |
| `{{ .Version }}` | Version string |

## Examples

### Minimal Configuration

```yaml
build:
  - name: echo-test
    command: echo
    args: ["Hello"]
    engine: go://generic-builder
```

### Full Configuration

```yaml
build:
  - name: generate-code
    command: protoc
    args:
      - "--go_out={{ .Dest }}"
      - "--go_opt=paths=source_relative"
      - "{{ .Src }}/service.proto"
    src: ./proto
    dest: ./pkg/api
    workDir: .
    env:
      PATH: "/usr/local/bin:${PATH}"
    engine: go://generic-builder
```

### Using Environment File

```yaml
build:
  - name: build-with-secrets
    command: ./scripts/deploy.sh
    envFile: .env.production
    engine: go://generic-builder
```

### Multiple Build Steps

```yaml
build:
  - name: lint
    command: golangci-lint
    args: ["run", "./..."]
    engine: go://generic-builder

  - name: format
    command: gofumpt
    args: ["-w", "."]
    engine: go://generic-builder

  - name: test
    command: go
    args: ["test", "./..."]
    engine: go://generic-builder
```

### Cross-Platform Build Script

```yaml
build:
  - name: build-all
    command: make
    args: ["all"]
    env:
      GOOS: linux
      GOARCH: amd64
    engine: go://generic-builder
```

## Generated Artifacts

Each successful build creates an artifact entry:

```yaml
artifacts:
  - name: my-command
    type: command-output
    location: "."
    timestamp: "2024-01-15T10:30:00Z"
    version: "echo-exit0"
```

## Default Behavior

When optional fields are not provided:

- `workDir` defaults to current directory
- `location` in artifact is set to `workDir`, then `src`, then `.`
- `version` is set to `{command}-exit{code}`

## Error Handling

- Non-zero exit codes cause build failure
- Error message includes exit code, stdout, and stderr
- All environment variables are passed to the subprocess

## See Also

- [Generic Builder Usage Guide](usage.md)
- [Forge Build Documentation](../../docs/forge-usage.md#building-artifacts)
