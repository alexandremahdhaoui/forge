# forge-dev Configuration Schema

## Overview

This document describes the configuration files required for `forge-dev` code generation. Each engine using forge-dev needs two configuration files in its directory:

1. **forge-dev.yaml** - Engine metadata and generation settings
2. **spec.openapi.yaml** - OpenAPI 3.0 schema defining the Spec structure

## forge-dev.yaml Schema

```yaml
# Required: Engine name (lowercase alphanumeric with hyphens)
name: my-engine

# Required: Engine type
# Values: builder, test-runner, testenv-subengine
type: builder

# Required: Engine version (semver format: x.y.z)
version: 0.15.0

# Optional: Human-readable description
description: My custom build engine

# Required: OpenAPI configuration
openapi:
  # Required: Relative path to OpenAPI spec file
  specPath: ./spec.openapi.yaml

# Required: Code generation settings
generate:
  # Required: Go package name for generated code
  packageName: main
```

### Field Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Engine name. Must be lowercase alphanumeric with hyphens, starting with a letter. Max 64 characters. |
| `type` | enum | Yes | Engine type. One of: `builder`, `test-runner`, `testenv-subengine` |
| `version` | string | Yes | Semantic version in format `x.y.z` |
| `description` | string | No | Human-readable description of the engine |
| `openapi.specPath` | string | Yes | Relative path to the OpenAPI spec file |
| `generate.packageName` | string | Yes | Go package name for generated files. Must be a valid Go identifier. |

### Engine Types

**builder**: For build engines that produce artifacts.
- Generated function signature: `BuildFunc(ctx, input mcptypes.BuildInput, spec *Spec) (*forge.Artifact, error)`
- Registers: `build`, `buildBatch`, `config-validate` tools

**test-runner**: For test runner engines.
- Generated function signature: `TestRunnerFunc(ctx, input mcptypes.RunInput, spec *Spec) (*forge.TestReport, error)`
- Registers: `run`, `config-validate` tools

**testenv-subengine**: For test environment subengines.
- Generated function signatures:
  - `CreateFunc(ctx, input engineframework.CreateInput, spec *Spec) (*engineframework.TestEnvArtifact, error)`
  - `DeleteFunc(ctx, input engineframework.DeleteInput, spec *Spec) error`
- Registers: `create`, `delete`, `config-validate` tools

## spec.openapi.yaml Schema

The OpenAPI spec file must define a `Spec` schema under `components.schemas`:

```yaml
openapi: 3.0.3
info:
  title: my-engine Spec Schema
  version: 0.15.0
components:
  schemas:
    Spec:
      type: object
      properties:
        # Define your spec fields here
        fieldName:
          type: string
          description: Field description
        # ... more fields
      required:
        - fieldName  # List required fields
```

### Supported Property Types

#### String

```yaml
myString:
  type: string
  description: A string value
```

Go type: `string`

#### Boolean

```yaml
myBool:
  type: boolean
  description: A boolean flag
```

Go type: `bool`

#### Integer

```yaml
myInt:
  type: integer
  description: An integer value
```

Go type: `int`

#### Number (Float)

```yaml
myFloat:
  type: number
  description: A floating-point value
```

Go type: `float64`

#### String Array

```yaml
myStringArray:
  type: array
  items:
    type: string
  description: An array of strings
```

Go type: `[]string`

#### Integer Array

```yaml
myIntArray:
  type: array
  items:
    type: integer
  description: An array of integers
```

Go type: `[]int`

#### String Map

```yaml
myStringMap:
  type: object
  additionalProperties:
    type: string
  description: A map of string to string
```

Go type: `map[string]string`

#### Enum

```yaml
myEnum:
  type: string
  enum:
    - value1
    - value2
    - value3
  description: An enumerated string
```

Go type: `string` (with validation that value is in the allowed set)

### Required Fields

Use the `required` array at the Spec level to mark required fields:

```yaml
Spec:
  type: object
  properties:
    requiredField:
      type: string
    optionalField:
      type: string
  required:
    - requiredField
```

Required fields:
- Must be provided in configuration
- Generate validation errors if missing
- Do not include `omitempty` in JSON tags

### Default Values

```yaml
myField:
  type: string
  default: "default-value"
```

Default values are currently stored in the schema but not automatically applied. Engines should check for zero values and apply defaults manually.

## Generated Files

### zz_generated.spec.go

Contains:
- `Spec` struct with JSON tags
- `FromMap(map[string]interface{}) (*Spec, error)` - Parse map to Spec
- `ToMap() map[string]interface{}` - Convert Spec to map

### zz_generated.validate.go

Contains:
- `Validate(spec *Spec) *mcptypes.ConfigValidateOutput` - Validate Spec struct
- `ValidateMap(m map[string]interface{}) *mcptypes.ConfigValidateOutput` - Parse and validate map

### zz_generated.mcp.go

Contains:
- Type-safe function type (e.g., `BuildFunc`, `TestRunnerFunc`)
- `SetupMCPServer(version string, fn TypedFunc) (*mcpserver.Server, error)`
- Wrapper functions that handle parsing and validation

## Examples

### Minimal Configuration

**forge-dev.yaml:**
```yaml
name: simple-engine
type: builder
version: 0.1.0
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
```

**spec.openapi.yaml:**
```yaml
openapi: 3.0.3
info:
  title: simple-engine
  version: 0.1.0
components:
  schemas:
    Spec:
      type: object
```

### Full Configuration

**forge-dev.yaml:**
```yaml
name: advanced-engine
type: builder
version: 0.15.0
description: An advanced build engine with many options
openapi:
  specPath: ./spec.openapi.yaml
generate:
  packageName: main
```

**spec.openapi.yaml:**
```yaml
openapi: 3.0.3
info:
  title: advanced-engine Spec Schema
  version: 0.15.0
components:
  schemas:
    Spec:
      type: object
      properties:
        outputDir:
          type: string
          description: Directory for generated output
        verbose:
          type: boolean
          description: Enable verbose logging
          default: false
        logLevel:
          type: string
          enum:
            - debug
            - info
            - warn
            - error
          description: Logging level
          default: info
        tags:
          type: array
          items:
            type: string
          description: Build tags to include
        env:
          type: object
          additionalProperties:
            type: string
          description: Environment variables
      required:
        - outputDir
```

## Limitations

The following OpenAPI features are **not supported** in v1:

- `$ref` references to other schemas
- `oneOf`, `anyOf`, `allOf` combinators
- Arrays of objects (nested object types)
- Deeply nested objects (only one level of nesting)

If your engine requires these features, it cannot use forge-dev and must implement manual Spec handling.

## See Also

- [forge-dev Usage Guide](usage.md)
- [OpenAPI 3.0 Specification](https://spec.openapis.org/oas/v3.0.3)
