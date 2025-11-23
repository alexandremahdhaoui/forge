# testenv-helm-install Migration Guide

This guide helps users migrate from the old ChartSpec format to the new multi-source format.

## Overview

The testenv-helm-install engine has been significantly enhanced to support multiple chart sources and advanced value composition. This guide covers:

1. Breaking changes in the ChartSpec format
2. Migration from old to new format
3. New features available after migration
4. Troubleshooting common migration issues

## Breaking Changes

### 1. Old `repo` Field Removed

**What Changed:**
- The old `repo` field is no longer supported
- The chart name format `"reponame/chartname"` is no longer supported
- New `sourceType` and `url` fields are now required

**Old Format (No Longer Works):**
```yaml
spec:
  charts:
    - name: podinfo/podinfo
      repo: https://stefanprodan.github.io/podinfo
      namespace: test-podinfo
```

**New Format (Required):**
```yaml
spec:
  charts:
    - name: podinfo-release
      sourceType: helm-repo
      url: https://stefanprodan.github.io/podinfo
      chartName: podinfo
      namespace: test-podinfo
```

### 2. Chart Name Specification

**What Changed:**
- Chart name is now separated from internal identifier
- `name` is now an internal identifier
- `chartName` specifies the actual Helm chart name (for helm-repo source type)
- `releaseName` specifies the Helm release name in the cluster (defaults to `name` if not set)

**Old Format:**
```yaml
charts:
  - name: nginx/nginx
    repo: https://charts.bitnami.com/bitnami
```

**New Format:**
```yaml
charts:
  - name: nginx-release        # Internal identifier
    sourceType: helm-repo
    url: https://charts.bitnami.com/bitnami
    chartName: nginx           # Chart name in repository
    releaseName: my-nginx      # Release name in cluster (optional, defaults to 'nginx-release')
```

## Migration Steps

### Step 1: Identify All ChartSpec Configurations

Find all instances of chart specifications in your project:

```bash
# Find forge.yaml files with testenv configurations
find . -name "forge.yaml" -exec grep -l "testenv:" {} \;

# Find chart specifications in test files
grep -r "charts:" . --include="*.yaml" --include="*.json"
```

### Step 2: Update Each Chart Configuration

For each chart, follow this pattern:

**Before:**
```yaml
spec:
  charts:
    - name: cert-manager/cert-manager
      repo: https://charts.jetstack.io
      namespace: cert-manager
      version: "v1.12.0"
      values:
        installCRDs: true
```

**After:**
```yaml
spec:
  charts:
    - name: cert-manager-release
      sourceType: helm-repo
      url: https://charts.jetstack.io
      chartName: cert-manager
      version: "v1.12.0"
      namespace: cert-manager
      createNamespace: true
      values:
        installCRDs: true
```

### Step 3: Add New Required Fields

Ensure all charts have the required fields:

- `sourceType`: One of `"helm-repo"`, `"git"`, `"oci"`, `"s3"`
- `url`: The source URL
- Source-specific fields:
  - For `helm-repo`: `chartName` is required
  - For `git`: `chartPath` and at least one git reference (branch/tag/commit/semver) are required
  - For `oci`: Chart name is embedded in URL, no `chartName` field needed
  - For `s3`: `s3BucketName` and `chartPath` are required

### Step 4: Test the Migration

Run your tests to ensure the migration works:

```bash
# Build all artifacts
forge build

# Run test stage with charts
forge test integration run

# Or run full test suite
forge test-all
```

## New Features Available After Migration

Once migrated, you can use these new features:

### 1. Git Source Type

Install charts directly from Git repositories:

```yaml
charts:
  - name: podinfo-from-git
    sourceType: git
    url: https://github.com/stefanprodan/podinfo
    gitBranch: master
    chartPath: charts/podinfo
    namespace: test-git
    createNamespace: true
```

### 2. OCI Registry Source

Pull charts from OCI registries:

```yaml
charts:
  - name: podinfo-oci
    sourceType: oci
    url: oci://ghcr.io/stefanprodan/charts/podinfo:6.0.0
    namespace: test-oci
    createNamespace: true
```

### 3. S3 Bucket Source

Download charts from S3-compatible storage:

```yaml
charts:
  - name: myapp-from-s3
    sourceType: s3
    url: http://localhost:9000
    s3BucketName: helm-charts
    chartPath: production/myapp-1.2.3.tgz
    s3BucketRegion: us-east-1
    authSecretName: s3-creds
    namespace: myapp
    createNamespace: true
```

### 4. ValueReferences

Load values from Kubernetes ConfigMaps and Secrets:

```yaml
charts:
  - name: myapp
    sourceType: helm-repo
    url: https://charts.example.com
    chartName: myapp
    namespace: production
    createNamespace: true
    valueReferences:
      - kind: ConfigMap
        name: base-config
      - kind: Secret
        name: secrets
        targetPath: secure
        optional: true
    values:
      replicaCount: 3
```

### 5. Nested Values

Use complex nested structures in the `values` field:

```yaml
charts:
  - name: complex-app
    sourceType: helm-repo
    url: https://charts.example.com
    chartName: myapp
    namespace: production
    values:
      server:
        replicas: 3
        config:
          database:
            host: db.example.com
            port: 5432
          cache:
            enabled: true
            ttl: 300
      resources:
        limits:
          cpu: 200m
          memory: 256Mi
```

## Troubleshooting

### Error: "sourceType is required"

**Symptom:**
```
Error: sourceType is required
```

**Solution:**
Add the `sourceType` field to your chart specification. Choose one of: `"helm-repo"`, `"git"`, `"oci"`, `"s3"`.

**Example:**
```yaml
charts:
  - name: myapp
    sourceType: helm-repo  # Add this field
    url: https://charts.example.com
    chartName: myapp
```

### Error: "chartName is required for helm-repo source"

**Symptom:**
```
Error: chartName is required when sourceType is 'helm-repo'
```

**Solution:**
Extract the chart name from your old `name` field and add it as `chartName`.

**Before:**
```yaml
name: nginx/nginx
repo: https://charts.bitnami.com/bitnami
```

**After:**
```yaml
name: nginx-release
sourceType: helm-repo
url: https://charts.bitnami.com/bitnami
chartName: nginx
```

### Error: "field 'repo' not recognized"

**Symptom:**
```
Warning: field 'repo' is not recognized and will be ignored
Error: url is required
```

**Solution:**
Replace `repo:` with `url:` and add `sourceType: helm-repo`.

**Before:**
```yaml
repo: https://charts.example.com
```

**After:**
```yaml
sourceType: helm-repo
url: https://charts.example.com
```

### Chart Not Found After Migration

**Symptom:**
Helm reports "chart not found" even though the old format worked.

**Possible Causes:**
1. `chartName` doesn't match the actual chart name in the repository
2. `url` points to the wrong repository
3. `version` constraint doesn't match any available versions

**Solution:**
Verify chart availability manually:

```bash
# Add the repository
helm repo add myrepo https://charts.example.com

# Search for the chart
helm search repo myrepo/myapp

# Verify the exact chart name and available versions
helm search repo myrepo/ --versions
```

### Values Not Being Applied

**Symptom:**
Chart installs but values from `values` field are not applied.

**Possible Causes:**
1. YAML formatting issues in nested values
2. Value precedence not understood
3. Chart doesn't support the specified values

**Solution:**
1. Verify your values are valid YAML:
   ```bash
   # Test values in a separate file
   cat > test-values.yaml <<EOF
   your:
     values:
       here: true
   EOF
   yamllint test-values.yaml
   ```

2. Understand value precedence:
   - `valuesFiles` (lowest)
   - `valueReferences` (middle)
   - `values` (highest)

3. Check chart's values.yaml for supported fields:
   ```bash
   helm show values bitnami/nginx
   ```

### Authentication Errors with Private Sources

**Symptom:**
```
Error: failed to download chart: 401 Unauthorized
```

**Solution:**
Add `authSecretName` with appropriate credentials:

For OCI registries:
```bash
kubectl create secret docker-registry oci-creds \
  --docker-server=ghcr.io \
  --docker-username=myuser \
  --docker-password=mytoken \
  --namespace=production
```

For S3 sources:
```bash
kubectl create secret generic s3-creds \
  --from-literal=accessKeyID=YOUR_ACCESS_KEY \
  --from-literal=secretAccessKey=YOUR_SECRET_KEY \
  --namespace=production
```

Then reference in your chart:
```yaml
charts:
  - name: private-chart
    sourceType: oci
    url: oci://ghcr.io/myorg/charts/myapp
    authSecretName: oci-creds
    namespace: production
```

## Backward Compatibility

**Important:** There is NO backward compatibility with the old format. All chart specifications must be migrated to the new format.

**Timeline:**
- Old format was deprecated: Version 0.x
- Old format removed: Current version
- No migration path preserves old configurations

**Recommendation:**
Update all configurations as part of a single migration effort to avoid confusion between old and new formats.

## Additional Resources

- [testenv-helm-install ChartSpec Reference](./testenv-helm-install-chartspec.md) - Complete field reference
- [testenv-helm-install MCP Documentation](../cmd/testenv-helm-install/MCP.md) - MCP server details and examples
- [FluxCD HelmRelease API](https://fluxcd.io/flux/components/helm/helmreleases/) - Inspiration for the new format
- [Helm Documentation](https://helm.sh/docs/) - Helm basics and concepts

## Getting Help

If you encounter issues during migration:

1. Review the [testenv-helm-install Testing Guide](./testenv-helm-install-testing.md)
2. Examine the examples in [testenv-helm-install-chartspec.md](./testenv-helm-install-chartspec.md)
3. Check the troubleshooting section above
4. Open an issue with:
   - Your old ChartSpec configuration
   - Your new ChartSpec configuration
   - Complete error messages
   - Output of `forge build` and `forge test <stage> run`
