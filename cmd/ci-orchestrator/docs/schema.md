# ci-orchestrator Configuration Schema

**Retired. Superseded by
[forge-ci-spec](https://github.com/alexandremahdhaoui/forge-ci-spec).**

This document described a planned configuration that was never defined. It has
been replaced by a real schema with conformance vectors.

`forge-ci-spec/spec/forge-ci.v1.yaml` is JSON Schema draft 2020-12. The vectors
in `testdata/cases.json` are what an implementation is tested against.

## What changed from the sketch

| Sketched here | In forge-ci |
|---|---|
| `pipelines:` holding named pipelines | `stages:`, one pipeline per file |
| `dependsOn` between stages | Nothing. Stages run in list order. |
| `steps:` with a raw `command:` | `targets:`, naming a forge target or a forge-ci verb |
| `approval:` as a field | `gates:`, a port with its own engines |
| `environment:` as a field | `params:`, using your own key names |

The last row is the important one. A CI that owns a vocabulary will collide
with yours. forge-ci never learns the words `region`, `cell` or `env`, so it
cannot.
