# go-gen-protobuf Configuration

Protocol Buffer code generator for Go

> Full OpenAPI specification: [spec.openapi.yaml](../spec.openapi.yaml)

## Fields

### `extraArgs`

- **Type:** `array of string`
- **Required:** No
- **Description:** Additional raw protoc arguments

### `goGrpcOpt`

- **Type:** `string`
- **Required:** No
- **Description:** --go-grpc_opt value (default "paths=source_relative")

### `goOpt`

- **Type:** `string`
- **Required:** No
- **Description:** --go_opt value (default "paths=source_relative")

### `includes`

- **Type:** `array of string`
- **Required:** No
- **Description:** Additional include paths for protoc (optional)

### `outputDir`

- **Type:** `string`
- **Required:** No
- **Description:** Directory for generated Go code (optional)

### `plugin`

- **Type:** `array of string`
- **Required:** No
- **Description:** --plugin values

### `protoDir`

- **Type:** `string`
- **Required:** No
- **Description:** Directory containing .proto files (optional)

### `protoPath`

- **Type:** `array of string`
- **Required:** No
- **Description:** Additional --proto_path values

