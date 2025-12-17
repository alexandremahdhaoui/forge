# go-lint-tags Configuration

Verify test files have proper build tags (unit, integration, e2e)

> Full OpenAPI specification: [spec.openapi.yaml](../spec.openapi.yaml)

## Fields

### `expectedTags`

- **Type:** `array of string`
- **Required:** No
- **Description:** List of expected build tags to check for (default is unit, integration, e2e)

### `rootDir`

- **Type:** `string`
- **Required:** No
- **Description:** Root directory to scan for test files (default is current directory)

