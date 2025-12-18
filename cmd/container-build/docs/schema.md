# container-build Configuration

Build container images using Docker, Podman, or Kaniko

> Full OpenAPI specification: [spec.openapi.yaml](../spec.openapi.yaml)

## Fields

### `buildArgs`

- **Type:** `map[string]string`
- **Required:** No
- **Description:** Build arguments (optional)

### `context`

- **Type:** `string`
- **Required:** No
- **Description:** Build context path (optional)

### `dockerfile`

- **Type:** `string`
- **Required:** No
- **Description:** Path to Dockerfile (optional)

### `push`

- **Type:** `boolean`
- **Required:** No
- **Description:** Whether to push image (optional)

### `registry`

- **Type:** `string`
- **Required:** No
- **Description:** Registry URL (optional)

### `tags`

- **Type:** `array of string`
- **Required:** No
- **Description:** Image tags (optional)

### `target`

- **Type:** `string`
- **Required:** No
- **Description:** Build target stage (optional)

