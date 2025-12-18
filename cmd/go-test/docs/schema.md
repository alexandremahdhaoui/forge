# go-test Configuration

Go test runner with coverage and JUnit reporting

> Full OpenAPI specification: [spec.openapi.yaml](../spec.openapi.yaml)

## Fields

### `args`

- **Type:** `array of string`
- **Required:** No
- **Description:** Additional arguments to pass to go test (optional)

### `cover`

- **Type:** `boolean`
- **Required:** No
- **Description:** Enable coverage (optional)

### `coverprofile`

- **Type:** `string`
- **Required:** No
- **Description:** Coverage profile output path (optional)

### `env`

- **Type:** `map[string]string`
- **Required:** No
- **Description:** Environment variables to set for tests (optional)

### `packages`

- **Type:** `array of string`
- **Required:** No
- **Description:** Packages to test (optional, defaults to ./...)

### `race`

- **Type:** `boolean`
- **Required:** No
- **Description:** Enable race detector (optional)

### `tags`

- **Type:** `array of string`
- **Required:** No
- **Description:** Build tags to use (optional)

### `timeout`

- **Type:** `string`
- **Required:** No
- **Description:** Test timeout (optional, e.g., "10m")

