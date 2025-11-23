# testenv-helm-install MCP Server

MCP server for installing Helm charts into Kubernetes clusters for test environments.

## Purpose

Installs and manages Helm charts as part of test environment setup. Works with kubeconfig files provided by testenv-kind to deploy charts into test clusters.

## Invocation

```bash
testenv-helm-install --mcp
```

Called by testenv orchestrator automatically.

## Available Tools

### `create`

Install Helm charts into a Kubernetes cluster.

**Input Schema:**

The `spec.charts` field accepts an array of ChartSpec objects with the following structure:

#### Core Fields (Required)

- `name` (string, required): Internal identifier for this chart configuration
- `sourceType` (string, required): Artifact acquisition strategy. Valid values:
  - `"helm-repo"`: Helm repository (HTTP/S)
  - `"git"`: Git repository
  - `"oci"`: OCI registry (requires Helm 3.8+)
  - `"s3"`: S3-compatible storage (AWS S3, MinIO, GCS)
- `url` (string, required): Primary locator for the source
  - For `helm-repo`: HTTP/S URL of the Helm repository
  - For `git`: HTTP/S or SSH URL of the git repo
  - For `oci`: Registry URL starting with `oci://` (e.g., `oci://ghcr.io/org/charts/mychart`)
  - For `s3`: HTTP/S URL of the S3-compatible endpoint (e.g., `http://localhost:9000` for MinIO, `https://s3.amazonaws.com` for AWS)

#### Helm Repository Fields (for sourceType="helm-repo")

- `chartName` (string, required): Name of the chart to fetch from the Helm repository
- `version` (string, optional): Chart version constraint (e.g., "6.0.0", "^1.0.0"). Defaults to "*" (latest)

#### Git Repository Fields (for sourceType="git")

- `chartPath` (string, required): Relative path to the chart directory within the repository (e.g., "charts/app")
- Git reference (at least one required, precedence: Commit > Tag > SemVer > Branch):
  - `gitCommit` (string, optional): Exact Git commit SHA to checkout (minimum 7 characters)
  - `gitTag` (string, optional): Git tag to checkout (e.g., "v1.0.0")
  - `gitSemVer` (string, optional): SemVer constraint to resolve against Git tags (e.g., "^1.0.0", ">=1.0.0 <2.0.0")
  - `gitBranch` (string, optional): Git branch to checkout (e.g., "main", "develop")
- `ignorePaths` ([]string, optional): .gitignore-style patterns to exclude (optimization placeholder, logs warning if used)

#### OCI Registry Fields (for sourceType="oci")

- `url` (string, required): OCI registry URL in format `oci://REGISTRY/REPOSITORY/CHART`
  - Examples:
    - `oci://ghcr.io/stefanprodan/charts/podinfo` (uses latest tag)
    - `oci://ghcr.io/stefanprodan/charts/podinfo:6.0.0` (specific version)
    - `oci://ghcr.io/stefanprodan/charts/podinfo@sha256:abc123...` (specific digest)
- `version` (string, optional): Chart version (alternative to specifying version in URL with `:tag`)
- `authSecretName` (string, optional): Name of Kubernetes Secret (type: `kubernetes.io/dockerconfigjson`) containing registry credentials
  - For private OCI registries, create a Secret with Docker config JSON format
  - Example: `kubectl create secret docker-registry oci-creds --docker-server=ghcr.io --docker-username=user --docker-password=token`
- `ociProvider` (string, optional): Signature verification provider. Values: `"cosign"`, `"notation"`
  - Currently logs a warning when set; actual cryptographic verification is reserved for future implementation

**Note**: Requires Helm 3.8 or later for OCI support. The chart name is embedded in the OCI URL, so `chartName` field should not be set.

#### S3 Bucket Fields (for sourceType="s3")

- `url` (string, required): S3-compatible endpoint URL (e.g., `http://localhost:9000` for MinIO, `https://s3.amazonaws.com` for AWS S3)
- `s3BucketName` (string, required): Name of the S3 bucket containing the chart
- `chartPath` (string, required): Path to the chart tarball within the bucket (e.g., `charts/myapp-1.0.0.tgz`)
  - Must end with `.tgz` or `.tar.gz`
  - Relative path from bucket root
- `s3BucketRegion` (string, optional): AWS region for the bucket. Defaults to `"us-east-1"`
- `authSecretName` (string, optional): Name of Kubernetes Secret containing S3 credentials
  - Secret must contain keys: `accessKeyID` (required), `secretAccessKey` (required), `sessionToken` (optional)
  - If not set, uses default AWS credentials (IAM role, environment variables)
  - Example: `kubectl create secret generic s3-creds --from-literal=accessKeyID=AKIAIOSFODNN7EXAMPLE --from-literal=secretAccessKey=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`

**Example S3 Chart Configuration:**
```yaml
spec:
  charts:
    - name: myapp
      sourceType: s3
      url: http://localhost:9000
      s3BucketName: helm-charts
      chartPath: production/myapp-1.2.3.tgz
      s3BucketRegion: us-east-1
      authSecretName: s3-creds
      namespace: myapp
      createNamespace: true
```

**Note**: The chart tarball is downloaded from S3 before installation. Git, OCI, and `chartName` fields should not be set for S3 sources.

#### Core Configuration

- `releaseName` (string, optional): Helm release name in the cluster. Defaults to `name` if not specified
- `namespace` (string, optional): Kubernetes namespace for the release. Defaults to "default"
- `createNamespace` (bool, optional): Create namespace if it doesn't exist. Defaults to false

#### Lifecycle & Remediation

- `timeout` (string, optional): Time to wait for Helm operations (e.g., "5m", "10m"). Defaults to "5m"
- `disableWait` (bool, optional): Skip waiting for resources to be ready. Defaults to false
- `forceUpgrade` (bool, optional): Use `helm upgrade --force` (recreates resources). Defaults to false
- `disableHooks` (bool, optional): Disable Helm hooks. Defaults to false
- `testEnable` (bool, optional): Run helm tests after installation. Defaults to false

#### Values Configuration

Values are composed from multiple sources with the following precedence (lowest to highest):
1. `valuesFiles` - Helm values files within the source artifact (lowest precedence)
2. `valueReferences` - Values from Kubernetes ConfigMaps/Secrets
3. `values` - Inline values (highest precedence)

- `values` (map[string]interface{}, optional): Inline Helm values to apply (highest precedence)
  - Supports nested maps, arrays, and all YAML types (strings, numbers, booleans, null)
  - Values are serialized to YAML and passed to Helm via `--values` flag
  - Type preservation: numbers remain numbers, booleans remain booleans, etc.
- `valuesFiles` ([]string, optional): Paths to values files within the source artifact (lowest precedence)
- `valueReferences` ([]ValueReference, optional): References to ConfigMaps/Secrets containing values

##### Nested Values Support

The `values` field supports arbitrary YAML structures including:

- **Nested maps**: Create hierarchical configuration structures
- **Arrays**: Define lists of items or environment variables
- **Mixed types**: Combine strings, numbers, booleans, and null values
- **Type preservation**: Values maintain their types (integers stay integers, not strings)

**Example: Nested Values**
```yaml
values:
  server:
    replicas: 3  # Number type preserved
    config:
      database:
        host: "db.example.com"
        port: 5432  # Number type preserved
      cache:
        enabled: true  # Boolean type preserved
        ttl: 300
    env:
      - name: "ENV"
        value: "production"
      - name: "DEBUG"
        value: "false"
```

This is equivalent to the following YAML values file:
```yaml
server:
  replicas: 3
  config:
    database:
      host: db.example.com
      port: 5432
    cache:
      enabled: true
      ttl: 300
  env:
    - name: ENV
      value: production
    - name: DEBUG
      value: "false"
```

#### ValueReferences

ValueReferences allow sourcing Helm values from existing Kubernetes ConfigMaps and Secrets. Each ValueReference has the following fields:

- `kind` (string, required): Resource type. Valid values: `"ConfigMap"`, `"Secret"`
- `name` (string, required): Name of the ConfigMap or Secret
- `valuesKey` (string, optional): Specific key to extract from the resource's data
  - If empty, all keys are merged into the values
  - If specified, only that key's value is used (parsed as YAML if applicable)
- `targetPath` (string, optional): Dot-notation path where values should be merged (e.g., `"server.config"`)
  - If empty, values are merged at the root level
  - Creates intermediate keys as needed
- `optional` (bool, optional): Whether the reference is optional. Defaults to `false`
  - If `true` and resource is not found, silently skips without error
  - If `false` and resource is not found, installation fails

**ValueReference Examples:**

Example 1: Simple ConfigMap reference (merge all keys at root):
```yaml
spec:
  charts:
    - name: myapp
      sourceType: helm-repo
      url: https://charts.example.com
      chartName: myapp
      namespace: production
      valueReferences:
        - kind: ConfigMap
          name: myapp-config
          # All keys from myapp-config merged at root level
```

Example 2: Extract specific key from Secret:
```yaml
spec:
  charts:
    - name: myapp
      sourceType: helm-repo
      url: https://charts.example.com
      chartName: myapp
      namespace: production
      valueReferences:
        - kind: Secret
          name: database-credentials
          valuesKey: connection-string
          targetPath: database.connectionString
          # Merges the 'connection-string' value to database.connectionString
```

Example 3: Optional reference with TargetPath:
```yaml
spec:
  charts:
    - name: myapp
      sourceType: helm-repo
      url: https://charts.example.com
      chartName: myapp
      namespace: production
      valueReferences:
        - kind: ConfigMap
          name: feature-flags
          optional: true
          targetPath: features
          # If feature-flags ConfigMap exists, merge to features.*
          # If not found, continue without error
```

Example 4: Multiple ValueReferences with precedence:
```yaml
spec:
  charts:
    - name: myapp
      sourceType: helm-repo
      url: https://charts.example.com
      chartName: myapp
      namespace: production
      valueReferences:
        - kind: ConfigMap
          name: base-config
          # First: merge all keys from base-config
        - kind: ConfigMap
          name: environment-overrides
          # Second: merge all keys from environment-overrides (overrides base-config)
        - kind: Secret
          name: secrets
          targetPath: secure
          # Third: merge secrets to secure.* path
      values:
        # Inline values have highest precedence and override everything above
        replicaCount: 3
```

**Important Notes:**
- ConfigMap and Secret must exist in the same namespace as the chart (specified by `namespace` field, defaults to `"default"`)
- Secret values are automatically base64-decoded
- Values are parsed as YAML if they contain YAML structures
- For value composition order, see the precedence list above

#### Authentication & Security

- `authSecretName` (string, optional): Name of Kubernetes Secret containing credentials
- `passCredentials` (bool, optional): Pass credentials to chart download. Defaults to false
- `insecureSkipVerify` (bool, optional): Skip TLS verification (development only). Defaults to false

#### Advanced Fields

The following fields are defined for future enhancement:
- `interval` (string): Reconciliation frequency (reserved for future use)
- `ociProvider` (string): OCI signature verification provider (`"cosign"` or `"notation"`)
  - Currently logs a warning when set but does not perform actual verification
  - Reserved for future implementation of cryptographic signature validation

**Output:**
```json
{
  "testID": "string",
  "files": {},
  "metadata": {
    "testenv-helm-install.chartCount": "2",
    "testenv-helm-install.chart.0.name": "cert-manager",
    "testenv-helm-install.chart.0.releaseName": "cert-manager",
    "testenv-helm-install.chart.0.namespace": "cert-manager",
    "testenv-helm-install.chart.1.name": "nginx-ingress",
    "testenv-helm-install.chart.1.releaseName": "nginx-ingress"
  },
  "managedResources": []
}
```

**What It Does:**
1. Locates kubeconfig from metadata (provided by testenv-kind)
2. For each chart in spec.charts:
   - Adds Helm repository if specified
   - Runs `helm install` with provided configuration
   - Stores chart metadata for cleanup
3. Returns metadata with installed chart information

### Example Usage

#### Basic Helm Repository Chart

```json
{
  "testID": "test-integration-20240101-abc123",
  "stage": "integration",
  "spec": {
    "charts": [
      {
        "name": "podinfo-release",
        "sourceType": "helm-repo",
        "url": "https://stefanprodan.github.io/podinfo",
        "chartName": "podinfo",
        "version": "6.0.0",
        "namespace": "test-podinfo",
        "releaseName": "test-podinfo",
        "createNamespace": true,
        "timeout": "5m",
        "disableWait": false
      }
    ]
  },
  "metadata": {
    "testenv-kind.kubeconfigPath": "/path/to/kubeconfig"
  }
}
```

#### Advanced Example with Values

```json
{
  "charts": [
    {
      "name": "nginx-release",
      "sourceType": "helm-repo",
      "url": "https://charts.bitnami.com/bitnami",
      "chartName": "nginx",
      "version": "^15.0.0",
      "namespace": "web",
      "releaseName": "my-nginx",
      "createNamespace": true,
      "timeout": "10m",
      "forceUpgrade": false,
      "disableHooks": false,
      "testEnable": true,
      "values": {
        "replicaCount": 3,
        "service": {
          "type": "LoadBalancer"
        }
      },
      "valuesFiles": ["custom-values.yaml"]
    }
  ]
}
```

#### Git Source with Branch

```json
{
  "charts": [
    {
      "name": "podinfo-from-git",
      "sourceType": "git",
      "url": "https://github.com/stefanprodan/podinfo",
      "gitBranch": "master",
      "chartPath": "charts/podinfo",
      "namespace": "test-git",
      "releaseName": "podinfo-git",
      "createNamespace": true,
      "timeout": "5m"
    }
  ]
}
```

#### Git Source with Tag

```json
{
  "charts": [
    {
      "name": "podinfo-v6",
      "sourceType": "git",
      "url": "https://github.com/stefanprodan/podinfo",
      "gitTag": "6.0.0",
      "chartPath": "charts/podinfo",
      "namespace": "test-git-tag",
      "createNamespace": true
    }
  ]
}
```

#### Git Source with Commit

```json
{
  "charts": [
    {
      "name": "podinfo-commit",
      "sourceType": "git",
      "url": "https://github.com/stefanprodan/podinfo",
      "gitCommit": "abc1234def5678",
      "chartPath": "charts/podinfo",
      "namespace": "test-git-commit",
      "createNamespace": true
    }
  ]
}
```

#### OCI Registry (Public)

```json
{
  "charts": [
    {
      "name": "podinfo-oci",
      "sourceType": "oci",
      "url": "oci://ghcr.io/stefanprodan/charts/podinfo",
      "version": "6.0.0",
      "namespace": "test-oci",
      "releaseName": "podinfo-oci",
      "createNamespace": true,
      "timeout": "5m"
    }
  ]
}
```

#### OCI Registry (Private with Authentication)

```json
{
  "charts": [
    {
      "name": "private-chart",
      "sourceType": "oci",
      "url": "oci://ghcr.io/myorg/charts/myapp:1.2.3",
      "authSecretName": "oci-registry-creds",
      "namespace": "production",
      "createNamespace": true,
      "values": {
        "replicas": 3,
        "image": {
          "tag": "v1.2.3"
        }
      }
    }
  ]
}
```

**Note**: Create the auth secret beforehand:
```bash
kubectl create secret docker-registry oci-registry-creds \
  --docker-server=ghcr.io \
  --docker-username=myusername \
  --docker-password=mytoken \
  --namespace=production
```

#### OCI Registry with Digest

```json
{
  "charts": [
    {
      "name": "pinned-chart",
      "sourceType": "oci",
      "url": "oci://ghcr.io/stefanprodan/charts/podinfo@sha256:abc123def456789...",
      "namespace": "test-oci-digest",
      "createNamespace": true
    }
  ]
}
```

#### Git Source with SemVer Constraint

```json
{
  "charts": [
    {
      "name": "podinfo-semver",
      "sourceType": "git",
      "url": "https://github.com/stefanprodan/podinfo",
      "gitSemVer": "^6.0.0",
      "chartPath": "charts/podinfo",
      "namespace": "test-git-semver",
      "createNamespace": true,
      "values": {
        "replicaCount": 2
      }
    }
  ]
}
```

#### Git Source with SSH URL

```json
{
  "charts": [
    {
      "name": "private-chart",
      "sourceType": "git",
      "url": "git@github.com:organization/private-repo.git",
      "gitBranch": "main",
      "chartPath": "deploy/charts/myapp",
      "namespace": "production",
      "createNamespace": true
    }
  ]
}
```

### `delete`

Uninstall Helm charts from a Kubernetes cluster.

**Input Schema:**
```json
{
  "testID": "string (required)",     // Test environment ID
  "metadata": {                       // Metadata from test environment
    "testenv-helm-install.chartCount": "2",
    "testenv-helm-install.chart.0.releaseName": "cert-manager",
    "testenv-helm-install.chart.0.namespace": "cert-manager",
    "testenv-kind.kubeconfigPath": "/path/to/kubeconfig"
  }
}
```

**Output:**
```json
{
  "success": true,
  "message": "Uninstalled 2 Helm chart(s)"
}
```

**What It Does:**
1. Extracts chart information from metadata
2. Uninstalls charts in reverse order (last installed, first removed)
3. Best-effort cleanup (logs warnings but continues on errors)

## Integration

Called by testenv MCP server during test environment creation/deletion. Must be positioned after testenv-kind in the testenv subengine list to ensure kubeconfig is available.

## Configuration

Example in `forge.yaml`:
```yaml
engines:
  - alias: k8s-with-helm
    type: testenv
    testenv:
      - engine: "go://testenv-kind"
      - engine: "go://testenv-lcr"
        spec:
          enabled: true
          autoPushImages: true
      - engine: "go://testenv-helm-install"
        spec:
          charts:
            - name: cert-manager-release
              sourceType: helm-repo
              url: https://charts.jetstack.io
              chartName: cert-manager
              version: v1.13.0
              namespace: cert-manager
              releaseName: cert-manager
              createNamespace: true
              values:
                installCRDs: true
            - name: nginx-ingress-release
              sourceType: helm-repo
              url: https://kubernetes.github.io/ingress-nginx
              chartName: ingress-nginx
              namespace: ingress-nginx
              releaseName: nginx-ingress
              createNamespace: true
```

## Implementation Details

- Uses `helm` CLI commands (requires helm to be installed)
- Finds kubeconfig from testenv-kind metadata
- Charts are installed sequentially in order
- Charts are uninstalled in reverse order during cleanup
- Supports custom release names, namespaces, and values
- Creates namespaces automatically if specified

## Requirements

- Helm CLI must be installed and available in PATH
- Kubeconfig must be provided by testenv-kind
- Charts must be accessible (public repos or pre-configured repos)

## See Also

- [ChartSpec Reference](../../docs/testenv-helm-install-chartspec.md) - Complete field documentation
- [Migration Guide](../../docs/testenv-helm-install-migration.md) - Upgrade from old format
- [Testing Guide](../../docs/testenv-helm-install-testing.md) - Test infrastructure setup
- [testenv MCP Server](../testenv/MCP.md)
- [testenv-kind MCP Server](../testenv-kind/MCP.md)
- [testenv-lcr MCP Server](../testenv-lcr/MCP.md)
