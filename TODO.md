# TODOs

## Refactor all forge/engine APIs using OpenAPI specification and code generation

1. Currently it's kind of a mess
1. Every packages/engines implement their stuff their way and it's not easy to understand what's going on etc...
1. We must generate most of the mcp-server/cli code from an OpenAPI specification
1. All engines (e.g. built-ins or user created ones) must be created from an OpenAPI specification -> All the base logic derives from the spec
1. There will be a CLI+mcp-server named `forge-dev` which is a CLI/mcp-server to make developing forge engines seamless. It should at least have a `bootstrap` and `config validate` and `validate` command.
1. Creating an OpenAPI spec must be easy, forge-dev must have a command to bootstrap engine creation or something. The common OpenAPI spec of the forge engine type (e.g. if we select a forge testenv engine) will not be generated, the generator (which will be a built-in in this repo) (based on the a spec for this engine) will generate the code accordingly. The OpenAPI spec is for the free-form spec field of the generator's configuration
6. The `forge-dev config validate [forge-dev config path]` command must validate that the config for the config for
7. The `forge-dev validate [path to an engine path]` command will validate that the engine is configured and implemented correctly, e.g. if code is not generated, or if configuration is wrong (through `forge-dev config validate`) or if the engine does not use the forge-dev config or is not tested or does not use the common packages/libraries from forge, it should exit with errors detailing what's wrong

## Refactor forge into packages/libraries to provide common libraries for engine creation

1. Libraries must be well documented
1. Process of creating an engine using the libraries must be well documented too
1. The engines (especially the built-ins) must almost only implement their "business logic" -> All commonalities must be abstraced into public libraries/packages
1. It must be extremely simple to implement an engine -> In this repo (i.e. a built-in) or in any other repo, we want user to be able to create engines very easily and create their own
1. Developing engines must be easy. There must be some common packages/libaries or testing framework or something to easily tests the packages

## forge config validate

1. All engines should implement config validate to ensure their spec is properly implemented (the free form spec for each implementation must be validated for each specific engine)
1. Config validate must be recursive, meaning that all mcp server of engines should have a config-validate tool to validate their own spec.

## forge docs get|list

1. This command is not implemented correctly
1. Docs are still weird
1. Many docs are missing
1. There is also docs that are redundant and some are way too large and should be broken down.
1. Docs/prompts are a huge mess and we MUST streamline that.
1. THERE MUST NOT BE PROMPTS anymore -> ALL DOCS -> `forge prompt ...` i.e. the prompt command of forge must be removed -> No backward compatibility, no deprecation, just completely remove it.
1. How does the recursive doc fetching actually works?
1. Forge docs list should provide list of docs from all built-in mcp server engines defined in this repo
1. forge docs list|get for engines should return entries such as `go-build/schema` for built-in docs such as the schema doc of the go-build built-in
