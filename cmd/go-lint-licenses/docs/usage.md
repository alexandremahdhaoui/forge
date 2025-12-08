# Go Lint Licenses Usage Guide

## Purpose

`go-lint-licenses` is a forge engine for verifying that all Go source files have proper license headers. It scans the repository for Go files and ensures each has a copyright or license identifier. This helps maintain license compliance across the codebase.

## Invocation

### MCP Mode

Run as an MCP server:

```bash
go-lint-licenses --mcp
```

Forge invokes this automatically when using:

```yaml
runner: go://go-lint-licenses
```

## Available MCP Tools

### `run`

Verify all Go files have license headers.

**Input Schema:**
```json
{
  "stage": "string (required)",
  "name": "string (optional)",
  "rootDir": "string (optional)"
}
```

**Output:**
```json
{
  "id": "string",
  "stage": "string",
  "status": "passed|failed",
  "startTime": "2025-01-06T10:00:00Z",
  "duration": 0.123,
  "testStats": {
    "total": 150,
    "passed": 150,
    "failed": 0,
    "skipped": 0
  },
  "errorMessage": "string"
}
```

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "run",
    "arguments": {
      "stage": "verify-license",
      "rootDir": "."
    }
  }
}
```

### `docs-list`

List all available documentation for go-lint-licenses.

### `docs-get`

Get a specific documentation by name.

**Input Schema:**
```json
{
  "name": "string (required)"
}
```

### `docs-validate`

Validate documentation completeness.

## Common Use Cases

### License Compliance Check

Run as part of CI pipeline to ensure all files have license headers:

```yaml
test:
  - name: verify-license
    runner: go://go-lint-licenses
```

### Pre-commit Hook

Verify license headers before commit:

```bash
forge test run verify-license
```

## Validation Rules

### Valid License Patterns

The following patterns are accepted (must appear in first 15 lines):
- `// Copyright ...`
- `// SPDX-License-Identifier: ...`
- `// Licensed under ...`

### Files Checked

- All `*.go` files
- Recursively scans rootDir
- Skips `vendor`, `.git`, `.tmp`, and `node_modules` directories
- Skips generated files (files starting with `// Code generated`)

### Pass Criteria

- All Go files have one of the valid license patterns
- Pattern must appear before the `package` declaration

### Fail Criteria

- Any Go file missing a license header
- Error message lists all files without headers

## Error Message Format

On failure:
```
Found 3 file(s) without license headers out of 150 total files

Files missing license headers:
  - pkg/myapp/handler.go
  - pkg/utils/helper.go
  - cmd/server/main.go

Go files must have one of these license header patterns:
  // Copyright ...
  // SPDX-License-Identifier: ...
  // Licensed under ...
```

## Adding License Headers

To fix files without license headers, add the appropriate header at the top:

### Copyright Header

```go
// Copyright 2024 Your Name or Company
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package myapp
```

### SPDX Header

```go
// SPDX-License-Identifier: Apache-2.0

package myapp
```

## Implementation Details

- Walks directory tree recursively
- Parses Go files to check for license patterns
- Skips generated files automatically
- Returns detailed error with file list on failure
- Fast execution (no compilation)
- No coverage tracking

## See Also

- [Go Lint Licenses Configuration Schema](schema.md)
- [go-lint-tags MCP Server](../../go-lint-tags/docs/usage.md)
- [go-lint MCP Server](../../go-lint/docs/usage.md)
