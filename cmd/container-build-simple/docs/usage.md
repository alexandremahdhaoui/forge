# container-build-simple

**Assemble a container image from prebuilt files and a base. No daemon, no
Containerfile, no shellout.**

## What problem does container-build-simple solve?

Some images are not builds. A toolchain image is a base plus a directory of
static binaries:

```dockerfile
FROM debian:stable-slim
COPY forge forge-ci forge-factory ... /usr/local/bin/
```

There is no `RUN` step anywhere in it. Everything a Containerfile builder
carries exists to support `RUN`: a daemon or a rootless emulation of one, a
build context upload, layer caching, and a separate mechanism for
cross-architecture builds. An image with no `RUN` needs none of it.

This engine appends one layer onto a base and writes an OCI image layout to
disk. Multi-architecture costs one layer set per architecture plus an index,
so it is free rather than a feature.

An image that genuinely needs `RUN` steps is a different job. Use
`forge://container-build` for that, and read its README first.

## How do I use container-build-simple?

```yaml
build:
  - name: toolchain
    src: .
    dest: ./build/images
    engine: forge://container-build-simple
    spec:
      base: debian:stable-slim
      binDir: /usr/local/bin
      platforms: [linux/amd64, linux/arm64]
      from:
        - "../forge/build/dist/*_linux_amd64"
        - "../forge/build/dist/*_linux_arm64"
      labels:
        org.opencontainers.image.source: https://github.com/owner/repo
```

```sh
forge build toolchain
```

## Where do the files go inside the image?

Under `binDir`, which also goes to the front of `PATH`. A file carrying the
`name_os_arch` travel suffix lands under its real name, so
`forge-ci_linux_arm64` becomes `/usr/local/bin/forge-ci` and a script inside
the image needs to know nothing about which machine assembled it.

A file carrying a suffix goes only to that platform. A file carrying none
goes to every platform, because a script or a certificate is the same on all
of them.

## Why does it push nothing?

Because a build writes a file and a release publishes it, exactly as a binary
does. This engine holds no credential and cannot reach a registry to write,
which means a build cannot publish by accident.

It records `forge.Artifact{Type: "container"}` with the layout path, and the
release side filters on that type.

## Why debian and not alpine?

The binaries are `CGO_ENABLED=0`, so they are static and would run on
`scratch`. The base exists for whatever else runs alongside them.

For a CI job container that is a shell and glibc. Alpine's only advantage is
size, which is worth nothing against a payload of hundreds of megabytes of
binaries, and it carries two real costs: musl breaks JavaScript actions in
edge cases, and every `uses:` step in a workflow is a JavaScript action; and
alpine job containers are x64 only, which silently drops the arm64 half of a
multi-architecture image.

Set `base: scratch` when nothing but the binaries has to run.

## What fails the build?

- A `from` glob that matches nothing. One glob quietly matching nothing while
  the others matched is exactly how an image ships missing a tool.
- A declared platform that got no files.
- Two files that would land under one name in the image. Both sources are
  named, because that is a declaration mistake rather than a machine failure.
- A platform string that is not `os/arch`.

## Is the output reproducible?

Every file in the layer carries a fixed modification time and the paths are
sorted, so the same inputs assemble to the same digest. A layer whose digest
moved on every build would republish an image nothing changed in.
