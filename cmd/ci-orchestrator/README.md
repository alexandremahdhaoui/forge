# ci-orchestrator

**Retired. Superseded by [forge-ci](https://github.com/alexandremahdhaoui/forge-ci).**

This command was a placeholder. It was never implemented, and its
`docs/schema.md` said so: "PLANNED - NOT YET IMPLEMENTED. The configuration
schema is not yet defined."

Orchestration now lives outside forge. forge answers what the targets are and
how to build them, in one repo, on one machine. That boundary is worth keeping.
Which repos at which commits form a release, what runs where, and what gets
promoted are different questions, and they belong in a different tool.

## What survives

The four tenets written here were right and forge-ci keeps them.

- **Accessible.** If you own one computer you can run your pipeline. No
  subscription and no account.
- **Reproducible.** What runs in CI must run locally. forge-ci ships a local
  compute engine and a local manager, so the first version needs no cloud at
  all.
- **Vendor agnostic.** Moving between CI systems must not require a rewrite.
  forge-ci is a CLI that any CI system calls, never a YAML translator.
- **No defaults, no side effects.** Every engine is named explicitly. Nothing
  is implied.

## What does not survive

The sketch in `docs/schema.md` used `dependsOn` between stages.

forge-ci has no dependency graph. Stages are an ordered list and the order is
the order you wrote them in. forge itself has no inter stage dependency either,
so a graph would have been a concept forge does not have.

## Where to look instead

| Repo | Holds |
|---|---|
| [forge-ci](https://github.com/alexandremahdhaoui/forge-ci) | The tool and its engines |
| [forge-ci-spec](https://github.com/alexandremahdhaoui/forge-ci-spec) | The pipeline schema and its conformance vectors |
