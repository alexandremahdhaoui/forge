# go-gen-bpf Configuration

BPF code generator using bpf2go

> Full OpenAPI specification: [spec.openapi.yaml](../spec.openapi.yaml)

## Fields

### `bpf2goVersion`

- **Type:** `string`
- **Required:** No
- **Description:** Version of bpf2go tool (default "latest")

### `cc`

- **Type:** `string`
- **Required:** No
- **Description:** C compiler binary (default bpf2go default)

### `cflags`

- **Type:** `array of string`
- **Required:** No
- **Description:** C compiler flags

### `goPackage`

- **Type:** `string`
- **Required:** No
- **Description:** Go package name (default basename of dest)

### `ident`

- **Type:** `string`
- **Required:** Yes
- **Description:** Go identifier for generated types (required)

### `outputDir`

- **Type:** `string`
- **Required:** No
- **Description:** Directory for generated Go code (optional)

### `outputStem`

- **Type:** `string`
- **Required:** No
- **Description:** Filename prefix (default "zz_generated")

### `sourceDir`

- **Type:** `string`
- **Required:** No
- **Description:** Directory containing BPF C source files (optional)

### `tags`

- **Type:** `array of string`
- **Required:** No
- **Description:** Build tags (default ["linux"])

### `types`

- **Type:** `array of string`
- **Required:** No
- **Description:** Specific types to generate (default all)

