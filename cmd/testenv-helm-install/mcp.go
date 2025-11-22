package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
)

// ChartSpec is a monolithic specification for a Helm Chart deployment.
// It synthesizes the capabilities of FluxCD's HelmRelease and Source CRDs
// (GitRepository, HelmRepository, OCIRepository, Bucket) into a single config.
type ChartSpec struct {
	// -------------------------------------------------------------------------
	// Core Identity & Location
	// -------------------------------------------------------------------------

	// Name represents the internal identifier for this chart configuration.
	// It is used to name the generated Flux Custom Resources.
	// Required.
	Name string `json:"name" yaml:"name"`

	// ReleaseName specifies the exact Helm release name to be used in the cluster.
	// If not set, it defaults to the 'Name' field.
	// Warning: Changing this field triggers a destructive uninstall/install.
	ReleaseName string `json:"releaseName,omitempty" yaml:"releaseName,omitempty"`

	// Namespace is the Kubernetes namespace where the Helm Release workload will be installed.
	// Defaults to 'default' or the active kubeconfig context namespace.
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`

	// -------------------------------------------------------------------------
	// Source Configuration (Polymorphic)
	// -------------------------------------------------------------------------

	// SourceType determines the strategy for artifact acquisition.
	// Valid values: "helm-repo", "git", "oci", "s3".
	// Required.
	SourceType string `json:"sourceType" yaml:"sourceType"`

	// URL is the primary locator for the source.
	// - 'helm-repo': HTTP/S URL of the index.
	// - 'git': HTTP/S or SSH URL of the git repo.
	// - 'oci': Registry URL starting with 'oci://'.
	// - 's3': The generic S3-compatible endpoint.
	URL string `json:"url" yaml:"url"`

	// Interval is the frequency at which the source is reconciled.
	// Must be a valid Go duration string (e.g., "10m", "1h").
	// Defaults to "10m" if unspecified.
	Interval string `json:"interval,omitempty" yaml:"interval,omitempty"`

	// -------------------------------------------------------------------------
	// Helm Repository Specifics
	// -------------------------------------------------------------------------

	// ChartName is the name of the chart to fetch from the Helm repository index.
	// Required when SourceType is "helm-repo".
	ChartName string `json:"chartName,omitempty" yaml:"chartName,omitempty"`

	// Version is the version constraint for the chart.
	// - 'helm-repo': SemVer range (e.g., "^1.0.0").
	// - 'oci': Specific tag or SemVer range.
	// - 'git': Ignored (use GitRef fields).
	// Defaults to "*" (latest) if unspecified.
	Version string `json:"version,omitempty" yaml:"version,omitempty"`

	// -------------------------------------------------------------------------
	// Git Repository Specifics
	// -------------------------------------------------------------------------

	// ChartPath is the relative file path to the chart directory within the source.
	// Required when SourceType is "git" or "s3".
	ChartPath string `json:"chartPath,omitempty" yaml:"chartPath,omitempty"`

	// GitBranch specifies the Git branch to checkout.
	// Used only when SourceType is "git".
	GitBranch string `json:"gitBranch,omitempty" yaml:"gitBranch,omitempty"`

	// GitTag specifies the Git tag to checkout.
	// Takes precedence over GitBranch if set.
	GitTag string `json:"gitTag,omitempty" yaml:"gitTag,omitempty"`

	// GitCommit specifies the exact Git commit SHA to checkout.
	// Takes precedence over GitTag and GitBranch.
	GitCommit string `json:"gitCommit,omitempty" yaml:"gitCommit,omitempty"`

	// GitSemVer specifies a semantic version range to match against Git tags.
	GitSemVer string `json:"gitSemVer,omitempty" yaml:"gitSemVer,omitempty"`

	// IgnorePaths is a list of .gitignore style patterns to exclude from the artifact.
	// Improves reconciliation performance by reducing artifact size.
	IgnorePaths []string `json:"ignorePaths,omitempty" yaml:"ignorePaths,omitempty"`

	// -------------------------------------------------------------------------
	// OCI Repository Specifics
	// -------------------------------------------------------------------------

	// OCIProvider specifies the verification provider for OCI artifacts.
	// Valid values: "cosign", "notation".
	OCIProvider string `json:"ociProvider,omitempty" yaml:"ociProvider,omitempty"`

	// OCILayerMediaType specifies the media type of the layer to extract.
	OCILayerMediaType string `json:"ociLayerMediaType,omitempty" yaml:"ociLayerMediaType,omitempty"`

	// -------------------------------------------------------------------------
	// S3 Bucket Specifics
	// -------------------------------------------------------------------------

	// S3BucketName is the name of the S3/Minio bucket.
	// Required when SourceType is "s3".
	S3BucketName string `json:"s3BucketName,omitempty" yaml:"s3BucketName,omitempty"`

	// S3BucketRegion is the region where the bucket exists.
	// Defaults to "us-east-1" for generic S3 providers.
	S3BucketRegion string `json:"s3BucketRegion,omitempty" yaml:"s3BucketRegion,omitempty"`

	// -------------------------------------------------------------------------
	// Authentication & Security
	// -------------------------------------------------------------------------

	// AuthSecretName is the name of a Kubernetes Secret containing credentials.
	// The secret must reside in the same namespace.
	AuthSecretName string `json:"authSecretName,omitempty" yaml:"authSecretName,omitempty"`

	// PassCredentials allows passing credentials to the HTTP driver when downloading
	// the chart tarball. Critical for private Helm repositories.
	// Defaults to false.
	PassCredentials bool `json:"passCredentials,omitempty" yaml:"passCredentials,omitempty"`

	// InsecureSkipVerify disables TLS verification.
	// Use only for development.
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty"`

	// -------------------------------------------------------------------------
	// Values Configuration & Composition
	// -------------------------------------------------------------------------

	// Values is a map of inline Helm values to apply.
	// These override ValuesFiles and ValueReferences.
	Values map[string]interface{} `json:"values,omitempty" yaml:"values,omitempty"`

	// ValuesFiles is a list of file paths within the Source artifact to use as values.
	ValuesFiles []string `json:"valuesFiles,omitempty" yaml:"valuesFiles,omitempty"`

	// ValueReferences allows referencing values from existing ConfigMaps or Secrets.
	ValueReferences []ValueReference `json:"valueReferences,omitempty" yaml:"valueReferences,omitempty"`

	// -------------------------------------------------------------------------
	// Lifecycle & Remediation (matches original fields but with better docs)
	// -------------------------------------------------------------------------

	// Timeout is the time to wait for any individual Helm operation.
	// Defaults to '5m0s'.
	Timeout string `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// CreateNamespace enables the creation of the target namespace if it does not exist.
	CreateNamespace bool `json:"createNamespace,omitempty" yaml:"createNamespace,omitempty"`

	// ForceUpgrade enables the use of 'helm upgrade --force'.
	// Recreates resources that cannot be patched.
	ForceUpgrade bool `json:"forceUpgrade,omitempty" yaml:"forceUpgrade,omitempty"`

	// DisableHooks prevents Helm hooks (pre-install, post-install) from running.
	DisableHooks bool `json:"disableHooks,omitempty" yaml:"disableHooks,omitempty"`

	// DisableWait determines whether to wait for all resources to be ready.
	// If false (default), the controller waits for resources to be ready.
	// If true, the release is marked successful immediately after manifests are applied.
	DisableWait bool `json:"disableWait,omitempty" yaml:"disableWait,omitempty"`

	// TestEnable triggers the execution of Helm tests after a release.
	TestEnable bool `json:"testEnable,omitempty" yaml:"testEnable,omitempty"`
}

// ValueReference represents a reference to a ConfigMap or Secret containing values.
type ValueReference struct {
	// Kind is the type of resource (ConfigMap or Secret).
	// Required.
	Kind string `json:"kind" yaml:"kind"`

	// Name is the name of the resource.
	// Required.
	Name string `json:"name" yaml:"name"`

	// ValuesKey is the specific key in the resource data to use.
	// If empty, all keys in the resource are merged.
	ValuesKey string `json:"valuesKey,omitempty" yaml:"valuesKey,omitempty"`

	// TargetPath is the YAML dot-notation path to merge the values into.
	// Example: "server.config".
	TargetPath string `json:"targetPath,omitempty" yaml:"targetPath,omitempty"`

	// Optional indicates if the referenced resource is required.
	Optional bool `json:"optional,omitempty" yaml:"optional,omitempty"`
}

// runMCPServer starts the testenv-helm-install MCP server with stdio transport.
func runMCPServer() error {
	server := mcpserver.New("testenv-helm-install", Version)

	config := engineframework.TestEnvSubengineConfig{
		Name:       "testenv-helm-install",
		Version:    Version,
		CreateFunc: installHelmCharts,
		DeleteFunc: uninstallHelmCharts,
	}

	if err := engineframework.RegisterTestEnvSubengineTools(server, config); err != nil {
		return err
	}

	return server.RunDefault()
}

// installHelmCharts implements the CreateFunc for installing Helm charts.
func installHelmCharts(ctx context.Context, input engineframework.CreateInput) (*engineframework.TestEnvArtifact, error) {
	log.Printf("Installing Helm charts: testID=%s, stage=%s", input.TestID, input.Stage)

	// Parse charts from spec
	charts, err := parseChartsFromSpec(input.Spec)
	if err != nil {
		// If spec.charts is not found or empty, skip gracefully
		log.Printf("No charts specified, skipping helm installation")

		// Return success with empty metadata
		return &engineframework.TestEnvArtifact{
			TestID:           input.TestID,
			Files:            map[string]string{},
			Metadata:         map[string]string{"testenv-helm-install.chartCount": "0"},
			ManagedResources: []string{},
		}, nil
	}

	if len(charts) == 0 {
		log.Printf("Empty charts list, skipping helm installation")

		// Return success with empty metadata
		return &engineframework.TestEnvArtifact{
			TestID:           input.TestID,
			Files:            map[string]string{},
			Metadata:         map[string]string{"testenv-helm-install.chartCount": "0"},
			ManagedResources: []string{},
		}, nil
	}

	// Find kubeconfig from tmpDir or metadata
	kubeconfigPath, err := findKubeconfig(input.TmpDir, input.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to find kubeconfig: %w", err)
	}

	log.Printf("Using kubeconfig: %s", kubeconfigPath)

	// Install each chart
	installedCharts := []string{}
	metadata := map[string]string{}

	for i, chart := range charts {
		// Validate required fields
		if chart.SourceType == "" {
			return nil, fmt.Errorf("chart %s: sourceType is required", chart.Name)
		}
		if chart.SourceType == "helm-repo" {
			if chart.URL == "" {
				return nil, fmt.Errorf("chart %s: url is required for helm-repo source", chart.Name)
			}
			if chart.ChartName == "" {
				return nil, fmt.Errorf("chart %s: chartName is required for helm-repo source", chart.Name)
			}
		}

		releaseName := chart.ReleaseName
		if releaseName == "" {
			releaseName = chart.Name
		}

		log.Printf("Installing chart %d/%d: %s (release: %s)", i+1, len(charts), chart.Name, releaseName)

		// Add helm repo if using helm-repo source type
		if chart.SourceType == "helm-repo" && chart.URL != "" {
			// Extract repo name from URL for chart reference
			repoName := extractRepoNameFromURL(chart.URL)
			if err := addHelmRepo(repoName, chart.URL); err != nil {
				return nil, fmt.Errorf("failed to add helm repo %s: %w", chart.URL, err)
			}
		}

		// Install the chart
		if err := installChart(chart, kubeconfigPath); err != nil {
			return nil, fmt.Errorf("failed to install chart %s: %w", chart.Name, err)
		}

		installedCharts = append(installedCharts, releaseName)

		// Store chart info in metadata
		prefix := fmt.Sprintf("testenv-helm-install.chart.%d", i)
		metadata[prefix+".name"] = chart.Name
		metadata[prefix+".releaseName"] = releaseName
		if chart.Namespace != "" {
			metadata[prefix+".namespace"] = chart.Namespace
		}
	}

	// Store count of installed charts
	metadata["testenv-helm-install.chartCount"] = fmt.Sprintf("%d", len(installedCharts))

	// Prepare files map (no files produced by helm install)
	files := map[string]string{}

	// Prepare managed resources (for cleanup)
	managedResources := []string{}

	// Return artifact
	return &engineframework.TestEnvArtifact{
		TestID:           input.TestID,
		Files:            files,
		Metadata:         metadata,
		ManagedResources: managedResources,
	}, nil
}

// uninstallHelmCharts implements the DeleteFunc for uninstalling Helm charts.
func uninstallHelmCharts(ctx context.Context, input engineframework.DeleteInput) error {
	log.Printf("Uninstalling Helm charts: testID=%s", input.TestID)

	// Extract chart count from metadata
	chartCountStr, ok := input.Metadata["testenv-helm-install.chartCount"]
	if !ok {
		// No charts to uninstall
		log.Printf("No charts found in metadata, skipping uninstall")
		return nil
	}

	var chartCount int
	if _, err := fmt.Sscanf(chartCountStr, "%d", &chartCount); err != nil {
		log.Printf("Warning: invalid chartCount in metadata: %v", err)
		return nil
	}

	// Find kubeconfig from metadata (use testenv-kind's kubeconfig)
	kubeconfigPath, ok := input.Metadata["testenv-kind.kubeconfigPath"]
	if !ok {
		log.Printf("Warning: kubeconfig not found in metadata, skipping helm uninstall")
		return nil
	}

	// Uninstall each chart in reverse order (best effort)
	for i := chartCount - 1; i >= 0; i-- {
		prefix := fmt.Sprintf("testenv-helm-install.chart.%d", i)
		releaseName := input.Metadata[prefix+".releaseName"]
		namespace := input.Metadata[prefix+".namespace"]

		if releaseName == "" {
			log.Printf("Warning: chart %d missing release name, skipping", i)
			continue
		}

		log.Printf("Uninstalling chart %d/%d: %s", chartCount-i, chartCount, releaseName)

		// Uninstall the chart (best effort)
		if err := uninstallChart(releaseName, namespace, kubeconfigPath); err != nil {
			log.Printf("Warning: failed to uninstall chart %s: %v", releaseName, err)
			// Continue with other charts (best effort cleanup)
		}
	}

	return nil
}

// parseChartsFromSpec extracts chart specifications from the spec map
func parseChartsFromSpec(spec map[string]any) ([]ChartSpec, error) {
	if spec == nil {
		return nil, fmt.Errorf("spec is nil")
	}

	chartsRaw, ok := spec["charts"]
	if !ok {
		return nil, fmt.Errorf("spec.charts not found")
	}

	// Marshal and unmarshal to convert to ChartSpec slice
	chartsJSON, err := json.Marshal(chartsRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal charts: %w", err)
	}

	var charts []ChartSpec
	if err := json.Unmarshal(chartsJSON, &charts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal charts: %w", err)
	}

	return charts, nil
}

// findKubeconfig locates the kubeconfig file from tmpDir or metadata
func findKubeconfig(tmpDir string, metadata map[string]string) (string, error) {
	// First try to get from metadata (testenv-kind provides this)
	if path, ok := metadata["testenv-kind.kubeconfigPath"]; ok && path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Try common locations in tmpDir
	commonNames := []string{"kubeconfig", "kubeconfig.yaml", ".kube/config"}
	for _, name := range commonNames {
		path := filepath.Join(tmpDir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("kubeconfig not found in tmpDir or metadata")
}

// extractRepoNameFromURL extracts a repo name from URL
// e.g., "https://stefanprodan.github.io/podinfo" -> "podinfo"
func extractRepoNameFromURL(url string) string {
	// Remove trailing slash if present
	url = strings.TrimSuffix(url, "/")

	// Extract last path segment
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "repo"
}

// addHelmRepo adds a helm repository
func addHelmRepo(name, repoURL string) error {
	log.Printf("Adding helm repo: %s -> %s", name, repoURL)

	// Add timeout for repo operations (2 minutes should be plenty)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "helm", "repo", "add", name, repoURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("helm repo add timed out after 2 minutes")
		}
		return fmt.Errorf("helm repo add failed: %w, output: %s", err, string(output))
	}

	// Update repo with same context
	cmd = exec.CommandContext(ctx, "helm", "repo", "update")
	output, err = cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("helm repo update timed out after 2 minutes")
		}
		return fmt.Errorf("helm repo update failed: %w, output: %s", err, string(output))
	}

	return nil
}

// installChart installs a helm chart using the ChartSpec
func installChart(chart ChartSpec, kubeconfigPath string) error {
	releaseName := chart.ReleaseName
	if releaseName == "" {
		releaseName = chart.Name
	}

	// For helm-repo source type, construct chart reference as repoName/chartName
	var chartRef string
	if chart.SourceType == "helm-repo" {
		if chart.ChartName == "" {
			return fmt.Errorf("chartName is required when sourceType is helm-repo")
		}
		// Extract repo name from URL for chart reference
		repoName := extractRepoNameFromURL(chart.URL)
		chartRef = fmt.Sprintf("%s/%s", repoName, chart.ChartName)
	} else {
		return fmt.Errorf("sourceType %s is not yet implemented", chart.SourceType)
	}

	args := []string{
		"install",
		releaseName,
		chartRef,
		"--kubeconfig", kubeconfigPath,
	}

	// Add version if specified
	if chart.Version != "" {
		args = append(args, "--version", chart.Version)
	}

	// Add namespace handling
	if chart.Namespace != "" {
		args = append(args, "--namespace", chart.Namespace)
		if chart.CreateNamespace {
			args = append(args, "--create-namespace")
		}
	}

	// Add timeout (default to 5m if not specified)
	timeout := chart.Timeout
	if timeout == "" {
		timeout = "5m"
	}
	args = append(args, "--timeout", timeout)

	// Add wait behavior (wait by default unless DisableWait is true)
	if !chart.DisableWait {
		args = append(args, "--wait")
	}

	// Add force upgrade if specified
	if chart.ForceUpgrade {
		args = append(args, "--force")
	}

	// Add disable hooks if specified
	if chart.DisableHooks {
		args = append(args, "--no-hooks")
	}

	// Add values from the Values map
	for key, value := range chart.Values {
		args = append(args, "--set", fmt.Sprintf("%s=%v", key, value))
	}

	// Add values files if specified
	for _, valuesFile := range chart.ValuesFiles {
		args = append(args, "--values", valuesFile)
	}

	// Log warning if ValueReferences is used (not yet implemented)
	if len(chart.ValueReferences) > 0 {
		log.Printf("Warning: ValueReferences specified but not yet implemented, ignoring")
	}

	log.Printf("Running: helm %v", args)

	// Calculate context timeout based on helm timeout plus buffer
	// Parse timeout or default to 5 minutes
	helmTimeout, err := time.ParseDuration(timeout)
	if err != nil {
		log.Printf("Warning: invalid timeout %s, defaulting to 5m", timeout)
		helmTimeout = 5 * time.Minute
	}
	contextTimeout := helmTimeout + 1*time.Minute // Add 1 minute buffer

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "helm", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("helm install timed out after %v", contextTimeout)
		}
		return fmt.Errorf("helm install failed: %w, output: %s", err, string(output))
	}

	log.Printf("Chart installed successfully: %s", releaseName)

	// Run helm tests if enabled
	if chart.TestEnable {
		log.Printf("Running helm tests for release: %s", releaseName)
		if err := runHelmTest(releaseName, chart.Namespace, kubeconfigPath, timeout); err != nil {
			log.Printf("Warning: helm test failed for %s: %v", releaseName, err)
			// Don't fail the install if tests fail, just log warning
		}
	}

	return nil
}

// runHelmTest runs helm test for a release
func runHelmTest(releaseName, namespace, kubeconfigPath, timeout string) error {
	args := []string{
		"test",
		releaseName,
		"--kubeconfig", kubeconfigPath,
		"--timeout", timeout,
	}

	if namespace != "" {
		args = append(args, "--namespace", namespace)
	}

	log.Printf("Running: helm %v", args)

	// Parse timeout for context
	helmTimeout, err := time.ParseDuration(timeout)
	if err != nil {
		helmTimeout = 5 * time.Minute
	}
	contextTimeout := helmTimeout + 1*time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "helm", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("helm test timed out after %v", contextTimeout)
		}
		return fmt.Errorf("helm test failed: %w, output: %s", err, string(output))
	}

	log.Printf("Helm tests passed for: %s", releaseName)
	return nil
}

// uninstallChart uninstalls a helm chart
func uninstallChart(releaseName, namespace, kubeconfigPath string) error {
	args := []string{
		"uninstall",
		releaseName,
		"--kubeconfig", kubeconfigPath,
		"--timeout", "2m", // Helm-level timeout
	}

	if namespace != "" {
		args = append(args, "--namespace", namespace)
	}

	log.Printf("Running: helm %v", args)

	// Add context timeout (3 minutes to allow helm's internal 2m timeout plus buffer)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "helm", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("helm uninstall timed out after 3 minutes")
		}
		return fmt.Errorf("helm uninstall failed: %w, output: %s", err, string(output))
	}

	log.Printf("Chart uninstalled successfully: %s", releaseName)
	return nil
}
