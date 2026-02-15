# forge MCP Server

MCP server for build orchestration and test environment management.

## Purpose

The forge CLI itself runs as an MCP server, providing AI agents direct access to build orchestration capabilities. When invoked with `--mcp`, forge exposes tools to build artifacts from forge.yaml configuration.

## Invocation

```bash
forge --mcp
```

Or configure in your AI agent's MCP settings:
```json
{
  "mcpServers": {
    "forge": {
      "command": "forge",
      "args": ["--mcp"]
    }
  }
}
```

## Available Tools

### `build`

Build artifacts defined in forge.yaml configuration. Returns lightweight summaries. Use `build-get` for full artifact details including dependencies.

**Input Schema:**
```json
{
  "name": "string (optional)",           // Specific artifact name to build
  "artifactName": "string (optional)"    // Alternative to "name"
}
```

**Behavior:**
- If `name` or `artifactName` is provided: builds only that specific artifact
- If neither is provided: builds all artifacts defined in forge.yaml
- Reads forge.yaml from current directory
- Updates artifact store with build results
- Invokes appropriate build engines via MCP

**Output:**

Returns a `BuildResult` with lightweight `ArtifactSummary` objects:

```json
{
  "content": [{
    "type": "text",
    "text": "Successfully built 1 artifact(s)"
  }],
  "artifact": {
    "artifacts": [
      {
        "name": "myapp",
        "type": "binary",
        "location": "./build/bin/myapp",
        "timestamp": "2025-01-15T10:30:00Z"
      }
    ],
    "summary": "Successfully built 1 artifact(s)"
  }
}
```

**ArtifactSummary Schema:**
- `name` (string): Artifact name
- `type` (string): Artifact type (e.g., "binary", "container", "generated", "formatted")
- `location` (string): File path or URL to the artifact
- `timestamp` (string): Build timestamp (RFC3339 format)

To get full details (version, dependencies), use `build-get` with the artifact name.

**Example (build all):**
```json
{
  "method": "tools/call",
  "params": {
    "name": "build",
    "arguments": {}
  }
}
```

**Example (build specific artifact):**
```json
{
  "method": "tools/call",
  "params": {
    "name": "build",
    "arguments": {
      "name": "myapp"
    }
  }
}
```

---

### `build-get`

Get full details of a built artifact by name, including dependencies and version info.

**Input Schema:**
```json
{
  "name": "string (required)"  // Artifact name
}
```

**Output:**

Returns the full `Artifact` object for the most recent build of the given name:

```json
{
  "content": [{
    "type": "text",
    "text": "Successfully retrieved artifact: myapp"
  }],
  "artifact": {
    "name": "myapp",
    "type": "binary",
    "location": "./build/bin/myapp",
    "timestamp": "2025-01-15T10:30:00Z",
    "version": "abc123def",
    "dependencies": [
      {
        "type": "file",
        "filePath": "/path/to/go.mod",
        "timestamp": "2025-01-15T09:00:00Z"
      },
      {
        "type": "externalPackage",
        "externalPackage": "sigs.k8s.io/yaml",
        "semver": "v1.6.0"
      }
    ],
    "dependencyDetectorEngine": "go://go-dependency-detector"
  }
}
```

**Artifact Schema:**
- `name` (string): Artifact name
- `type` (string): Artifact type
- `location` (string): File path or URL to the artifact
- `timestamp` (string): Build timestamp (RFC3339)
- `version` (string): Git commit hash or version identifier
- `dependencies` (array): Tracked dependencies (file paths with timestamps, external packages with semver)
- `dependencyDetectorEngine` (string): Engine URI used for dependency detection
- `dependencyDetectorSpec` (object): Engine-specific configuration

---

### `test-create`

Create a test environment for a specific test stage. Returns the full test environment details.

**Input Schema:**
```json
{
  "stage": "string (required)"  // Test stage name (e.g., "integration", "e2e")
}
```

**Output:**

Returns a structured `TestEnvironment` object with complete environment details:

```json
{
  "content": [{
    "type": "text",
    "text": "Successfully created test environment for stage: integration"
  }],
  "artifact": {
    "id": "test-uuid-123",
    "name": "integration",
    "status": "created",
    "createdAt": "2025-01-15T10:30:00Z",
    "updatedAt": "2025-01-15T10:30:00Z",
    "tmpDir": "/tmp/forge-test-integration-test-uuid-123",
    "files": {
      "testenv-kind.kubeconfig": "kubeconfig",
      "testenv-lcr.credentials": "credentials.json"
    },
    "managedResources": [
      "/tmp/forge-test-integration-test-uuid-123"
    ],
    "metadata": {
      "testenv-kind.clusterName": "forge-integration-test-uuid-123",
      "testenv-lcr.registryURL": "https://localhost:5000"
    }
  }
}
```

**TestEnvironment Schema:**
- `id` (string): Unique test environment identifier
- `name` (string): Test stage name
- `status` (string): Environment status ("created", "running", "passed", "failed", "partially_deleted")
- `createdAt` (string): Creation timestamp (RFC3339)
- `updatedAt` (string): Last update timestamp (RFC3339)
- `tmpDir` (string): Temporary directory path for this environment
- `files` (object): Map of file keys to relative paths (relative to tmpDir)
- `managedResources` (array): List of files/directories managed by this environment
- `metadata` (object): Engine-specific metadata (namespaced by engine name)

---

### `test-get`

Get full details of a test environment by ID, including files, metadata, managed resources, and env vars.

**Input Schema:**
```json
{
  "stage": "string (required)",   // Test stage name
  "testID": "string (required)",  // Test environment ID
  "format": "string (optional)"   // Output format: "json", "yaml", or "table" (default)
}
```

**Output:**

Returns the same `TestEnvironment` structure as `test-create`.

---

### `test-list`

List test reports for a stage. Returns lightweight summaries. Use `test-run` for full test report details or `test-get` for full test environment details.

**Input Schema:**
```json
{
  "stage": "string (required)",  // Test stage name
  "format": "string (optional)"  // Output format: "json", "yaml", or "table" (default)
}
```

**Output:**

Returns a `TestListResult` with lightweight `TestReportSummary` objects:

```json
{
  "content": [{
    "type": "text",
    "text": "Successfully listed 2 test report(s) for stage: unit"
  }],
  "artifact": {
    "reports": [
      {
        "id": "report-uuid-abc123",
        "stage": "unit",
        "status": "passed",
        "startTime": "2025-01-15T10:30:00Z"
      },
      {
        "id": "report-uuid-def456",
        "stage": "unit",
        "status": "failed",
        "startTime": "2025-01-14T09:00:00Z"
      }
    ],
    "stage": "unit",
    "count": 2
  }
}
```

**TestReportSummary Schema:**
- `id` (string): Unique test report identifier
- `stage` (string): Test stage name
- `status` (string): Test result ("passed" or "failed")
- `startTime` (string): Test start timestamp (RFC3339)

To get the full test report (stats, coverage, errors), use `test-run`. To get the full test environment, use `test-get` with stage and testID.

---

### `test-run`

Run tests for a specific test stage. Returns the full test report with stats, coverage, and error details.

**Input Schema:**
```json
{
  "stage": "string (required)",   // Test stage name
  "testID": "string (optional)"   // Existing test environment ID (auto-creates if not provided)
}
```

**Output:**

Returns a structured `TestReport` object with detailed test results:

```json
{
  "content": [{
    "type": "text",
    "text": "Successfully ran tests for stage: unit"
  }],
  "isError": false,
  "artifact": {
    "id": "report-uuid-789",
    "stage": "unit",
    "status": "passed",
    "startTime": "2025-01-15T10:30:00Z",
    "duration": 12.5,
    "testStats": {
      "total": 42,
      "passed": 42,
      "failed": 0,
      "skipped": 0
    },
    "coverage": {
      "percentage": 85.5,
      "filePath": ".forge/tmp/coverage.out"
    },
    "artifactFiles": [
      ".forge/tmp/test-report.xml"
    ],
    "outputPath": ".forge/tmp/test-output.log",
    "errorMessage": "",
    "createdAt": "2025-01-15T10:30:12Z",
    "updatedAt": "2025-01-15T10:30:12Z"
  }
}
```

**TestReport Schema:**
- `id` (string): Unique test report identifier
- `stage` (string): Test stage name
- `status` (string): Test result ("passed" or "failed")
- `startTime` (string): Test start timestamp (RFC3339)
- `duration` (number): Test duration in seconds
- `testStats` (object):
  - `total` (number): Total number of tests
  - `passed` (number): Number of passed tests
  - `failed` (number): Number of failed tests
  - `skipped` (number): Number of skipped tests
- `coverage` (object):
  - `percentage` (number): Code coverage percentage (0-100)
  - `filePath` (string): Path to coverage file
- `artifactFiles` (array): List of artifact files generated (e.g., XML reports)
- `outputPath` (string): Path to detailed test output
- `errorMessage` (string): Error message if tests failed
- `createdAt` (string): Report creation timestamp (RFC3339)
- `updatedAt` (string): Last update timestamp (RFC3339)

**Note:** If tests fail, `isError` is set to `true` but the artifact still contains the full `TestReport`.

---

### `test-delete`

Delete a test environment.

**Input Schema:**
```json
{
  "stage": "string (required)",  // Test stage name
  "testID": "string (required)"  // Test environment ID to delete
}
```

**Output:**
```text
Successfully deleted test environment: test-uuid-123
```

---

### `test-all`

Build all artifacts and run all test stages sequentially with fail-fast behavior. Returns lightweight summaries. Use `build-get` for artifact details and `test-get` with stage/testID for full test environment details.

**Fail-Fast Behavior:**
- Execution stops immediately on the first test stage failure
- Partial results are returned with `stoppedEarly: true`
- Only completed test stages appear in `testReports`

**Input Schema:**
```json
{}  // No parameters required
```

**Output:**

Returns a `TestAllResult` with lightweight summaries for both artifacts and test reports:

**Success Case (all tests pass):**
```json
{
  "content": [{
    "type": "text",
    "text": "Successfully completed test-all: 3 artifact(s) built, 4 test stage(s) run, 4 passed, 0 failed"
  }],
  "artifact": {
    "buildArtifacts": [
      {
        "name": "myapp",
        "type": "binary",
        "location": "./build/bin/myapp",
        "timestamp": "2025-01-15T10:30:00Z"
      }
    ],
    "testReports": [
      {
        "id": "report-uuid-1",
        "stage": "lint",
        "status": "passed",
        "startTime": "2025-01-15T10:31:00Z"
      },
      {
        "id": "report-uuid-2",
        "stage": "unit",
        "status": "passed",
        "startTime": "2025-01-15T10:32:00Z"
      }
    ],
    "stoppedEarly": false,
    "summary": "3 artifact(s) built, 4 test stage(s) run, 4 passed, 0 failed"
  }
}
```

**Failure Case (stopped early):**
```json
{
  "content": [{
    "type": "text",
    "text": "Test-all completed with failures: 3 artifact(s) built, 2 of 4 test stage(s) run (stopped early due to failure), 1 passed, 1 failed"
  }],
  "artifact": {
    "buildArtifacts": [
      {
        "name": "myapp",
        "type": "binary",
        "location": "./build/bin/myapp",
        "timestamp": "2025-01-15T10:30:00Z"
      }
    ],
    "testReports": [
      {
        "id": "report-uuid-1",
        "stage": "lint",
        "status": "passed",
        "startTime": "2025-01-15T10:31:00Z"
      },
      {
        "id": "report-uuid-2",
        "stage": "unit",
        "status": "failed",
        "startTime": "2025-01-15T10:32:00Z"
      }
    ],
    "stoppedEarly": true,
    "summary": "3 artifact(s) built, 2 of 4 test stage(s) run (stopped early due to failure), 1 passed, 1 failed"
  }
}
```

**TestAllResult Schema:**
- `buildArtifacts` (array): Array of `ArtifactSummary` objects (name, type, location, timestamp). Use `build-get` for full details.
- `testReports` (array): Array of `TestReportSummary` objects (id, stage, status, startTime) — contains only completed stages. Use `test-run` for full reports.
- `stoppedEarly` (boolean): `true` if execution stopped due to a failure, `false` if all stages completed
- `summary` (string): Human-readable summary of results

**Note:** With fail-fast behavior, execution stops on the first test stage failure. Check `stoppedEarly` to determine if partial results were returned.

---

### `config-validate`

Validate forge.yaml configuration file.

**Input Schema:**
```json
{
  "configPath": "string (optional)"  // Path to config file (defaults to "forge.yaml")
}
```

**Output:**
```text
Configuration is valid
```

Or on error:
```text
Configuration validation failed: <error details>
```

---

### `docs-list`

List available documentation. Supports three modes: list engines, list docs for an engine, or list all docs.

**Input Schema:**
```json
{
  "engine": "string (optional)"  // Engine name, "all" for all docs, empty for engines list
}
```

**Modes:**
- `engine=""` (empty or omitted): Returns list of engines with documentation
- `engine="all"`: Returns all docs from all engines
- `engine="<name>"`: Returns docs for a specific engine (e.g., "forge", "go-build")

**Output (engines list mode):**
```json
{
  "content": [{
    "type": "text",
    "text": "Found 4 engine(s) with documentation"
  }],
  "artifact": {
    "engines": [
      {"name": "forge", "docCount": 5},
      {"name": "go-build", "docCount": 2},
      {"name": "testenv", "docCount": 3}
    ]
  }
}
```

**Output (docs for engine mode):**
```json
{
  "content": [{
    "type": "text",
    "text": "Found 2 doc(s) for engine 'go-build'"
  }],
  "artifact": {
    "engine": "go-build",
    "docs": [
      {
        "engine": "go-build",
        "name": "usage",
        "title": "Go Build Usage Guide",
        "description": "How to use go-build",
        "path": "cmd/go-build/docs/usage.md",
        "tags": ["usage", "guide"]
      },
      {
        "engine": "go-build",
        "name": "schema",
        "title": "Configuration Schema",
        "description": "Configuration options for go-build",
        "path": "cmd/go-build/docs/schema.md"
      }
    ]
  }
}
```

**Output (all docs mode):**
```json
{
  "content": [{
    "type": "text",
    "text": "Found 10 doc(s) across all engines"
  }],
  "artifact": {
    "docs": [
      {
        "engine": "forge",
        "name": "architecture",
        "title": "Forge Architecture",
        "description": "System architecture overview",
        "path": "docs/architecture.md"
      },
      {
        "engine": "go-build",
        "name": "usage",
        "title": "Go Build Usage Guide",
        "description": "How to use go-build",
        "path": "cmd/go-build/docs/usage.md"
      }
    ]
  }
}
```

**Example (list engines):**
```json
{
  "method": "tools/call",
  "params": {
    "name": "docs-list",
    "arguments": {}
  }
}
```

**Example (list docs for engine):**
```json
{
  "method": "tools/call",
  "params": {
    "name": "docs-list",
    "arguments": {
      "engine": "go-build"
    }
  }
}
```

**Example (list all docs):**
```json
{
  "method": "tools/call",
  "params": {
    "name": "docs-list",
    "arguments": {
      "engine": "all"
    }
  }
}
```

---

### `docs-get`

Retrieve the content of a specific documentation file.

**Input Schema:**
```json
{
  "name": "string (required)"  // Document name, optionally prefixed with engine (e.g., "go-build/usage")
}
```

**Output:**
```json
{
  "content": [{
    "type": "text",
    "text": "# Go Build Usage Guide\n\n..."
  }]
}
```

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "docs-get",
    "arguments": {
      "name": "go-build/usage"
    }
  }
}
```

---

### `list`

List available build targets and test stages defined in forge.yaml.

**Input Schema:**
```json
{
  "category": "string (optional)"  // "build" or "test" - if omitted, lists both
}
```

**Output:**
```json
{
  "content": [{
    "type": "text",
    "text": "Build targets:\n- myapp\n- mylib\n\nTest stages:\n- unit\n- integration"
  }],
  "artifact": {
    "build": ["myapp", "mylib"],
    "test": ["unit", "integration"]
  }
}
```

---

### `prompt-list`

List all available documentation prompts.

**Input Schema:**
```json
{}  // No parameters
```

---

### `prompt-get`

Get a specific documentation prompt by name.

**Input Schema:**
```json
{
  "name": "string (required)"  // Prompt name
}
```

---

## How It Works

1. Loads forge.yaml configuration from current directory
2. Reads existing artifact store
3. Filters build specs by artifact name (if provided)
4. Groups specs by build engine
5. Invokes each engine via MCP:
   - Single spec: calls engine's `build` tool
   - Multiple specs: calls engine's `buildBatch` tool
6. Updates artifact store with build results
7. Returns summary of build operations

## Integration with Forge

The forge MCP server orchestrates other MCP build engines:

```yaml
# forge.yaml
name: my-project
artifactStorePath: .forge/artifacts.yaml

build:
  - name: myapp
    src: ./cmd/myapp
    engine: go://go-build      # Invokes go-build MCP server

  - name: myimage
    src: ./Containerfile
    engine: go://container-build  # Invokes container-build MCP server
```

When you call the forge `build` tool, it:
1. Parses the engine URIs (e.g., `go://go-build`)
2. Launches the corresponding MCP server binary
3. Calls the appropriate tool on that server
4. Aggregates results

## CLI Usage

The forge CLI also supports traditional command-line usage:

```bash
# Build all artifacts
forge build

# Build specific artifact
forge build myapp

# Test operations (new command structure)
forge test run unit                 # Run tests
forge test list unit                # List test reports
forge test get unit <TEST_ID>       # Get test report details
forge test delete unit <TEST_ID>    # Delete test report

# Test environment management
forge test list-env integration     # List test environments
forge test get-env integration <ENV_ID>    # Get environment details
forge test create-env integration   # Create test environment
forge test delete-env integration <ENV_ID> # Delete test environment
```

See [forge-usage.md](../../docs/forge-usage.md) for complete CLI documentation.

## Architecture

The forge MCP server acts as an orchestrator, coordinating multiple specialized MCP servers:

```
┌─────────────┐
│   AI Agent  │
│   or User   │
└──────┬──────┘
       │ MCP
┌──────▼──────┐
│    forge    │ MCP Server (orchestrator)
│  --mcp mode │
└──────┬──────┘
       │ Spawns and coordinates
       ├──────────────┬─────────────┐
       │              │             │
┌──────▼──────┐ ┌────▼────┐  ┌─────▼─────┐
│  go-build   │ │ testenv │  │test-runner│
│ MCP Server  │ │   MCP   │  │    MCP    │
└─────────────┘ └─────────┘  └───────────┘
```

## See Also

- [go-build MCP Server](../go-build/MCP.md)
- [container-build MCP Server](../container-build/MCP.md)
- [testenv MCP Server](../testenv/MCP.md)
- [Forge CLI Documentation](../../docs/forge-usage.md)
- [Forge Design Document](../../DESIGN.md)
