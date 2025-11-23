# Using Lazy Rebuild

You are helping a user understand and optimize their forge build process using lazy rebuild to skip rebuilding unchanged artifacts.

## What is Lazy Rebuild?

Lazy rebuild is an automatic optimization in forge that tracks dependencies for build artifacts and skips rebuilding them when nothing has changed. This significantly speeds up incremental builds, especially in large projects or CI/CD pipelines.

## How It Works

When you run `forge build`:

1. **First Build**: All artifacts are built, and forge automatically tracks their dependencies
2. **Subsequent Builds**: Forge checks if dependencies have changed:
   - Compares file modification timestamps
   - Checks go.mod versions for external packages
   - Skips rebuild if nothing changed
   - Rebuilds only if dependencies modified

**Performance Impact:**
- Small projects: Saves seconds per build
- Large projects: Saves minutes per build
- CI/CD: Only rebuilds affected artifacts in PRs

## Quick Start

### Go Binaries (Automatic)

For Go binaries, lazy rebuild works automatically with no configuration:

```yaml
build:
  - name: my-app
    src: ./cmd/my-app
    dest: ./build/bin
    engine: go://go-build
```

**First build:**
```bash
forge build
# Builds my-app and tracks dependencies
```

**Second build (no changes):**
```bash
forge build
# ⏭  Skipping my-app (unchanged)
```

**After modifying code:**
```bash
touch cmd/my-app/main.go
forge build
# 🔨 Building my-app (dependency ./cmd/my-app/main.go modified)
```

### Container Images (Requires Configuration)

For container images, you must explicitly configure dependency tracking:

```yaml
build:
  - name: api-image
    src: ./containers/api/Containerfile
    engine: go://container-build
    spec:
      dependsOn:
        - engine: go://go-dependency-detector
          spec:
            filePath: ./cmd/api/main.go
            funcName: main
```

**Why explicit configuration?**
- Containerfiles can depend on many sources (code, assets, configs)
- Forge needs to know what Go code the container uses
- You specify the entry point (main.go + main function)

## Force Rebuild

To skip lazy rebuild and force rebuild all artifacts:

```bash
forge build --force
# or
forge build -f
```

**When to use --force:**
- After upgrading Go version
- After modifying build environment (LDFLAGS, etc.)
- When troubleshooting build issues
- After manual changes to build outputs

## What Gets Tracked?

### Go Binaries (Automatic)

**File Dependencies:**
- All local .go files imported by main package
- Includes transitive dependencies (A imports B, B imports C)
- Stored as absolute paths with RFC3339 timestamps
- Example:
  ```
  /absolute/path/to/pkg/util/helper.go @ 2025-11-23T10:00:00Z
  ```

**External Package Dependencies:**
- Third-party packages from go.mod
- Package identifier + semantic version
- Supports pseudo-versions
- Example:
  ```
  github.com/foo/bar @ v1.2.3
  ```

### Container Images (Explicit)

Same as Go binaries, but only for dependencies you specify in `dependsOn`.

## Rebuild Reasons

When forge rebuilds an artifact, it shows why:

```bash
# First time building
🔨 Building my-app (no previous build)

# Artifact was deleted
🔨 Building my-app (artifact file missing)

# Source file changed
🔨 Building my-app (dependency /path/to/file.go modified)

# User requested force rebuild
🔨 Building my-app (force flag set)

# Artifact from before lazy rebuild feature
🔨 Building my-app (dependencies not tracked)
```

## Troubleshooting

### All artifacts rebuild every time

**Symptom:** Every `forge build` rebuilds everything

**Possible causes:**
1. Artifact store missing (first build after clean)
2. Using `--force` flag
3. Artifact files deleted
4. Dependencies not tracked (artifacts built before lazy rebuild feature)

**Solution:**
```bash
# Check artifact store exists
ls -la .forge/artifacts.yaml

# Check if dependencies are tracked
cat .forge/artifacts.yaml | grep -A 10 "dependencies:"

# If empty, do one clean build
rm -rf build/ .forge/
forge build
# Now subsequent builds should skip unchanged artifacts
```

### Container images always rebuild

**Symptom:** Container images rebuild even when code unchanged

**Possible causes:**
1. Missing `dependsOn` configuration
2. Incorrect filePath or funcName in spec

**Solution:**
```yaml
# Add dependsOn to container spec
build:
  - name: my-image
    src: ./containers/my-app/Containerfile
    engine: go://container-build
    spec:
      dependsOn:
        - engine: go://go-dependency-detector
          spec:
            filePath: ./cmd/my-app/main.go  # Path to main package
            funcName: main                   # Entry point function
```

### Artifact not rebuilding when it should

**Symptom:** Modified code but artifact not rebuilding

**Possible causes:**
1. Modified file not in dependency tree
2. go.mod changes not detected
3. External file changes (not Go code)

**Solution:**
```bash
# Force rebuild to update dependencies
forge build --force

# Check what dependencies are tracked
cat .forge/artifacts.yaml | grep -A 20 "name: my-app"

# Verify your changes are in tracked files
```

## Advanced Usage

### Container with Multiple Entry Points

If your container has multiple Go entry points:

```yaml
build:
  - name: multi-service-image
    src: ./containers/multi/Containerfile
    engine: go://container-build
    spec:
      dependsOn:
        - engine: go://go-dependency-detector
          spec:
            filePath: ./cmd/api/main.go
            funcName: main
        - engine: go://go-dependency-detector
          spec:
            filePath: ./cmd/worker/main.go
            funcName: main
```

This tracks dependencies from both entry points.

### Debugging Dependency Detection

To see what dependencies are tracked:

```bash
# Build artifact
forge build my-app

# Check artifact store
cat .forge/artifacts.yaml

# Look for your artifact
# Find "dependencies:" section
# Verify file paths and timestamps
```

## Best Practices

### 1. Commit artifact store to git

Add to `.gitignore`:
```gitignore
# Build outputs
build/

# BUT commit artifact store for team sharing
!.forge/artifacts.yaml
```

**Benefits:**
- Team members get immediate lazy rebuild benefits
- CI/CD pipelines can use cached builds
- Consistent build state across team

**Alternative:**
If artifact store contains sensitive paths, keep it in `.gitignore` and accept first-build overhead.

### 2. Use --force in CI for releases

```yaml
# .github/workflows/release.yml
- name: Build release artifacts
  run: forge build --force
```

Ensures release builds are always fresh.

### 3. Configure dependsOn for all containers

If containers use Go code, always add `dependsOn`:

```yaml
# ❌ Bad: Container won't track Go code changes
- name: api-image
  src: ./containers/api/Containerfile
  engine: go://container-build

# ✅ Good: Container tracks Go code changes
- name: api-image
  src: ./containers/api/Containerfile
  engine: go://container-build
  spec:
    dependsOn:
      - engine: go://go-dependency-detector
        spec:
          filePath: ./cmd/api/main.go
          funcName: main
```

### 4. Keep artifact store clean

Artifact store auto-prunes old builds (keeps 3 most recent). No manual maintenance needed.

## Performance Tips

### For Large Projects

**Parallel builds:**
Lazy rebuild doesn't affect parallelism. Forge still builds in parallel when needed.

**Selective rebuilds:**
```bash
# Rebuild only specific artifact
forge build my-app

# Rebuild all (respects lazy rebuild)
forge build
```

### For CI/CD

**Cache artifact store:**
```yaml
# .github/workflows/build.yml
- uses: actions/cache@v3
  with:
    path: .forge/artifacts.yaml
    key: forge-artifacts-${{ hashFiles('go.sum') }}
```

Benefits:
- Skip rebuilding unchanged artifacts across CI runs
- Faster PR builds
- Reduced CI minutes

## Limitations

Current limitations of lazy rebuild:

1. **Go Only**: Only works for Go code dependencies (not Python, Rust, etc.)
2. **Main Packages Only**: Only tracks dependencies for main packages (not libraries)
3. **No Build Environment Tracking**: Doesn't detect changes to:
   - Go version
   - Environment variables (LDFLAGS, GOOS, etc.)
   - Build tools (compiler updates)
4. **Container Requires Config**: Containers need explicit `dependsOn` configuration

**Workarounds:**
- Use `--force` when changing build environment
- For non-Go dependencies, consider custom build engine
- For libraries, lazy rebuild tracks when they're used by main packages

## Reference

### Rebuild Decision Logic

Forge rebuilds if ANY of:
- `--force` flag used
- No previous build found
- Artifact file missing
- Dependencies not tracked
- Any file dependency timestamp changed
- Dependency detector not configured (containers)

Forge skips rebuild if ALL of:
- Previous build exists
- Artifact file exists
- Dependencies tracked
- All file timestamps match
- Dependency detector configured (if needed)

### Related Documentation

- **forge.yaml schema**: [docs/forge-schema.md](../forge-schema.md#lazy-rebuild)
- **Architecture**: [ARCHITECTURE.md](../../ARCHITECTURE.md#lazy-rebuild)
- **go-dependency-detector**: [cmd/go-dependency-detector/MCP.md](../../cmd/go-dependency-detector/MCP.md)

## Common Patterns

### Pattern 1: Multi-Binary Project

```yaml
name: my-project
artifactStorePath: .forge/artifacts.yaml

build:
  # CLI tool
  - name: cli
    src: ./cmd/cli
    dest: ./build/bin
    engine: go://go-build

  # API server
  - name: api
    src: ./cmd/api
    dest: ./build/bin
    engine: go://go-build

  # Worker
  - name: worker
    src: ./cmd/worker
    dest: ./build/bin
    engine: go://go-build
```

**Behavior:**
- First build: All 3 binaries built
- Modify `cmd/cli/main.go`: Only cli rebuilt
- Modify `pkg/shared/util.go`: All 3 rebuilt (if they import it)

### Pattern 2: Service with Container

```yaml
build:
  # Build Go binary
  - name: api
    src: ./cmd/api
    dest: ./build/bin
    engine: go://go-build

  # Build container using binary
  - name: api-image
    src: ./containers/api/Containerfile
    engine: go://container-build
    spec:
      dependsOn:
        - engine: go://go-dependency-detector
          spec:
            filePath: ./cmd/api/main.go
            funcName: main
```

**Behavior:**
- Modify Go code: Both binary and image rebuild
- Modify Containerfile: Only image rebuilds
- No changes: Both skip rebuild

### Pattern 3: Microservices

```yaml
build:
  # Auth service
  - name: auth
    src: ./cmd/auth
    dest: ./build/bin
    engine: go://go-build
  - name: auth-image
    src: ./containers/auth/Containerfile
    engine: go://container-build
    spec:
      dependsOn:
        - engine: go://go-dependency-detector
          spec:
            filePath: ./cmd/auth/main.go
            funcName: main

  # Payment service
  - name: payment
    src: ./cmd/payment
    dest: ./build/bin
    engine: go://go-build
  - name: payment-image
    src: ./containers/payment/Containerfile
    engine: go://container-build
    spec:
      dependsOn:
        - engine: go://go-dependency-detector
          spec:
            filePath: ./cmd/payment/main.go
            funcName: main
```

**Behavior:**
- Modify auth service: Only auth binary and image rebuild
- Modify payment service: Only payment binary and image rebuild
- Modify shared library: All services rebuild
