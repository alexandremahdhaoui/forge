# Forge Architecture

**How forge orchestrates builds and tests through MCP servers**

> "I needed a build system that AI agents could understand and control directly. Forge's MCP-first design means Claude Code can build my project, manage test environments, and run tests without any wrapper scripts." - Forge User

## What is forge's architecture?

Forge is built entirely on the Model Context Protocol (MCP). Every component - the CLI, build engines, test runners, and test environment managers - is an MCP server communicating via stdio-based JSON-RPC 2.0.

```
                    +-----------------+
                    |   forge CLI     |   Orchestrator (MCP client + server)
                    |   forge.yaml    |   Reads configuration, coordinates engines
                    +--------+--------+
                             |
                      MCP over stdio
                             |
        +--------------------+--------------------+
        |                    |                    |
+-------v-------+    +-------v-------+    +-------v-------+
|   go-build    |    |   testenv     |    |   go-test     |   Build/Test Engines
|   (server)    |    |   (server)    |    |   (server)    |   (MCP servers)
+---------------+    +-------+-------+    +---------------+
                             |
                      +------+------+
                      |             |
               +------v------+ +----v--------+
               |testenv-kind | |testenv-lcr  |   Subengines
               |  (server)   | |  (server)   |   (MCP servers)
               +-------------+ +-------------+
```

## How do components communicate?

All communication uses MCP over stdio:

1. **forge CLI** starts an engine binary with `--mcp` flag
2. **Engine** becomes an MCP server, listening on stdin/stdout
3. **forge** sends JSON-RPC 2.0 requests (e.g., `tools/call` for `build`)
4. **Engine** processes request, returns JSON-RPC response
5. **forge** closes connection, engine exits

This uniform protocol means AI agents can invoke any forge component directly.

## Where is state stored?

**Artifact Store** (`.ignore.artifact-store.yaml`):
- Tracks all built artifacts (binaries, containers)
- Records git version, timestamp, dependencies
- Stores TestEnvironment metadata for test stages
- Automatically prunes old artifacts (keeps 3 most recent)

**Test Environments** (in artifact store):
- Created by `forge test <stage> create`
- Stores tmpDir, files, metadata, managed resources
- Cleanup via `forge test <stage> delete <id>`

**Configuration** (`forge.yaml`):
- Defines build specs, test stages, engine aliases
- Single source of truth for project orchestration

## Detailed Documentation

- **[ARCHITECTURE.md](../../ARCHITECTURE.md)** - Full system architecture, all components, design patterns
- **[testenv-architecture.md](./testenv-architecture.md)** - Test environment system design and data flows
