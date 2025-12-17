# go-gen-openapi

**Generate Go client and server code from OpenAPI specifications.**

> "We were manually writing HTTP clients that drifted from our API specs. go-gen-openapi generates type-safe clients and servers directly from our OpenAPI YAML - now our Go code is always in sync with the spec."

## What problem does go-gen-openapi solve?

Hand-writing HTTP clients and server handlers is error-prone and diverges from API specifications over time. go-gen-openapi uses oapi-codegen to generate type-safe Go code from OpenAPI specs, with automatic dependency tracking for lazy rebuild.

## How do I use go-gen-openapi?

Add a build target to `forge.yaml`:

```yaml
build:
  - name: example-api-v1
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/example-api.v1.yaml
      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: exampleclient
```

Run the generator:

```bash
forge build
```

## What configuration options are available?

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Build step name |
| `spec.sourceFile` | Yes | Path to OpenAPI spec file |
| `spec.destinationDir` | Yes | Output directory for generated code |
| `spec.client.enabled` | No | Generate client code |
| `spec.client.packageName` | No | Package name for client |
| `spec.server.enabled` | No | Generate server code |
| `spec.server.packageName` | No | Package name for server |

## How do I generate both client and server?

```yaml
build:
  - name: products-api-v1
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/products-api.v1.yaml
      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: productsclient
      server:
        enabled: true
        packageName: productsserver
```

This generates:
- `./pkg/generated/productsclient/zz_generated.oapi-codegen.go`
- `./pkg/generated/productsserver/zz_generated.oapi-codegen.go`

## How do I handle multiple API versions?

Each version needs a separate build spec:

```yaml
build:
  - name: example-api-v1
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/example-api.v1.yaml
      destinationDir: ./pkg/generated
      client: { enabled: true, packageName: exampleclient }

  - name: example-api-v2
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/example-api.v2.yaml
      destinationDir: ./pkg/generated
      client: { enabled: true, packageName: exampleclientv2 }
```

## What environment variables are available?

| Variable | Default | Description |
|----------|---------|-------------|
| `OAPI_CODEGEN_VERSION` | `v2.3.0` | Version of oapi-codegen to use |

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../MCP.md) - MCP tool documentation
- [oapi-codegen docs](https://github.com/oapi-codegen/oapi-codegen) - Upstream documentation
