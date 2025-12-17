# TODOs

## Open TODOs

- [ ] [0005] Fix documentation in this repo: documentation in this repo is a bit strange

## Done

- [x] [0006] The forge docs list command is way too difficult to read and pick information from
    1. `forge docs list` will list the overall "categories" of documentation, i.e.: `forge`, `forge-dev`, `go-build`,... Basically all engine categories
    1. Then `forge docs list forge` will be used to list all docs in the `forge` category
    1. `forge docs list all` will be used to list all docs of all categories
    1. `forge docs list go-build` would list all go-build docs
    1. The `forge docs` commands have options to return the list in human readable table format, JSON or YAML (both for `forge docs list` and `forge docs list [category]`) -- if and only if it's not already implemented
- [x] [0007] Update all usage docs of all engines in this repo
- [x] [0004] forge-dev and OpenAPI based framework is not complete
    1. The forge-dev and framework uses packages that are inside the ./internal folder, that will not work easily
    1. The main.go could also be generated (zz_generated.main.go) or something -> most of the code can be generated it's very simple
    1. The forge-dev and framework could also ensure that mandatory tools like `validate` (or config validate I forgot) and `docs` are already implemented (generated).
        1. E.g.: for docs user will just have to add docs in the `<path>/docs` and an entry for this doc in `<path>/docs/list.yaml`; actually the list.yaml could also be generated
        1. The mandatory doc `schema.md` could be generated from the open api spec and with some info from the forge-dev.yaml; the generated `schema.md` should also reference the link to the openapi spec for users to check the spec directly.
        1. The mandatory doc `usage.md` cannot be generated, but must be present.
        1. NB: validate might already be implemented but not docs I think
    1. NB: about common libraries/packages/framework:
        1. Libraries must be well documented
        1. Process of creating an engine using the libraries must be well documented too
        1. The engines (especially the built-ins) must almost only implement their "business logic" -> All commonalities must be abstraced into generated code and public libraries/packages if applicable
        1. It must be extremely simple to implement an engine -> In this repo (i.e. a built-in) or in any other repo, we want user to be able to create engines very easily and create their own
        1. Developing engines must be easy. There must be some common packages/libaries or testing framework or something to easily tests the engine
- [x] forge docs get|list
    1. This command is not implemented correctly
    1. Docs are still weird
    1. Many docs are missing
    1. There is also docs that are redundant and some are way too large and should be broken down.
    1. Docs/prompts are a huge mess and we MUST streamline that.
    1. THERE MUST NOT BE PROMPTS anymore -> ALL DOCS -> `forge prompt ...` i.e. the prompt command of forge must be removed -> No backward compatibility, no deprecation, just completely remove it.
    1. How does the recursive doc fetching actually works?
    1. Forge docs list should provide list of docs from all built-in mcp server engines defined in this repo
    1. forge docs list|get for engines should return entries such as `go-build/schema` for built-in docs such as the schema doc of the go-build built-in
- [x] Refactor all forge/engine APIs using OpenAPI specification and code generation
    1. Currently it's kind of a mess
    1. Every packages/engines implement their stuff their way and it's not easy to understand what's going on etc...
    1. We must generate most of the mcp-server/cli code from an OpenAPI specification
    1. All engines (e.g. built-ins or user created ones) must be created from an OpenAPI specification -> All the base logic derives from the spec
    1. There will be a CLI+mcp-server named `forge-dev` which is a CLI/mcp-server to make developing forge engines seamless. It should at least have a `bootstrap` and `config validate` and `validate` command.
    1. Creating an OpenAPI spec must be easy, forge-dev must have a command to bootstrap engine creation or something. The common OpenAPI spec of the forge engine type (e.g. if we select a forge testenv engine) will not be generated, the generator (which will be a built-in in this repo) (based on the a spec for this engine) will generate the code accordingly. The OpenAPI spec is for the free-form spec field of the generator's configuration
    1. The `forge-dev config validate [forge-dev config path]` command must validate that the config for the config for
    1. The `forge-dev validate [path to an engine path]` command will validate that the engine is configured and implemented correctly, e.g. if code is not generated, or if configuration is wrong (through `forge-dev config validate`) or if the engine does not use the forge-dev config or is not tested or does not use the common packages/libraries from forge, it should exit with errors detailing what's wrong
