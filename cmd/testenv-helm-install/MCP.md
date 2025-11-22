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
  - `"git"`: Git repository (not yet implemented)
  - `"oci"`: OCI registry (not yet implemented)
  - `"s3"`: S3 bucket (not yet implemented)
- `url` (string, required): Primary locator for the source
  - For `helm-repo`: HTTP/S URL of the Helm repository
  - For `git`: HTTP/S or SSH URL of the git repo
  - For `oci`: Registry URL starting with `oci://`
  - For `s3`: S3-compatible endpoint

#### Helm Repository Fields (for sourceType="helm-repo")

- `chartName` (string, required): Name of the chart to fetch from the Helm repository
- `version` (string, optional): Chart version constraint (e.g., "6.0.0", "^1.0.0"). Defaults to "*" (latest)

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

- `values` (map[string]interface{}, optional): Inline Helm values to apply
- `valuesFiles` ([]string, optional): Paths to values files within the source artifact
- `valueReferences` ([]ValueReference, optional): References to ConfigMaps/Secrets (not yet implemented)

#### Authentication & Security

- `authSecretName` (string, optional): Name of Kubernetes Secret containing credentials
- `passCredentials` (bool, optional): Pass credentials to chart download. Defaults to false
- `insecureSkipVerify` (bool, optional): Skip TLS verification (development only). Defaults to false

#### Advanced Fields (Future Implementation)

The following fields are defined but not yet implemented:
- Git repository fields: `chartPath`, `gitBranch`, `gitTag`, `gitCommit`, `gitSemVer`, `ignorePaths`
- OCI repository fields: `ociProvider`, `ociLayerMediaType`
- S3 bucket fields: `s3BucketName`, `s3BucketRegion`
- `interval` (string): Reconciliation frequency

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

- [testenv MCP Server](../testenv/MCP.md)
- [testenv-kind MCP Server](../testenv-kind/MCP.md)
- [testenv-lcr MCP Server](../testenv-lcr/MCP.md)
