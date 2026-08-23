# go-license-header

**Add an Apache License 2.0 header to Go files that don't have one.**

> "go-lint-licenses told us 40 files were missing a header. It never fixed a single one. go-license-header is the other half: it writes the header go-lint-licenses was only ever going to complain about."

## What problem does go-license-header solve?

`go-lint-licenses` verifies license headers and fails the build when one is
missing, but it never writes anything — by design, a check-only test stage
shouldn't mutate source. Someone still has to add the header by hand, and a
hand-rolled fix script is exactly the kind of thing that quietly re-appends
a header on a second run and stomps the year someone already set.
go-license-header only ever touches a file that has no header at all.

## How do I use go-license-header?

Add a build target to `forge.yaml`:

```yaml
build:
  - name: license-headers
    src: .
    engine: forge://go-license-header
    spec:
      holder: "Your Name"
```

Run it:

```bash
forge build license-headers
```

Pair it with `go-lint-licenses` as a test stage so CI still fails on a file
someone adds without running the fixer first.

## What configuration options are available?

| Field | Required | Description |
|-------|----------|--------------|
| `holder` | Yes | Copyright holder name written into new headers |
| `rootDir` | No | Root directory to scan for `*.go` files. Defaults to the build's `src`/`path`, then the current directory |
| `excludeDirs` | No | Extra directory names to skip, in addition to the built-in defaults |

Built-in excluded directories: `target`, `vendor`, `.git`, `build`, `tmp`,
`.tmp`, `node_modules`.

## What counts as "already has a header"?

Same rule `go-lint-licenses` uses: the leading run of blank lines and `//`
line comments, before any real code. If any of those lines contains
`Copyright`, `SPDX-License-Identifier`, or `Licensed under`, the file is
left completely untouched, including its existing year and holder.

## What year does a new header get?

The year the engine runs in. An existing header's year is never changed.

## Shared implementation

go-license-header and `rust-license-header` are thin, language-specific
wrappers around `internal/licenseheader`, the same package `go-lint-licenses`
uses for its own detection. One implementation, tested once, backs the
checker and both fixers.
