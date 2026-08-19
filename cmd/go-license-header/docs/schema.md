# go-license-header Configuration

Adds an Apache License 2.0 header to Go source files that don't already have a license header. Never rewrites a file that already has one, so it never touches an existing copyright year. Fixes what go-lint-licenses only checks.


> Full OpenAPI specification: [spec.openapi.yaml](../spec.openapi.yaml)

## Fields

### `excludeDirs`

- **Type:** `array of string`
- **Required:** No
- **Description:** Additional directory names to skip, beyond the built-in defaults (target, vendor, .git, build, tmp, .tmp, node_modules)


### `holder`

- **Type:** `string`
- **Required:** Yes
- **Description:** Copyright holder name to insert into new headers

### `rootDir`

- **Type:** `string`
- **Required:** No
- **Description:** Root directory to scan for *.go files (defaults to src or current directory)

