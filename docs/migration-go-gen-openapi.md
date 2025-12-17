# Migration Guide: go-gen-openapi Configuration

## Overview

### What Changed

The `go-gen-openapi` tool has been refactored to follow the standard forge build engine pattern. Previously, `go-gen-openapi` was configured through a dedicated `generateOpenAPI` section at the root level of `forge.yaml`. Now, it is configured like all other build tools through the `build` section.

**Old configuration location:**
```yaml
generateOpenAPI:     # Root-level configuration (DEPRECATED)
  defaults: {...}
  specs: [...]
```

**New configuration location:**
```yaml
build:               # Standard build section
  - name: my-api-v1
    engine: go://go-gen-openapi
    spec: {...}
```

### Why This Changed

This refactoring brings several benefits:

- **Consistency**: All build engines (go-build, container-build, go-gen-openapi) now use the same configuration pattern
- **MCP Integration**: Enables direct invocation through the Model Context Protocol, making forge fully AI-native
- **Build Orchestration**: Works seamlessly with `forge build` command alongside other build steps
- **Artifact Tracking**: Properly integrated with forge's artifact store for dependency management

### Breaking Change Notice

This is a **breaking change**. The old `generateOpenAPI` configuration format is no longer supported. If forge detects the old format in your `forge.yaml`, it will return the following error:

```
Error: generateOpenAPI configuration is no longer supported.
Please migrate to build section. See docs/migration-go-gen-openapi.md for migration instructions.
```

You must migrate your configuration to continue using `go-gen-openapi` with the latest version of forge.

---

## Key Differences

### 1. One BuildSpec Per API Version (No Versions Array)

**OLD:** One spec entry with multiple versions in an array
```yaml
generateOpenAPI:
  specs:
    - name: example-api
      versions: [v1, v2, v3]  # Multiple versions in one entry
```

**NEW:** Separate BuildSpec for each API version
```yaml
build:
  - name: example-api-v1    # One BuildSpec for v1
    engine: go://go-gen-openapi
    spec: {...}

  - name: example-api-v2    # One BuildSpec for v2
    engine: go://go-gen-openapi
    spec: {...}

  - name: example-api-v3    # One BuildSpec for v3
    engine: go://go-gen-openapi
    spec: {...}
```

**Why:** This aligns with forge's "one BuildSpec = one artifact" pattern, enabling independent builds and clear artifact tracking per version.

### 2. No Shared Defaults (Explicit Configuration)

**OLD:** Shared defaults applied to all specs
```yaml
generateOpenAPI:
  defaults:
    sourceDir: ./api
    destinationDir: ./pkg/generated
  specs:
    - name: api1      # Inherits defaults
    - name: api2      # Inherits defaults
```

**NEW:** Each BuildSpec is self-contained with explicit configuration
```yaml
build:
  - name: api1-v1
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/api1.v1.yaml
      destinationDir: ./pkg/generated    # Explicit

  - name: api2-v1
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/api2.v1.yaml
      destinationDir: ./pkg/generated    # Explicit
```

**Why:** Each BuildSpec is independent and self-contained, making configurations clearer without hidden inheritance.

**Optional:** You can use YAML anchors to reduce duplication:
```yaml
build:
  - name: api1-v1
    engine: go://go-gen-openapi
    spec: &common-config
      destinationDir: ./pkg/generated
      sourceFile: ./api/api1.v1.yaml
      client:
        enabled: true
        packageName: api1client

  - name: api2-v1
    engine: go://go-gen-openapi
    spec:
      <<: *common-config
      sourceFile: ./api/api2.v1.yaml
      client:
        packageName: api2client
```

### 3. Recommended: Use sourceFile Instead of Templated Paths

**RECOMMENDED:** Explicit source file path
```yaml
spec:
  sourceFile: ./api/example-api.v1.yaml  # Clear and explicit
```

**Also Supported:** Templated source file path (backward compatibility)
```yaml
spec:
  sourceDir: ./api
  name: example-api
  version: v1
  # Results in: ./api/example-api.v1.yaml
```

**Why sourceFile is recommended:**
- More explicit and clear
- No ambiguity about file location
- Easier for new users to understand
- Reduces configuration complexity

**When to use templated pattern:**
- Migrating from old format and want to minimize changes
- Consistent file naming convention across your project
- Programmatic generation of configurations

---

## Old Configuration Format

Here is an example of the old `generateOpenAPI` configuration format:

```yaml
# OLD FORMAT (no longer supported)
name: my-project
artifactStorePath: .ignore.artifact-store.yaml

generateOpenAPI:
  defaults:
    sourceDir: ./api
    destinationDir: ./pkg/generated

  specs:
    # Example 1: Users API with two versions, client only
    - name: users-api
      versions: [v1, v2]
      client:
        enabled: true
        packageName: usersclient

    # Example 2: Products API with one version, both client and server
    - name: products-api
      versions: [v1]
      client:
        enabled: true
        packageName: productsclient
      server:
        enabled: true
        packageName: productsserver
```

---

## New Configuration Format

Here is the equivalent configuration in the new format:

### Example 1: Using sourceFile (RECOMMENDED)

```yaml
# NEW FORMAT (recommended)
name: my-project
artifactStorePath: .ignore.artifact-store.yaml

build:
  # Users API v1 - client only
  - name: users-api-v1
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/users-api.v1.yaml
      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: usersclient

  # Users API v2 - client only
  - name: users-api-v2
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/users-api.v2.yaml
      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: usersclientv2    # Different package name for v2

  # Products API v1 - both client and server
  - name: products-api-v1
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/products-api.v1.yaml
      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: productsclient
      server:
        enabled: true
        packageName: productsserver
```

### Example 2: Using Templated Paths (Backward Compatibility)

```yaml
# NEW FORMAT (templated pattern)
name: my-project
artifactStorePath: .ignore.artifact-store.yaml

build:
  # Users API v1
  - name: users-api-v1
    engine: go://go-gen-openapi
    spec:
      # Templated source file: {sourceDir}/{name}.{version}.yaml
      sourceDir: ./api
      name: users-api
      version: v1
      # Results in: ./api/users-api.v1.yaml

      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: usersclient

  # Users API v2
  - name: users-api-v2
    engine: go://go-gen-openapi
    spec:
      sourceDir: ./api
      name: users-api
      version: v2
      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: usersclientv2

  # Products API v1
  - name: products-api-v1
    engine: go://go-gen-openapi
    spec:
      sourceDir: ./api
      name: products-api
      version: v1
      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: productsclient
      server:
        enabled: true
        packageName: productsserver
```

---

## Step-by-Step Migration Instructions

Follow these steps to migrate your `forge.yaml` configuration:

### Step 1: Back Up Your Current Configuration

Before making changes, save a copy of your current `forge.yaml`:

```bash
cp forge.yaml forge.yaml.backup
```

### Step 2: Identify Your API Specs and Versions

Review your existing `generateOpenAPI` configuration and list out:
- All API specs (each entry in the `specs` array)
- All versions for each spec
- Default values (`sourceDir`, `destinationDir`)

### Step 3: Create BuildSpec Entries

For each combination of spec and version, create a new entry in the `build` section:

1. **Set the name**: Use the pattern `{spec-name}-{version}` (e.g., `users-api-v1`)
2. **Set the engine**: Always `go://go-gen-openapi`
3. **Configure the spec field**:
   - **sourceFile**: (Recommended) Use explicit path: `{defaults.sourceDir}/{spec.name}.{version}.yaml`
   - **OR sourceDir/name/version**: For templated pattern
   - **destinationDir**: Copy from old `defaults.destinationDir` or spec-specific override
   - **client**: Copy client configuration from old spec
   - **server**: Copy server configuration from old spec

### Step 4: Update Package Names for Different Versions

If you have multiple versions of the same API, ensure each version has a unique package name:

```yaml
# v1
client:
  packageName: usersclient

# v2
client:
  packageName: usersclientv2    # Add version suffix
```

### Step 5: Remove Old generateOpenAPI Section

Delete the entire `generateOpenAPI` section from your `forge.yaml`:

```yaml
# DELETE THIS ENTIRE SECTION:
generateOpenAPI:
  defaults:
    sourceDir: ./api
    destinationDir: ./pkg/generated
  specs:
    - name: ...
```

### Step 6: Validate Your Configuration

Run forge to verify the configuration is valid:

```bash
forge build --help
```

If there are syntax errors, you'll see YAML parsing errors.

### Step 7: Test the Migration

Run the build command to test code generation:

```bash
# Build all APIs
forge build

# Or build specific API
forge build users-api-v1
```

Verify that:
- Code is generated in the expected location
- Package names are correct
- No errors occur during generation

### Step 8: Verify Generated Code

Check that the generated code files exist and contain the expected package names:

```bash
# Check generated files
ls -la ./pkg/generated/

# Verify package names
grep "^package" ./pkg/generated/*/zz_generated.oapi-codegen.go
```

---

## Common Scenarios

### Scenario 1: Multiple Versions of the Same API

**OLD:**
```yaml
generateOpenAPI:
  defaults:
    sourceDir: ./api
    destinationDir: ./pkg/generated
  specs:
    - name: users-api
      versions: [v1, v2, v3]
      client:
        enabled: true
        packageName: usersclient
```

**NEW:**
```yaml
build:
  - name: users-api-v1
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/users-api.v1.yaml
      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: usersclient

  - name: users-api-v2
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/users-api.v2.yaml
      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: usersclientv2    # Different package name

  - name: users-api-v3
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/users-api.v3.yaml
      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: usersclientv3    # Different package name
```

**Key Points:**
- Each version becomes a separate BuildSpec
- Package names should be unique per version
- Each BuildSpec explicitly specifies all configuration

### Scenario 2: API with Both Client and Server

**OLD:**
```yaml
generateOpenAPI:
  defaults:
    sourceDir: ./api
    destinationDir: ./pkg/generated
  specs:
    - name: products-api
      versions: [v1]
      client:
        enabled: true
        packageName: productsclient
      server:
        enabled: true
        packageName: productsserver
```

**NEW:**
```yaml
build:
  - name: products-api-v1
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/products-api.v1.yaml
      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: productsclient
      server:
        enabled: true
        packageName: productsserver
```

**Key Points:**
- One BuildSpec can generate both client and server
- No need for separate BuildSpecs for client and server
- Both packages are generated from the same source spec

### Scenario 3: Multiple APIs with Different Configurations

**OLD:**
```yaml
generateOpenAPI:
  defaults:
    sourceDir: ./api
    destinationDir: ./pkg/generated
  specs:
    - name: internal-api
      versions: [v1]
      client:
        enabled: true
        packageName: internalclient
      server:
        enabled: true
        packageName: internalserver

    - name: public-api
      versions: [v1]
      destinationDir: ./pkg/public    # Override default
      client:
        enabled: true
        packageName: publicclient
```

**NEW:**
```yaml
build:
  - name: internal-api-v1
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/internal-api.v1.yaml
      destinationDir: ./pkg/generated
      client:
        enabled: true
        packageName: internalclient
      server:
        enabled: true
        packageName: internalserver

  - name: public-api-v1
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/public-api.v1.yaml
      destinationDir: ./pkg/public    # Different destination
      client:
        enabled: true
        packageName: publicclient
```

**Key Points:**
- Each BuildSpec can have different configuration
- No shared defaults, so destinationDir must be explicit
- Configuration differences are clearer and more explicit

### Scenario 4: Using Defaults for Multiple Similar Configs

If you want to reduce duplication when you have many similar configurations, use YAML anchors:

**NEW (with YAML anchors):**
```yaml
build:
  # Define anchor with common configuration
  - name: api1-v1
    engine: go://go-gen-openapi
    spec: &api-defaults
      destinationDir: ./pkg/generated
      sourceFile: ./api/api1.v1.yaml
      client:
        enabled: true
        packageName: api1client

  # Reuse anchor and override specific fields
  - name: api2-v1
    engine: go://go-gen-openapi
    spec:
      <<: *api-defaults
      sourceFile: ./api/api2.v1.yaml
      client:
        packageName: api2client

  - name: api3-v1
    engine: go://go-gen-openapi
    spec:
      <<: *api-defaults
      sourceFile: ./api/api3.v1.yaml
      client:
        packageName: api3client
```

---

## Troubleshooting

### Error: "generateOpenAPI configuration is no longer supported"

**Full Error:**
```
Error: generateOpenAPI configuration is no longer supported.
Please migrate to build section. See docs/migration-go-gen-openapi.md for migration instructions.
```

**Cause:** Your `forge.yaml` still contains the old `generateOpenAPI` section at the root level.

**Solution:**
1. Follow the migration steps above to convert your configuration
2. Remove the `generateOpenAPI` section from `forge.yaml`
3. Add new BuildSpec entries in the `build` section

### Error: "must provide either 'sourceFile' or all of 'sourceDir', 'name', and 'version'"

**Cause:** The spec field is missing required source file configuration.

**Solution:** Provide EITHER:
- `sourceFile: ./path/to/spec.yaml` (recommended)
- OR all three of: `sourceDir`, `name`, and `version`

**Example:**
```yaml
spec:
  sourceFile: ./api/example-api.v1.yaml  # Option 1 (recommended)
  # OR
  # sourceDir: ./api
  # name: example-api
  # version: v1
```

### Error: "client.packageName is required when client.enabled=true"

**Cause:** Client generation is enabled but no package name is specified.

**Solution:** Add the `packageName` field to the client configuration:

```yaml
spec:
  client:
    enabled: true
    packageName: myclient    # Required when enabled=true
```

### Error: "server.packageName is required when server.enabled=true"

**Cause:** Server generation is enabled but no package name is specified.

**Solution:** Add the `packageName` field to the server configuration:

```yaml
spec:
  server:
    enabled: true
    packageName: myserver    # Required when enabled=true
```

### Error: "at least one of client.enabled or server.enabled must be true"

**Cause:** Both client and server generation are disabled.

**Solution:** Enable at least one type of code generation:

```yaml
spec:
  client:
    enabled: true        # Enable client
    packageName: myclient
  # OR
  server:
    enabled: true        # Enable server
    packageName: myserver
  # OR both
```

### Error: "invalid type for field..."

**Cause:** A field in the spec has the wrong data type (e.g., string instead of boolean).

**Common Issues:**
- `enabled` must be boolean (`true`/`false`), not string (`"true"`/`"false"`)
- `packageName` must be string
- `sourceFile`, `sourceDir`, `destinationDir` must be strings

**Solution:** Verify field types in your YAML:

```yaml
spec:
  sourceFile: ./api/example.yaml    # String (no quotes needed in YAML)
  destinationDir: ./pkg/generated   # String
  client:
    enabled: true                   # Boolean (not "true")
    packageName: myclient           # String
```

### Generated Code Not Found

**Cause:** Code was generated but not at the expected location.

**Solution:**
1. Check the `destinationDir` in your spec configuration
2. Check the `packageName` - generated files are at `{destinationDir}/{packageName}/`
3. Run build with verbose output to see where files are generated
4. Verify file name is `zz_generated.oapi-codegen.go`

**Example:**
```yaml
spec:
  destinationDir: ./pkg/generated
  client:
    packageName: myclient

# Generated file will be at:
# ./pkg/generated/myclient/zz_generated.oapi-codegen.go
```

### Package Name Conflicts Between Versions

**Cause:** Multiple API versions using the same package name.

**Solution:** Use unique package names for each version:

```yaml
build:
  - name: api-v1
    spec:
      client:
        packageName: apiclient        # v1

  - name: api-v2
    spec:
      client:
        packageName: apiclientv2      # v2 (different name)
```

---

## Additional Resources

- **forge.yaml Schema Reference**: See `docs/user/forge-yaml-schema.md` for complete schema documentation
- **go-gen-openapi MCP Documentation**: See `cmd/go-gen-openapi/MCP.md` for MCP interface details
- **Example Configuration**: See the commented example in `forge.yaml` in the build section

If you encounter issues not covered in this guide, please refer to the forge documentation or file an issue.
