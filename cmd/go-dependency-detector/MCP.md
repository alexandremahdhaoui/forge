# go-dependency-detector MCP Server

MCP server for detecting Go code dependencies to enable lazy rebuild optimization.

## Purpose

Provides MCP tools for detecting all dependencies (local files and external packages) for a given Go function. This enables intelligent caching and lazy rebuild decisions by tracking what a function actually depends on.

## Invocation

```bash
go-dependency-detector --mcp
```

Forge invokes this automatically via:
```yaml
engine: go://go-dependency-detector
```

## Available Tools

### `detectDependencies`

Detect all dependencies for a specific Go function.

**Input Schema:**
```json
{
  "filePath": "string (required)",        // Path to Go source file
  "funcName": "string (required)",        // Name of function to analyze (e.g., "main")
  "spec": {}                              // Engine-specific configuration (optional)
}
```

**Output:**
```json
{
  "dependencies": [
    {
      "type": "file",                     // Dependency type: "file" or "externalPackage"
      "filePath": "string",               // Absolute path to file (if type=file)
      "timestamp": "string",              // RFC3339 timestamp (if type=file)
      "externalPackage": "string",        // Package identifier (if type=externalPackage)
      "semver": "string"                  // Semantic version (if type=externalPackage)
    }
  ]
}
```

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "detectDependencies",
    "arguments": {
      "filePath": "./cmd/myapp/main.go",
      "funcName": "main"
    }
  }
}
```

## Integration with Forge

### Basic Usage

In `forge.yaml`:
```yaml
build:
  - name: myapp
    src: ./cmd/myapp
    dest: ./build/bin
    engine: go://go-build
    spec:
      dependencyDetector: go://go-dependency-detector
      entrypoint:
        file: ./cmd/myapp/main.go
        function: main
```

## Implementation Details

- Parses Go AST to find function dependencies
- Recursively follows local package imports (transitive dependencies)
- Extracts versions for external packages from go.mod
- Handles replace directives in go.mod
- Prevents infinite loops on circular dependencies
- Returns absolute file paths with timestamps
- Skips standard library imports

## Dependency Types

**File Dependencies (`type: "file"`):**
- Local Go source files imported by the function
- Includes transitive dependencies (A imports B, B imports C)
- Contains absolute file path and modification timestamp

**External Package Dependencies (`type: "externalPackage"`):**
- Third-party packages from go.mod
- Contains package identifier and semantic version
- Supports pseudo-versions (e.g., v0.0.0-20231010123456-abcdef123456)

## See Also

- [go-build MCP Server](../go-build/MCP.md)
- [Forge Build Documentation](../../docs/forge-usage.md#building-artifacts)
