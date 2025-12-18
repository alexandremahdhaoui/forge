# parallel-test-runner Configuration

Parallel test runner that runs multiple sub-runners concurrently

> Full OpenAPI specification: [spec.openapi.yaml](../spec.openapi.yaml)

## Fields

### `primaryCoverageRunner`

- **Type:** `string`
- **Required:** No
- **Description:** Name of the runner whose coverage is used. If not specified or runner not found, Coverage.Enabled=false in result.

### `runners`

- **Type:** `array of interface{}`
- **Required:** Yes
- **Description:** List of test runners to execute in parallel. Each item must be an object with 'name' (required), 'engine' (required), and 'spec' (optional) fields.

