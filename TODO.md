# TODOs

## forge config validate

1. All engines should implement config validate to ensure their spec is properly implemented (the free form spec for each implementation must be validated for each specific engine)
2. Config validate must be recursive, meaning that all mcp server of engines should have a config-validate tool to validate their own spec.

## forge docs get|list

1. This command doesn't sleep to be implemented correctly
2. Docs are still weird
3. Many docs are missing
4. How does the recursive doc fetching actually works?
5. Forge docs list should provide list of docs from all built-in mcp server engines defined in this repo
6. forge docs list|get for engines should return entries such as `go-build/schema` for built-in docs such as the schema doc of the go-build built-in
