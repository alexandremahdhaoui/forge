# testenv-kind Configuration

Kind (Kubernetes IN Docker) cluster manager for test environments

> Full OpenAPI specification: [spec.openapi.yaml](../spec.openapi.yaml)

## Fields

### `config`

- **Type:** `string`
- **Required:** No
- **Description:** Path to kind config file for cluster customization

### `image`

- **Type:** `string`
- **Required:** No
- **Description:** Kind node image to use (e.g., kindest/node:v1.27.0)

### `name`

- **Type:** `string`
- **Required:** No
- **Description:** Custom name suffix for the kind cluster (default uses test ID)

### `retain`

- **Type:** `boolean`
- **Required:** No
- **Description:** Whether to retain the cluster on failure for debugging

### `waitTimeout`

- **Type:** `string`
- **Required:** No
- **Description:** Timeout for waiting for cluster to be ready (e.g., 5m)

