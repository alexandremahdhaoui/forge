# go-lint-licenses

**Verify all Go source files have license headers.**

> "Our legal team requires license headers on every file. go-lint-licenses catches missing headers before code review, saving time and ensuring compliance."

## What problem does go-lint-licenses solve?

Manually checking license headers is tedious and error-prone. go-lint-licenses scans all Go files and fails if any are missing copyright or SPDX license identifiers, helping maintain license compliance.

## How do I use go-lint-licenses?

```yaml
test:
  - name: verify-license
    runner: go://go-lint-licenses
```

Run with:

```bash
forge test run verify-license
```

## What license patterns are valid?

Files must have one of these patterns in the first 15 lines:

```go
// Copyright ...
// SPDX-License-Identifier: ...
// Licensed under ...
```

## What files are checked?

- All `*.go` files recursively
- Skips: `vendor/`, `.git/`, `.tmp/`, `node_modules/`
- Skips generated files (files starting with `// Code generated`)

## How do I add a license header?

SPDX format (minimal):
```go
// SPDX-License-Identifier: Apache-2.0

package myapp
```

Full copyright format:
```go
// Copyright 2024 Your Name or Company
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package myapp
```

## What output does it produce?

On success:
```json
{
  "stage": "verify-license",
  "status": "passed",
  "testStats": {
    "total": 150,
    "passed": 150,
    "failed": 0
  }
}
```

On failure:
```
Found 3 file(s) without license headers out of 150 total files

Files missing license headers:
  - pkg/myapp/handler.go
  - pkg/utils/helper.go
  - cmd/server/main.go
```

## What's next?

- [schema.md](schema.md) - Configuration reference
- [MCP.md](../../MCP.md) - MCP tool documentation
