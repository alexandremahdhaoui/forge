# rust-license-header

**Add an Apache License 2.0 header to Rust files that don't have one.**

> "Our first license-header script re-appended the header on every run and stomped the original copyright year. rust-license-header only ever touches a file that has no header at all, so re-running it is always safe."

## What problem does rust-license-header solve?

A license header needs to exist on every source file, but a script that
blindly prepends a header breaks the moment it runs twice: it either
duplicates the header or overwrites a copyright year someone already set.
rust-license-header never rewrites a file that already has a header. It only
ever adds one to a file that has none.

## How do I use rust-license-header?

Add a build target to `forge.yaml`:

```yaml
build:
  - name: license-headers
    src: .
    engine: go://rust-license-header
    spec:
      holder: "Your Name"
```

Run it:

```bash
forge build license-headers
```

## What configuration options are available?

| Field | Required | Description |
|-------|----------|--------------|
| `holder` | Yes | Copyright holder name written into new headers |
| `rootDir` | No | Root directory to scan for `*.rs` files. Defaults to the build's `src`/`path`, then the current directory |
| `excludeDirs` | No | Extra directory names to skip, in addition to the built-in defaults |

Built-in excluded directories: `target`, `.git`, `build`, `tmp`, `node_modules`.

## What counts as "already has a header"?

rust-license-header reads a file's leading run of blank lines and `//` line
comments, before any real code. If any of those lines contains `Copyright`,
`SPDX-License-Identifier`, or `Licensed under`, the file is left completely
untouched, including its existing year and holder. A file is only ever
given a new header if none of its leading comments match any of those.

## What year does a new header get?

The year the engine runs in. An existing header's year is never changed,
no matter how old it is. If a file was first licensed in 2024 and the
engine runs again in 2026, that file's header still reads 2024.

## What files are skipped?

- Anything under `target/`, `.git/`, `build/`, `tmp/`, `node_modules/`, or a
  directory named in `excludeDirs`
- Any file whose first non-empty line starts with `// Code generated`, the
  same convention `go-lint-licenses` uses for Go files
