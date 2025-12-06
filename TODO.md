# TODOs

## Parallel build and parallel tests

The idea is to allow forge to run all kinds of tests in parallel to speed up the process when running `forge test-all`.
That being said we may need to still run test or build targets before some other.
That is, we need a way to require some target before executing some others.

ACTUAL SOLUTION:

The implementation should be a go://parallel-test-runner and go://parallel-builder that can in their spec accept respectively
test-runners and builders, these should be specified in a list and will be ran by these engines in parallel.
No need for "require" "needs" and complexity added. We just have parallel runners/builders that will run multiple targets in parallel.

BUT how about the artifact store etc... How does the output data returned by each runners/builders to their respective parallel engines
will be properly returned and how to ensure it makes sense? We also want that some stuff such as lazy-build with dependency detectors
still runs correctly.

## Coverage issue

Some tests does not implement coverage. Their coverage if not enabled should not be used to calculate overall coverage! If not enabled coverage should not be shown as 0% like it is currently

## DOCUMENTATION IDEA

Refactor the documentation:

- All engines (testenv-subengine, testenv, test-runners, builders, etc...) implements a "docs" subcommand/mcp-tool which can have "get" and "list" subcommands (like forge basically)
- These commands can be used to get documentation located under ./docs (like for forge)
- Except that for all engines, we search into their directory, so for container-build we would look into [FORGE URL]/cmd/container-build/docs/list.yaml to find the list of docs and then the same thing for finding the doc itself.

This would be a more "scalable" way of writing and maintaining docs. We could even have linters that ensures the docs list.yaml is up to date nd that the docs are actually kept up to date. The linter could also ensure that a minimum amount of docs are created such as the basic usage document to ensure that users or AI Coding Agent can configure the engine/forge. Another mandatory one could be "schema.md" that explains the expected schema for this engine.
