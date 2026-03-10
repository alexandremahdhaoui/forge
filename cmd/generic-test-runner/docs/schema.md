# generic-test-runner Configuration

Execute any command as a test step

> Full OpenAPI specification: [spec.openapi.yaml](../spec.openapi.yaml)

## Fields

### `args`

- **Type:** `array of string`
- **Required:** No
- **Description:** Command arguments (optional)

### `command`

- **Type:** `string`
- **Required:** Yes
- **Description:** Command to execute (required)

### `context`

- **Type:** `string`
- **Required:** No
- **Description:** Context directory for command execution (optional)

### `env`

- **Type:** `map[string]string`
- **Required:** No
- **Description:** Environment variables (optional)

### `envFile`

- **Type:** `string`
- **Required:** No
- **Description:** Path to environment file (optional)

