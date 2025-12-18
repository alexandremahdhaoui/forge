# parallel-builder Configuration

Parallel builder that runs multiple sub-builders concurrently

> Full OpenAPI specification: [spec.openapi.yaml](../spec.openapi.yaml)

## Fields

### `builders`

- **Type:** `array of interface{}`
- **Required:** Yes
- **Description:** List of sub-builder configurations to run in parallel. Each item must be an object with 'engine' (required), 'name' (optional), and 'spec' (optional) fields.

