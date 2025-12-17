# Lazy Rebuild

**Skip rebuilding unchanged artifacts automatically.**

> "My full build went from 3 minutes to 10 seconds on incremental changes. I didn't configure anything - it just worked."

## What is lazy rebuild?

Lazy rebuild is an automatic optimization that tracks dependencies for build artifacts and skips rebuilding when nothing has changed. Forge compares file timestamps and go.mod versions to determine what needs rebuilding.

## How does forge know when to rebuild?

**Forge rebuilds if ANY of:**
- `--force` flag used
- No previous build exists
- Artifact file is missing
- Any source file timestamp changed
- Any go.mod dependency version changed

**Forge skips rebuild if ALL of:**
- Previous build exists
- Artifact file exists
- All file timestamps match
- All go.mod versions match

## How do I see it in action?

```bash
# First build - builds everything
forge build
# Output: Building my-app...

# Second build - nothing changed
forge build
# Output: Skipping my-app (unchanged)

# After modifying code
touch cmd/my-app/main.go
forge build
# Output: Building my-app (dependency ./cmd/my-app/main.go modified)
```

## How do I force a rebuild?

```bash
forge build --force
# or
forge build -f
```

Use `--force` after:
- Upgrading Go version
- Changing build environment (LDFLAGS, GOOS, etc.)
- Manual changes to build outputs

## How do I debug rebuild issues?

**Artifacts always rebuild:**
```bash
# Check artifact store exists
ls -la .forge/artifacts.yaml

# Check if dependencies are tracked
cat .forge/artifacts.yaml | grep -A 10 "dependencies:"

# Clean rebuild to reset tracking
rm -rf build/ .forge/
forge build
```

**Artifacts never rebuild when they should:**
```bash
# Force rebuild to update dependency tracking
forge build --force

# Verify your changed file is in the dependency tree
cat .forge/artifacts.yaml | grep -A 20 "name: my-app"
```

## What are the limitations?

- **Go only**: Only tracks Go code dependencies
- **Main packages only**: Tracks dependencies for main packages, not libraries
- **No build environment tracking**: Changes to Go version, LDFLAGS, GOOS require `--force`
- **Containers need explicit config**: Container images require `dependsOn` configuration

## What's next?

- [Schema Reference](./forge-yaml-schema.md) - `dependsOn` configuration for containers
- [Getting Started](./getting-started.md) - Basic forge setup
