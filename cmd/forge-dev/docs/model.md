# The forge-dev model

forge-dev generates programs from typed sources along two axes. Kind says
what the program is. Language says what emits it. A generator fills one
kind and language cell. The defaults ship here; a custom generator is any
forge engine, so a new cell never touches forge-dev.

## What core owns and generators never do

forge-dev core parses `forge-dev.yaml` and the OpenAPI spec, validates
them, computes the source checksum, skips fresh outputs, emits
`zz_generated.runnable.yaml` and the shared docs, and writes files. A
generator only answers file contents. This split is the contract: a
generator cannot weaken freshness, escape the engine directory, or
redefine the runnable manifest.

## Kind, axis one

| Kind | Surface | The generated skeleton |
|---|---|---|
| mcp-server | `surface.tools` | stdio JSON-RPC server, config-validate derived from the Spec schema |
| rest-api | the OpenAPI paths | HTTP server. Defined; no builtin cell yet, needs a `generator:` |
| cli | `surface.commands` | argv dispatcher, exit codes pass through, unknown command fails loud |
| binary | none | nothing; the runnable manifest and the docs are the output |

A custom kind is any other name plus whatever surface its generator
defines. Core passes the surface through opaquely and the generator
validates it. Every kind's runnable manifest has the same `inputs:`
shape, so forge-factory checks inputs without knowing kinds.

### mcp-server profiles

The old engine types survive as profiles: preset tool surfaces plus the
framework wiring, data internal to the mcp-server kind.

| Profile | Handler the author writes |
|---|---|
| builder | `Build(ctx, BuildInput, *Spec) (*Artifact, error)` |
| test-runner | `Run(ctx, RunInput, *Spec) (*TestReport, error)` |
| testenv-subengine | `Create` and `Delete` |
| dependency-detector | the detect tool registration |
| none, the default | one handler per `surface.tools` entry |

## Language, axis two

Builtin cells today: mcp-server in go, rust, python and typescript
(profiles are go only); cli in go; binary in every language. Every other
cell is external.

## Generator, the extension door

```yaml
name: my-gui
kind: gui
language: rust
generator: forge://github.com/x/gui-gen/engines/gui-gen
surface:
  windows:
    - name: main-window
```

A generator is an MCP engine with one tool, `generate`. It receives the
normalized model and answers files:

```json
// input
{"name","version","description","kind","language","packageName",
 "surface","runtime","openapiSpec","checksum","srcDir"}
// output
{"files":[{"path":"zz_generated_gui.rs","content":"..."}]}
```

Paths are relative to the engine directory and never escape it. The
generator resolves like every engine: `forge://` through the factory and
register, so a generator is versioned, proven and cached like anything
else it generates for.

## Migration from type:

`type: X` fails validation with one line naming the fix:

- `type: builder` and the other three: `kind: mcp-server` plus
  `profile: X`.
- `type: generic`: `kind: mcp-server` and move `generate.tools` to
  `surface.tools`.
