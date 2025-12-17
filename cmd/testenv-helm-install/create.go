// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"gopkg.in/yaml.v3"
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
	// Valid values: "helm-repo", "git", "oci", "s3", "local".
	// Required.
	SourceType string `json:"sourceType" yaml:"sourceType"`

	// URL is the primary locator for the source.
	// - 'helm-repo': HTTP/S URL of the index.
	// - 'git': HTTP/S or SSH URL of the git repo.
	// - 'oci': Registry URL starting with 'oci://'.
	// - 's3': The generic S3-compatible endpoint.
	URL string `json:"url" yaml:"url"`

	// Path is the filesystem path to a local chart directory.
	// Required when SourceType is "local".
	Path string `json:"path,omitempty" yaml:"path,omitempty"`

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

// resolveChartPath resolves a chart path relative to RootDir if applicable.
// Returns the resolved absolute path, or the original path if no resolution is needed.
// Returns an error if the resolved path does not exist.
func resolveChartPath(chart ChartSpec, rootDir string) (string, error) {
	// Only resolve local charts with non-empty paths
	if chart.SourceType != "local" || chart.Path == "" {
		return chart.Path, nil
	}

	resolvedPath := chart.Path

	// Resolve relative paths using RootDir
	if rootDir != "" && !filepath.IsAbs(chart.Path) {
		resolvedPath = filepath.Join(rootDir, chart.Path)
		log.Printf("Resolved local chart path: %s", resolvedPath)
	}

	// Validate resolved path exists (fail-fast)
	if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
		return "", fmt.Errorf("local chart not found: %s", resolvedPath)
	}

	return resolvedPath, nil
}

// Create implements the CreateFunc for installing Helm charts.
// The spec parameter is available but charts are parsed from input.Spec via parseChartsFromSpec.
func Create(ctx context.Context, input engineframework.CreateInput, _ *Spec) (*engineframework.TestEnvArtifact, error) {
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

	// Resolve relative paths for local charts using RootDir
	// This MUST happen before installChart() since installChart has no access to CreateInput
	for i := range charts {
		if charts[i].SourceType == "local" && charts[i].Path != "" {
			resolvedPath, err := resolveChartPath(charts[i], input.RootDir)
			if err != nil {
				return nil, err
			}
			charts[i].Path = resolvedPath
		}
	}

	// Get kubeconfig path from environment (primary source, from testenv-kind)
	// Fallback to findKubeconfig for backward compatibility
	kubeconfigPath := ""
	if envKubeconfig, ok := input.Env["KUBECONFIG"]; ok && envKubeconfig != "" {
		kubeconfigPath = envKubeconfig
		log.Printf("Using KUBECONFIG from environment: %s", kubeconfigPath)
	} else {
		// Fallback to legacy behavior (search tmpDir and metadata)
		var err error
		kubeconfigPath, err = findKubeconfig(input.TmpDir, input.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to find kubeconfig: %w", err)
		}
		log.Printf("Using kubeconfig from legacy sources (tmpDir/metadata): %s", kubeconfigPath)
	}

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
		if chart.SourceType == "local" {
			if chart.Path == "" {
				return nil, fmt.Errorf("chart %s: path is required for local source", chart.Name)
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

// Delete implements the DeleteFunc for uninstalling Helm charts.
func Delete(ctx context.Context, input engineframework.DeleteInput, _ *Spec) error {
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

	// Check if kubeconfig file exists - if not, this is a bug in the cleanup order
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return fmt.Errorf("kubeconfig file does not exist at %s - cluster was deleted before helm uninstall (cleanup order bug)", kubeconfigPath)
	}

	// Uninstall each chart in reverse order
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

// resolveGitRef determines which Git reference to use based on precedence.
// Precedence: Commit > Tag > SemVer > Branch
// Returns: ref (the actual git ref to checkout), refType ("commit", "tag", "semver", or "branch"), error
func resolveGitRef(chart ChartSpec) (ref string, refType string, err error) {
	// Commit takes precedence over all
	if chart.GitCommit != "" {
		// Validate commit SHA (should be at least 7 characters and hexadecimal)
		if len(chart.GitCommit) < 7 {
			return "", "", fmt.Errorf("invalid git commit: too short (minimum 7 characters)")
		}
		for _, c := range chart.GitCommit {
			isDigit := c >= '0' && c <= '9'
			isLowerHex := c >= 'a' && c <= 'f'
			isUpperHex := c >= 'A' && c <= 'F'
			if !isDigit && !isLowerHex && !isUpperHex {
				return "", "", fmt.Errorf("invalid git commit: contains non-hexadecimal character: %c", c)
			}
		}
		return chart.GitCommit, "commit", nil
	}

	// Tag takes precedence over SemVer and Branch
	if chart.GitTag != "" {
		return chart.GitTag, "tag", nil
	}

	// SemVer takes precedence over Branch
	if chart.GitSemVer != "" {
		return chart.GitSemVer, "semver", nil
	}

	// Branch is the lowest precedence
	if chart.GitBranch != "" {
		return chart.GitBranch, "branch", nil
	}

	// No git reference specified
	return "", "", fmt.Errorf("no git reference specified: one of GitCommit, GitTag, GitSemVer, or GitBranch is required")
}

// resolveSemVerTag finds the latest Git tag matching the SemVer constraint.
// It lists all tags in the repository, filters for valid SemVer tags,
// and returns the latest tag that satisfies the constraint.
func resolveSemVerTag(repoPath string, semverConstraint string) (string, error) {
	// Parse the semver constraint
	constraint, err := semver.NewConstraint(semverConstraint)
	if err != nil {
		return "", fmt.Errorf("invalid semver constraint %q: %w", semverConstraint, err)
	}

	// List all tags in the repository
	cmd := exec.Command("git", "tag", "-l")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to list git tags: %w, output: %s", err, string(output))
	}

	// Parse tags and find matching versions
	tagLines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var matchingVersions []*semver.Version
	var matchingTags []string

	for _, tagLine := range tagLines {
		tag := strings.TrimSpace(tagLine)
		if tag == "" {
			continue
		}

		// Try to parse as semver (strip 'v' prefix if present)
		versionStr := strings.TrimPrefix(tag, "v")
		version, err := semver.NewVersion(versionStr)
		if err != nil {
			// Skip non-semver tags
			continue
		}

		// Check if version matches constraint
		if constraint.Check(version) {
			matchingVersions = append(matchingVersions, version)
			matchingTags = append(matchingTags, tag)
		}
	}

	if len(matchingVersions) == 0 {
		return "", fmt.Errorf("no git tags match semver constraint %q", semverConstraint)
	}

	// Find the latest version
	latestIndex := 0
	for i := 1; i < len(matchingVersions); i++ {
		if matchingVersions[i].GreaterThan(matchingVersions[latestIndex]) {
			latestIndex = i
		}
	}

	return matchingTags[latestIndex], nil
}

// applyIgnorePatterns removes files matching ignore patterns from cloned repo.
// This is a placeholder for future optimization.
// Currently logs a warning if patterns are provided but does not apply them.
// nolint:unused // Placeholder for future feature implementation
func applyIgnorePatterns(repoPath string, ignorePatterns []string) error {
	if len(ignorePatterns) > 0 {
		log.Printf("Warning: IgnorePaths specified but not yet fully implemented (optimization placeholder)")
		log.Printf("Patterns specified: %v", ignorePatterns)
		// TODO: Implement .gitignore-style pattern matching to remove files
		// This would reduce artifact size and improve performance
	}
	return nil
}

// validateGitSource validates required fields for Git source type.
func validateGitSource(chart ChartSpec) error {
	// Validate URL
	if chart.URL == "" {
		return fmt.Errorf("url is required for git source type")
	}

	// Validate URL format (must be http://, https://, or git@)
	if !strings.HasPrefix(chart.URL, "http://") &&
		!strings.HasPrefix(chart.URL, "https://") &&
		!strings.HasPrefix(chart.URL, "git@") {
		return fmt.Errorf("invalid url format: must start with http://, https://, or git@")
	}

	// Validate ChartPath
	if chart.ChartPath == "" {
		return fmt.Errorf("chartPath is required for git source type")
	}

	// ChartPath must be relative (no leading /)
	if strings.HasPrefix(chart.ChartPath, "/") {
		return fmt.Errorf("chartPath must be relative (no leading /)")
	}

	// Validate at least one git reference is specified
	if chart.GitCommit == "" && chart.GitTag == "" && chart.GitSemVer == "" && chart.GitBranch == "" {
		return fmt.Errorf("at least one git reference (GitCommit, GitTag, GitSemVer, or GitBranch) is required")
	}

	return nil
}

// buildGitCloneCommand builds the git clone command arguments based on the ref type.
// For branches and tags, use shallow clone for performance.
// For commits and semver, use full clone (semver needs all tags, commit needs full history).
func buildGitCloneCommand(url, destDir, ref, refType string) []string {
	args := []string{"clone"}

	// Use shallow clone for branches and tags
	if refType == "branch" || refType == "tag" {
		args = append(args, "--branch", ref, "--depth", "1")
	}

	// For commit and semver, do full clone
	// (commit needs full history to checkout specific SHA, semver needs all tags)

	args = append(args, url, destDir)
	return args
}

// cloneGitRepository clones a Git repository and checks out the specified ref.
// Returns the full path to the chart directory and a cleanup function.
// The cleanup function must be called to remove the cloned repository.
func cloneGitRepository(chart ChartSpec, destDir string) (chartPath string, cleanup func(), err error) {
	// Validate required fields
	if chart.URL == "" {
		return "", nil, fmt.Errorf("URL is required for git source type")
	}
	if chart.ChartPath == "" {
		return "", nil, fmt.Errorf("ChartPath is required for git source type")
	}

	// Resolve git reference
	ref, refType, err := resolveGitRef(chart)
	if err != nil {
		return "", nil, err
	}

	// Create unique clone directory
	cloneDir := filepath.Join(destDir, "git-clone")
	if err := os.MkdirAll(cloneDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("failed to create clone directory: %w", err)
	}

	cleanup = func() {
		if err := os.RemoveAll(cloneDir); err != nil {
			log.Printf("Warning: failed to remove git clone directory %s: %v", cloneDir, err)
		}
	}

	// Add timeout for clone operation (5 minutes)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Build clone command
	args := buildGitCloneCommand(chart.URL, cloneDir, ref, refType)
	cmd := exec.CommandContext(ctx, "git", args...)

	log.Printf("Cloning git repository: %s (ref: %s, type: %s)", chart.URL, ref, refType)
	startTime := time.Now()

	output, err := cmd.CombinedOutput()
	if err != nil {
		cleanup()
		if ctx.Err() == context.DeadlineExceeded {
			return "", nil, fmt.Errorf("git clone timed out after 5 minutes")
		}
		return "", nil, fmt.Errorf("failed to clone git repository %s: %w, output: %s", chart.URL, err, string(output))
	}

	cloneDuration := time.Since(startTime)
	if cloneDuration > 30*time.Second {
		log.Printf("Warning: git clone took %v (>30s)", cloneDuration)
	}

	// For commits and semver, need to checkout specific ref
	switch refType {
	case "commit":
		// Checkout specific commit
		checkoutCmd := exec.CommandContext(ctx, "git", "checkout", ref)
		checkoutCmd.Dir = cloneDir
		output, err := checkoutCmd.CombinedOutput()
		if err != nil {
			cleanup()
			return "", nil, fmt.Errorf("failed to checkout commit %s: %w, output: %s", ref, err, string(output))
		}
		log.Printf("Checked out commit: %s", ref)
	case "semver":
		// Resolve semver to specific tag
		tag, err := resolveSemVerTag(cloneDir, ref)
		if err != nil {
			cleanup()
			return "", nil, fmt.Errorf("failed to resolve semver %s: %w", ref, err)
		}
		// Checkout the resolved tag
		checkoutCmd := exec.CommandContext(ctx, "git", "checkout", tag)
		checkoutCmd.Dir = cloneDir
		output, err := checkoutCmd.CombinedOutput()
		if err != nil {
			cleanup()
			return "", nil, fmt.Errorf("failed to checkout tag %s: %w, output: %s", tag, err, string(output))
		}
		log.Printf("Resolved semver %s to tag %s and checked out", ref, tag)
	}

	// Construct chart path
	chartPath = filepath.Join(cloneDir, chart.ChartPath)

	// Verify Chart.yaml exists
	chartYamlPath := filepath.Join(chartPath, "Chart.yaml")
	if _, err := os.Stat(chartYamlPath); os.IsNotExist(err) {
		cleanup()
		return "", nil, fmt.Errorf("chart.yaml not found at %s", chartPath)
	}

	log.Printf("Successfully cloned and validated chart at: %s", chartPath)
	return chartPath, cleanup, nil
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

// parseYAMLValue parses a YAML string and returns the parsed value.
// If the string is not valid YAML, returns an error.
// Simple strings that are not YAML structures are returned as-is.
func parseYAMLValue(yamlStr string) (interface{}, error) {
	// Try to parse as YAML
	var result interface{}
	err := yaml.Unmarshal([]byte(yamlStr), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML value: %w", err)
	}

	// yaml.Unmarshal returns the parsed structure
	// Simple strings are returned as strings
	return result, nil
}

// extractValueByKey extracts a value from the data map by key.
// If valuesKey is empty, returns all data as a map[string]interface{}.
// If valuesKey is specified, extracts that specific key and parses it as YAML if needed.
func extractValueByKey(data map[string]string, valuesKey string) (interface{}, error) {
	// If no specific key requested, return all data as map
	if valuesKey == "" {
		result := make(map[string]interface{})
		for k, v := range data {
			result[k] = v
		}
		return result, nil
	}

	// Extract specific key
	value, ok := data[valuesKey]
	if !ok {
		return nil, fmt.Errorf("key %q not found in data", valuesKey)
	}

	// Try to parse as YAML
	parsed, err := parseYAMLValue(value)
	if err != nil {
		// If parsing fails, return as string
		return value, nil
	}

	return parsed, nil
}

// resolveValueReference fetches and processes a single ValueReference.
// Returns the resolved values to be merged.
// If the reference is optional and the resource is not found, returns nil without error.
func resolveValueReference(kubeconfigPath, namespace string, ref ValueReference) (interface{}, error) {
	// Determine which fetch function to use
	var data map[string]string
	var err error

	switch strings.ToLower(ref.Kind) {
	case "configmap":
		data, err = fetchConfigMap(kubeconfigPath, namespace, ref.Name)
	case "secret":
		data, err = fetchSecret(kubeconfigPath, namespace, ref.Name)
	default:
		return nil, fmt.Errorf("unsupported Kind %q: must be ConfigMap or Secret", ref.Kind)
	}

	// Handle errors
	if err != nil {
		// Check if resource not found
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "NotFound") {
			if ref.Optional {
				log.Printf("Info: optional %s %s/%s not found, skipping", ref.Kind, namespace, ref.Name)
				return nil, nil
			}
		}
		return nil, err
	}

	// Extract value by key
	return extractValueByKey(data, ref.ValuesKey)
}

// mergeValuesAtPath merges newValues into baseValues at the specified dot-notation path.
// If targetPath is empty, merges at the root level.
// Creates intermediate keys as needed.
// If the target already exists and both old and new values are maps, merges recursively.
// Otherwise, the new value replaces the old value.
func mergeValuesAtPath(baseValues map[string]interface{}, newValues interface{}, targetPath string) error {
	// If targetPath is empty, merge at root level
	if targetPath == "" {
		newMap, ok := newValues.(map[string]interface{})
		if !ok {
			return fmt.Errorf("cannot merge non-map value at root level (got type %T)", newValues)
		}
		mergeMap(baseValues, newMap)
		return nil
	}

	// Split path by dots
	pathParts := strings.Split(targetPath, ".")

	// Navigate/create path to the parent of the target
	current := baseValues
	for i := 0; i < len(pathParts)-1; i++ {
		key := pathParts[i]

		// Check if key exists
		if existing, ok := current[key]; ok {
			// Key exists, verify it's a map
			existingMap, ok := existing.(map[string]interface{})
			if !ok {
				return fmt.Errorf("type conflict at path %q: existing value is %T, cannot navigate deeper",
					strings.Join(pathParts[:i+1], "."), existing)
			}
			current = existingMap
		} else {
			// Key doesn't exist, create it as a map
			newMap := make(map[string]interface{})
			current[key] = newMap
			current = newMap
		}
	}

	// Set or merge the value at the final key
	finalKey := pathParts[len(pathParts)-1]

	// Check if final key exists
	if existing, ok := current[finalKey]; ok {
		// If both are maps, merge recursively
		existingMap, existingIsMap := existing.(map[string]interface{})
		newMap, newIsMap := newValues.(map[string]interface{})

		if existingIsMap && newIsMap {
			mergeMap(existingMap, newMap)
			return nil
		}
	}

	// Either key doesn't exist, or we're replacing the value
	current[finalKey] = newValues
	return nil
}

// mergeMap recursively merges src into dst.
// For keys that exist in both maps:
// - If both values are maps, merge recursively
// - Otherwise, src value overwrites dst value
func mergeMap(dst, src map[string]interface{}) {
	for key, srcValue := range src {
		if dstValue, ok := dst[key]; ok {
			// Key exists in both
			dstMap, dstIsMap := dstValue.(map[string]interface{})
			srcMap, srcIsMap := srcValue.(map[string]interface{})

			if dstIsMap && srcIsMap {
				// Both are maps, merge recursively
				mergeMap(dstMap, srcMap)
			} else {
				// Replace with src value
				dst[key] = srcValue
			}
		} else {
			// Key only exists in src, copy it
			dst[key] = srcValue
		}
	}
}

// installChart installs a helm chart using the ChartSpec
func installChart(chart ChartSpec, kubeconfigPath string) error {
	releaseName := chart.ReleaseName
	if releaseName == "" {
		releaseName = chart.Name
	}

	// Handle different source types
	var chartRef string
	var cleanup func()

	switch chart.SourceType {
	case "helm-repo":
		if chart.ChartName == "" {
			return fmt.Errorf("chartName is required when sourceType is helm-repo")
		}
		// Extract repo name from URL for chart reference
		repoName := extractRepoNameFromURL(chart.URL)
		chartRef = fmt.Sprintf("%s/%s", repoName, chart.ChartName)

	case "local":
		if chart.Path == "" {
			return fmt.Errorf("path is required when sourceType is local")
		}
		// For local charts, use the path directly
		chartRef = chart.Path

	case "git":
		// Validate Git source
		if err := validateGitSource(chart); err != nil {
			return fmt.Errorf("invalid git source: %w", err)
		}

		// Create temporary directory for Git clone
		tmpDir, err := os.MkdirTemp("", "helm-git-*")
		if err != nil {
			return fmt.Errorf("failed to create temp dir: %w", err)
		}

		// Clone repository
		chartPath, cleanupFunc, err := cloneGitRepository(chart, tmpDir)
		if err != nil {
			_ = os.RemoveAll(tmpDir)
			return fmt.Errorf("failed to clone git repository: %w", err)
		}
		cleanup = func() {
			cleanupFunc()
			_ = os.RemoveAll(tmpDir)
		}
		defer cleanup()

		chartRef = chartPath
		log.Printf("Using git chart at: %s", chartRef)

	case "oci":
		// Validate OCI source
		if err := validateOCISource(chart); err != nil {
			return fmt.Errorf("invalid oci source: %w", err)
		}

		// Setup OCI authentication if needed
		authCleanup, err := setupOCIAuth(kubeconfigPath, chart)
		if err != nil {
			return fmt.Errorf("failed to setup OCI auth: %w", err)
		}
		defer authCleanup()

		// Verify OCI signature if OCIProvider is set (optional)
		if err := verifyOCISignature(chart); err != nil {
			return fmt.Errorf("failed to verify OCI signature: %w", err)
		}

		// For OCI, helm can install directly from oci:// URL
		// Format: oci://registry/repository/chart
		chartRef = chart.URL
		log.Printf("Using OCI chart: %s", chartRef)

	case "s3":
		// Validate S3 source
		if err := validateS3Source(chart); err != nil {
			return fmt.Errorf("invalid s3 source: %w", err)
		}

		// Setup S3 client with authentication
		s3Client, err := setupS3Auth(kubeconfigPath, chart)
		if err != nil {
			return fmt.Errorf("failed to setup S3 auth: %w", err)
		}

		// Create temporary directory for S3 download
		tmpDir, err := os.MkdirTemp("", "helm-s3-*")
		if err != nil {
			return fmt.Errorf("failed to create temp dir for S3 download: %w", err)
		}
		cleanup = func() {
			_ = os.RemoveAll(tmpDir)
		}
		defer cleanup()

		// Download chart from S3
		chartPath, err := downloadFromS3(s3Client, chart, tmpDir)
		if err != nil {
			return fmt.Errorf("failed to download chart from S3: %w", err)
		}

		chartRef = chartPath
		log.Printf("Using S3 chart at: %s", chartRef)

	default:
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

	// Compose values from multiple sources
	// Priority (lowest to highest): ValuesFiles < ValueReferences < inline Values
	composedValues := make(map[string]interface{})

	// Note: ValuesFiles are handled by helm directly, not merged here
	// They have the lowest precedence and are passed via --values flag

	// Process ValueReferences (medium precedence)
	for _, ref := range chart.ValueReferences {
		// Use default namespace if chart namespace is not set
		namespace := chart.Namespace
		if namespace == "" {
			namespace = "default"
		}
		refValues, err := resolveValueReference(kubeconfigPath, namespace, ref)
		if err != nil {
			return fmt.Errorf("failed to resolve ValueReference %s/%s: %w", ref.Kind, ref.Name, err)
		}

		// If refValues is nil (optional reference not found), skip
		if refValues != nil {
			if ref.TargetPath != "" {
				// Merge at specific path
				err = mergeValuesAtPath(composedValues, refValues, ref.TargetPath)
			} else {
				// Merge at root level
				refMap, ok := refValues.(map[string]interface{})
				if !ok {
					return fmt.Errorf("ValueReference %s/%s returned non-map value at root level (type %T)", ref.Kind, ref.Name, refValues)
				}
				mergeMap(composedValues, refMap)
			}
			if err != nil {
				return fmt.Errorf("failed to merge values from %s/%s: %w", ref.Kind, ref.Name, err)
			}
		}
	}

	// Apply inline Values (highest precedence)
	for key, value := range chart.Values {
		composedValues[key] = value
	}

	// If we have composed values from ValueReferences or inline Values, write to temp file
	var valuesTempFile string
	if len(composedValues) > 0 {
		// Create temp file for values
		tmpFile, err := os.CreateTemp("", "helm-values-*.yaml")
		if err != nil {
			return fmt.Errorf("failed to create temp values file: %w", err)
		}
		valuesTempFile = tmpFile.Name()
		defer func() {
			if err := os.Remove(valuesTempFile); err != nil {
				log.Printf("Warning: failed to remove temp values file %s: %v", valuesTempFile, err)
			}
		}()

		// Marshal values to YAML
		valuesYAML, err := yaml.Marshal(composedValues)
		if err != nil {
			return fmt.Errorf("failed to marshal values to YAML: %w", err)
		}

		// Write to temp file
		if _, err := tmpFile.Write(valuesYAML); err != nil {
			if closeErr := tmpFile.Close(); closeErr != nil {
				log.Printf("Warning: failed to close temp file: %v", closeErr)
			}
			return fmt.Errorf("failed to write values to temp file: %w", err)
		}
		if err := tmpFile.Close(); err != nil {
			log.Printf("Warning: failed to close temp file: %v", err)
		}

		log.Printf("Composed values from %d ValueReferences and inline values, wrote to: %s", len(chart.ValueReferences), valuesTempFile)
	}

	// Add values files if specified (lowest precedence)
	for _, valuesFile := range chart.ValuesFiles {
		args = append(args, "--values", valuesFile)
	}

	// Add composed values file (medium precedence - after ValuesFiles)
	if valuesTempFile != "" {
		args = append(args, "--values", valuesTempFile)
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

// parseOCIReference parses an OCI URL into components.
// OCI URL format: oci://REGISTRY/REPOSITORY/CHART[:TAG][@DIGEST]
// Examples:
//   - oci://ghcr.io/stefanprodan/charts/podinfo:6.0.0
//   - oci://ghcr.io/stefanprodan/charts/podinfo@sha256:abc123
//   - oci://docker.io/myuser/mychart (defaults to :latest)
//
// Returns: registry, repository, chart, tag, digest, error
// If digest is present, tag will be empty (digest takes precedence).
// If neither tag nor digest is present, tag defaults to "latest".
func parseOCIReference(ociURL string) (registry, repository, chart, tag, digest string, err error) {
	// Validate OCI prefix
	if ociURL == "" {
		return "", "", "", "", "", fmt.Errorf("empty OCI URL")
	}
	if !strings.HasPrefix(ociURL, "oci://") {
		return "", "", "", "", "", fmt.Errorf("OCI URL must start with oci://")
	}

	// Remove oci:// prefix
	path := strings.TrimPrefix(ociURL, "oci://")
	if path == "" {
		return "", "", "", "", "", fmt.Errorf("invalid OCI URL: no path after oci://")
	}

	// Check for digest (@sha256:...) - do this first before processing colons
	var digestPart string
	if strings.Contains(path, "@") {
		parts := strings.SplitN(path, "@", 2)
		path = parts[0]
		digestPart = parts[1]
	}

	// Parse path components first to separate registry from chart path
	// This is important because registry can contain ':' for port (e.g., localhost:5000)
	pathComponents := strings.Split(path, "/")
	if len(pathComponents) < 2 {
		return "", "", "", "", "", fmt.Errorf("invalid OCI URL: must have at least registry and chart")
	}

	// Check for tag (:version) only in the LAST component (chart name)
	var tagPart string
	lastComponent := pathComponents[len(pathComponents)-1]
	if strings.Contains(lastComponent, ":") {
		parts := strings.SplitN(lastComponent, ":", 2)
		pathComponents[len(pathComponents)-1] = parts[0]
		tagPart = parts[1]
	}

	// First component is registry
	registry = pathComponents[0]
	if registry == "" {
		return "", "", "", "", "", fmt.Errorf("invalid OCI URL: empty registry")
	}

	// Last component is chart
	chart = pathComponents[len(pathComponents)-1]
	if chart == "" {
		return "", "", "", "", "", fmt.Errorf("invalid OCI URL: empty chart name")
	}

	// Middle components are repository (if any)
	if len(pathComponents) > 2 {
		repository = strings.Join(pathComponents[1:len(pathComponents)-1], "/")
	}

	// Digest takes precedence over tag
	if digestPart != "" {
		digest = digestPart
		tag = ""
	} else if tagPart != "" {
		tag = tagPart
	} else {
		// Default to latest if neither tag nor digest specified
		tag = "latest"
	}

	return registry, repository, chart, tag, digest, nil
}

// extractRegistryFromOCI extracts the registry from an OCI URL.
// Example: "oci://ghcr.io/org/chart" -> "ghcr.io"
func extractRegistryFromOCI(ociURL string) (string, error) {
	registry, _, _, _, _, err := parseOCIReference(ociURL)
	if err != nil {
		return "", err
	}
	return registry, nil
}

// DockerConfig represents the structure of Docker config.json
type DockerConfig struct {
	Auths map[string]DockerAuth `json:"auths"`
}

// DockerAuth represents authentication for a registry
type DockerAuth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Auth     string `json:"auth,omitempty"` // base64 encoded "username:password"
}

// parseDockerConfigJSON parses Docker config JSON and extracts credentials for the specified registry.
// Supports both explicit username/password fields and base64-encoded auth field.
func parseDockerConfigJSON(configJSON string, registry string) (username, password string, err error) {
	var config DockerConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return "", "", fmt.Errorf("failed to parse docker config JSON: %w", err)
	}

	if config.Auths == nil {
		return "", "", fmt.Errorf("no auths found in docker config")
	}

	auth, ok := config.Auths[registry]
	if !ok {
		return "", "", fmt.Errorf("no auth found for registry %s", registry)
	}

	// Check for explicit username/password
	if auth.Username != "" && auth.Password != "" {
		return auth.Username, auth.Password, nil
	}

	// Check for base64-encoded auth field
	if auth.Auth != "" {
		// Decode base64
		decoded, err := base64.StdEncoding.DecodeString(auth.Auth)
		if err != nil {
			return "", "", fmt.Errorf("failed to decode auth field: %w", err)
		}

		// Split into username:password
		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid auth format: expected 'username:password'")
		}

		return parts[0], parts[1], nil
	}

	return "", "", fmt.Errorf("no credentials found in auth for registry %s", registry)
}

// setupOCIAuth configures Helm to authenticate with OCI registry.
// If AuthSecretName is provided, fetch credentials from Secret.
// Returns cleanup function to remove temporary credentials.
// IMPORTANT: Uses temporary Docker config directory to avoid race conditions.
func setupOCIAuth(kubeconfigPath string, chart ChartSpec) (cleanup func(), err error) {
	// If no auth secret specified, skip authentication (public registry)
	if chart.AuthSecretName == "" {
		return func() {}, nil
	}

	// Extract registry from chart URL
	registry, err := extractRegistryFromOCI(chart.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract registry from OCI URL: %w", err)
	}

	// Fetch Secret using kubectl
	namespace := chart.Namespace
	if namespace == "" {
		namespace = "default"
	}

	log.Printf("Fetching auth secret %s from namespace %s", chart.AuthSecretName, namespace)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfigPath,
		"get", "secret", chart.AuthSecretName,
		"-n", namespace,
		"-o", "json")

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("kubectl get secret timed out after 30 seconds")
		}
		return nil, fmt.Errorf("failed to fetch secret %s: %w, output: %s", chart.AuthSecretName, err, string(output))
	}

	// Parse Secret JSON
	var secret struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(output, &secret); err != nil {
		return nil, fmt.Errorf("failed to parse secret JSON: %w", err)
	}

	// Extract .dockerconfigjson field
	dockerConfigJSON, ok := secret.Data[".dockerconfigjson"]
	if !ok {
		return nil, fmt.Errorf("secret %s does not contain .dockerconfigjson field", chart.AuthSecretName)
	}

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(dockerConfigJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to decode .dockerconfigjson: %w", err)
	}

	// Parse Docker config and extract credentials
	username, password, err := parseDockerConfigJSON(string(decoded), registry)
	if err != nil {
		return nil, fmt.Errorf("failed to parse docker config: %w", err)
	}

	// Create temporary Docker config directory
	tempDockerConfig, err := os.MkdirTemp("", "docker-config-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp docker config directory: %w", err)
	}

	// Set DOCKER_CONFIG environment variable
	originalDockerConfig := os.Getenv("DOCKER_CONFIG")
	if err := os.Setenv("DOCKER_CONFIG", tempDockerConfig); err != nil {
		_ = os.RemoveAll(tempDockerConfig)
		return nil, fmt.Errorf("failed to set DOCKER_CONFIG: %w", err)
	}

	log.Printf("Using temporary Docker config: %s", tempDockerConfig)

	// Run helm registry login
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd = exec.CommandContext(ctx, "helm", "registry", "login", registry,
		"--username", username,
		"--password", password)

	output, err = cmd.CombinedOutput()
	if err != nil {
		// Cleanup temp directory
		_ = os.RemoveAll(tempDockerConfig)
		if originalDockerConfig != "" {
			_ = os.Setenv("DOCKER_CONFIG", originalDockerConfig)
		} else {
			_ = os.Unsetenv("DOCKER_CONFIG")
		}

		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("helm registry login timed out after 30 seconds")
		}
		return nil, fmt.Errorf("helm registry login failed: %w, output: %s", err, string(output))
	}

	log.Printf("Successfully logged in to registry: %s", registry)

	// Return cleanup function
	cleanup = func() {
		// Logout from registry (best effort)
		logoutCtx, logoutCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer logoutCancel()

		logoutCmd := exec.CommandContext(logoutCtx, "helm", "registry", "logout", registry)
		if logoutOutput, logoutErr := logoutCmd.CombinedOutput(); logoutErr != nil {
			log.Printf("Warning: helm registry logout failed: %v, output: %s", logoutErr, string(logoutOutput))
		}

		// Restore original DOCKER_CONFIG
		if originalDockerConfig != "" {
			if err := os.Setenv("DOCKER_CONFIG", originalDockerConfig); err != nil {
				log.Printf("Warning: failed to restore DOCKER_CONFIG: %v", err)
			}
		} else {
			if err := os.Unsetenv("DOCKER_CONFIG"); err != nil {
				log.Printf("Warning: failed to unset DOCKER_CONFIG: %v", err)
			}
		}

		// Remove temporary config directory
		if err := os.RemoveAll(tempDockerConfig); err != nil {
			log.Printf("Warning: failed to remove temp docker config %s: %v", tempDockerConfig, err)
		}

		log.Printf("Cleaned up OCI auth for registry: %s", registry)
	}

	return cleanup, nil
}

// validateOCISource validates required fields for OCI source type
func validateOCISource(chart ChartSpec) error {
	// Validate URL
	if chart.URL == "" {
		return fmt.Errorf("url is required for oci source type")
	}

	// Validate URL starts with oci://
	if !strings.HasPrefix(chart.URL, "oci://") {
		return fmt.Errorf("url must start with oci:// for oci source type")
	}

	// Validate URL is parseable
	_, _, _, _, _, err := parseOCIReference(chart.URL)
	if err != nil {
		return fmt.Errorf("invalid oci url: %w", err)
	}

	// Git fields should not be set for OCI sources
	if chart.GitBranch != "" || chart.GitTag != "" || chart.GitCommit != "" || chart.GitSemVer != "" {
		return fmt.Errorf("git reference fields (GitBranch, GitTag, GitCommit, GitSemVer) should not be set for oci source type")
	}

	// ChartPath should not be set for OCI sources
	if chart.ChartPath != "" {
		return fmt.Errorf("chartPath should not be set for oci source type")
	}

	// ChartName should not be set for OCI sources (chart name is in the URL)
	if chart.ChartName != "" {
		return fmt.Errorf("chartName should not be set for oci source type (chart name is part of the oci URL)")
	}

	return nil
}

// verifyOCISignature verifies OCI chart signature using specified provider.
// Returns nil if verification passes or if OCIProvider is not set.
// Currently logs a warning if OCIProvider is set (not yet fully implemented).
func verifyOCISignature(chart ChartSpec) error {
	// If no OCIProvider specified, skip verification
	if chart.OCIProvider == "" {
		return nil
	}

	// Log warning that OCIProvider verification is not yet fully implemented
	log.Printf("Warning: OCIProvider verification (%s) is not yet fully implemented. Skipping signature verification for chart %s", chart.OCIProvider, chart.Name)

	// In the future, implement verification using:
	// - cosign: for chart.OCIProvider == "cosign"
	// - notation: for chart.OCIProvider == "notation"

	// For now, just return nil (don't fail the installation)
	return nil
}

// -------------------------------------------------------------------------
// S3 Source Type Functions
// -------------------------------------------------------------------------

// extractS3CredentialsFromSecret extracts S3 credentials from a Kubernetes Secret's Data field.
// The Secret should contain base64-encoded keys: accessKeyID, secretAccessKey, and optionally sessionToken.
// Returns the credentials as strings and an error if required fields are missing.
func extractS3CredentialsFromSecret(secretData map[string]string) (accessKeyID, secretAccessKey, sessionToken string, err error) {
	// Extract accessKeyID (required)
	accessKeyID, ok := secretData["accessKeyID"]
	if !ok || accessKeyID == "" {
		return "", "", "", fmt.Errorf("accessKeyID is required in Secret")
	}

	// Extract secretAccessKey (required)
	secretAccessKey, ok = secretData["secretAccessKey"]
	if !ok || secretAccessKey == "" {
		return "", "", "", fmt.Errorf("secretAccessKey is required in Secret")
	}

	// Extract sessionToken (optional)
	sessionToken = secretData["sessionToken"]

	return accessKeyID, secretAccessKey, sessionToken, nil
}

// validateS3Credentials validates that the required S3 credentials are present.
func validateS3Credentials(accessKeyID, secretAccessKey string) error {
	if accessKeyID == "" {
		return fmt.Errorf("accessKeyID is required")
	}
	if secretAccessKey == "" {
		return fmt.Errorf("secretAccessKey is required")
	}
	return nil
}

// setupS3Auth configures and returns an S3 client with credentials from a Kubernetes Secret.
// If AuthSecretName is not set, it returns a client using default credentials (IAM role).
// Returns an error if the Secret cannot be fetched or credentials are invalid.
func setupS3Auth(kubeconfigPath string, chart ChartSpec) (*S3Client, error) {
	// Determine region (default to us-east-1)
	region := chart.S3BucketRegion
	if region == "" {
		region = "us-east-1"
	}

	// If no auth secret specified, use default credentials (IAM role)
	if chart.AuthSecretName == "" {
		log.Printf("No AuthSecretName specified, using default AWS credentials (IAM role)")
		return NewS3Client(chart.URL, region)
	}

	// Fetch Secret using kubectl
	namespace := chart.Namespace
	if namespace == "" {
		namespace = "default"
	}

	log.Printf("Fetching S3 auth secret %s from namespace %s", chart.AuthSecretName, namespace)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfigPath,
		"get", "secret", chart.AuthSecretName,
		"-n", namespace,
		"-o", "jsonpath={.data}")

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("kubectl get secret timed out after 30 seconds")
		}
		return nil, fmt.Errorf("failed to fetch secret %s: %w, output: %s", chart.AuthSecretName, err, string(output))
	}

	// Parse Secret data (base64-encoded values)
	var secretData map[string]string
	if err := json.Unmarshal(output, &secretData); err != nil {
		return nil, fmt.Errorf("failed to parse secret data: %w", err)
	}

	// Decode base64 values
	decodedData := make(map[string]string)
	for key, value := range secretData {
		decoded, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return nil, fmt.Errorf("failed to decode secret field %s: %w", key, err)
		}
		decodedData[key] = string(decoded)
	}

	// Extract credentials from Secret
	accessKeyID, secretAccessKey, sessionToken, err := extractS3CredentialsFromSecret(decodedData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract S3 credentials: %w", err)
	}

	// Validate credentials
	if err := validateS3Credentials(accessKeyID, secretAccessKey); err != nil {
		return nil, err
	}

	// Create S3 client with explicit credentials
	log.Printf("Creating S3 client with credentials from Secret %s", chart.AuthSecretName)
	return NewS3ClientWithCredentials(chart.URL, region, accessKeyID, secretAccessKey, sessionToken)
}

// parseS3Path extracts the S3 object key from the ChartPath.
// For S3, ChartPath is already the object key (relative path within bucket).
func parseS3Path(chartPath string) string {
	return chartPath
}

// buildS3DownloadParams extracts the bucket name and object key from ChartSpec.
// Returns an error if required fields are missing.
func buildS3DownloadParams(chart ChartSpec) (bucket, key string, err error) {
	// Validate bucket name
	if chart.S3BucketName == "" {
		return "", "", fmt.Errorf("s3BucketName is required for S3 source type")
	}

	// Validate chart path
	if chart.ChartPath == "" {
		return "", "", fmt.Errorf("chartPath is required for S3 source type")
	}

	bucket = chart.S3BucketName
	key = parseS3Path(chart.ChartPath)

	return bucket, key, nil
}

// downloadFromS3 downloads a chart tarball from an S3 bucket to a local directory.
// Returns the full path to the downloaded chart file.
func downloadFromS3(client *S3Client, chart ChartSpec, destDir string) (string, error) {
	// Build download parameters
	bucket, key, err := buildS3DownloadParams(chart)
	if err != nil {
		return "", err
	}

	log.Printf("Downloading chart from S3: bucket=%s, key=%s", bucket, key)

	// Extract filename from key (last path component)
	filename := filepath.Base(key)
	if filename == "" || filename == "." || filename == "/" {
		return "", fmt.Errorf("invalid chart path: cannot determine filename from %s", key)
	}

	// Construct destination path
	destPath := filepath.Join(destDir, filename)

	// Download file from S3
	if err := client.DownloadFile(bucket, key, destPath); err != nil {
		return "", fmt.Errorf("failed to download from S3: %w", err)
	}

	// Verify the downloaded file exists
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		return "", fmt.Errorf("downloaded file not found at %s", destPath)
	}

	log.Printf("Successfully downloaded chart to: %s", destPath)
	return destPath, nil
}

// validateS3Source validates required fields for S3 source type.
func validateS3Source(chart ChartSpec) error {
	// Validate URL
	if chart.URL == "" {
		return fmt.Errorf("url is required for s3 source type")
	}

	// Validate URL format (must be http:// or https://)
	if !strings.HasPrefix(chart.URL, "http://") && !strings.HasPrefix(chart.URL, "https://") {
		return fmt.Errorf("invalid url format: must start with http:// or https://")
	}

	// Validate S3BucketName
	if chart.S3BucketName == "" {
		return fmt.Errorf("s3BucketName is required for s3 source type")
	}

	// Validate ChartPath
	if chart.ChartPath == "" {
		return fmt.Errorf("chartPath is required for s3 source type")
	}

	// Validate ChartPath ends with .tgz or .tar.gz
	if !strings.HasSuffix(chart.ChartPath, ".tgz") && !strings.HasSuffix(chart.ChartPath, ".tar.gz") {
		return fmt.Errorf("chartPath must end with .tgz or .tar.gz for s3 source type")
	}

	// Git fields should not be set for S3 sources
	if chart.GitBranch != "" || chart.GitTag != "" || chart.GitCommit != "" || chart.GitSemVer != "" {
		return fmt.Errorf("git reference fields (GitBranch, GitTag, GitCommit, GitSemVer) should not be set for s3 source type")
	}

	// OCI fields should not be set for S3 sources
	if chart.OCIProvider != "" || chart.OCILayerMediaType != "" {
		return fmt.Errorf("oci fields (OCIProvider, OCILayerMediaType) should not be set for s3 source type")
	}

	// ChartName should not be set for S3 sources
	if chart.ChartName != "" {
		return fmt.Errorf("chartName should not be set for s3 source type (chart name is in the tarball)")
	}

	return nil
}
