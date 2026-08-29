# container-build-simple Configuration

Assemble a container image from prebuilt files and a base, with no daemon, no Containerfile and no shellout

> Full OpenAPI specification: [spec.openapi.yaml](../spec.openapi.yaml)

## Fields

### `base`

- **Type:** `string`
- **Required:** No
- **Description:** The base image. Use "scratch" for none.
A static binary runs on scratch, so the base exists for whatever else runs alongside it. For a CI job container that is a shell and glibc, which is why the default is a debian and not an alpine: musl breaks JavaScript actions in edge cases, and every "uses:" step in a workflow is a JavaScript action.


### `binDir`

- **Type:** `string`
- **Required:** No
- **Description:** Where the files land inside the image, and the front of PATH.

### `env`

- **Type:** `map[string]string`
- **Required:** No
- **Description:** Environment variables to set in the image config.

### `from`

- **Type:** `array of string`
- **Required:** Yes
- **Description:** Globs of the files that go into the layer, relative to the repository. A glob that matches nothing fails the build, because an image that silently ships empty fails on the runner that tries to use it, days later and far from the cause.


### `labels`

- **Type:** `map[string]string`
- **Required:** No
- **Description:** Labels to set in the image config.

### `platforms`

- **Type:** `array of string`
- **Required:** No
- **Description:** The os/arch pairs to assemble, like linux/amd64. Each becomes one manifest in the index. Defaults to linux/amd64.
A file is matched to a platform by its name_os_arch suffix, and lands in the image under its real name.


