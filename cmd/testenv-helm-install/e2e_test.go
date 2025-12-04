//go:build e2e

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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestE2E_HelmRepo tests Helm repository chart source end-to-end
func TestE2E_HelmRepo(t *testing.T) {
	if os.Getenv("SKIP_E2E_TESTS") != "" {
		t.Skip("Skipping E2E test (SKIP_E2E_TESTS is set)")
	}

	kubeconfigPath, cleanup := setupE2ETestCluster(t)
	defer cleanup()

	// Create namespace
	createNamespace(t, kubeconfigPath, "helm-repo-test")

	// Create ChartSpec for nginx from Bitnami repo
	chartSpec := map[string]interface{}{
		"releaseName": "nginx-test",
		"namespace":   "helm-repo-test",
		"chart": map[string]interface{}{
			"spec": map[string]interface{}{
				"sourceRef": map[string]interface{}{
					"name": "bitnami",
				},
				"chart":   "nginx",
				"version": "15.0.0",
			},
		},
		"helmRepositories": []map[string]interface{}{
			{
				"name": "bitnami",
				"url":  "https://charts.bitnami.com/bitnami",
			},
		},
	}

	// Install chart
	installChart(t, kubeconfigPath, chartSpec)

	// Verify release
	if !verifyReleaseE2E(t, kubeconfigPath, "nginx-test", "helm-repo-test") {
		t.Fatal("Release not found or not deployed")
	}

	// Verify deployment is running
	verifyDeployment(t, kubeconfigPath, "helm-repo-test", "nginx-test")

	t.Log("SUCCESS: Helm repository source test passed")
}

// TestE2E_Git tests Git repository chart source end-to-end
func TestE2E_Git(t *testing.T) {
	if os.Getenv("SKIP_E2E_TESTS") != "" {
		t.Skip("Skipping E2E test (SKIP_E2E_TESTS is set)")
	}

	kubeconfigPath, cleanup := setupE2ETestCluster(t)
	defer cleanup()

	// Create namespace
	createNamespace(t, kubeconfigPath, "git-test")

	// Create a test Git repository with a Helm chart
	gitRepoPath := createTestGitRepo(t)

	// Create ChartSpec for Git source
	chartSpec := map[string]interface{}{
		"releaseName": "git-chart-test",
		"namespace":   "git-test",
		"chart": map[string]interface{}{
			"spec": map[string]interface{}{
				"sourceRef": map[string]interface{}{
					"kind": "GitRepository",
					"name": "test-repo",
				},
				"chart": "charts/test-chart",
			},
		},
		"gitRepositories": []map[string]interface{}{
			{
				"name": "test-repo",
				"url":  fmt.Sprintf("file://%s", gitRepoPath),
				"ref": map[string]interface{}{
					"branch": "main",
				},
			},
		},
	}

	// Install chart
	installChart(t, kubeconfigPath, chartSpec)

	// Verify release
	if !verifyReleaseE2E(t, kubeconfigPath, "git-chart-test", "git-test") {
		t.Fatal("Release not found or not deployed")
	}

	t.Log("SUCCESS: Git source test passed")
}

// TestE2E_OCI tests OCI registry chart source end-to-end
func TestE2E_OCI(t *testing.T) {
	if os.Getenv("SKIP_E2E_TESTS") != "" {
		t.Skip("Skipping E2E test (SKIP_E2E_TESTS is set)")
	}

	kubeconfigPath, cleanup := setupE2ETestCluster(t)
	defer cleanup()

	// Create namespace
	createNamespace(t, kubeconfigPath, "oci-test")

	// Use nginx chart from Docker Hub (OCI registry)
	chartSpec := map[string]interface{}{
		"releaseName": "oci-chart-test",
		"namespace":   "oci-test",
		"chart": map[string]interface{}{
			"spec": map[string]interface{}{
				"sourceRef": map[string]interface{}{
					"kind": "OCIRepository",
					"name": "dockerhub",
				},
				"chart":   "oci://registry-1.docker.io/bitnamicharts/nginx",
				"version": "15.0.0",
			},
		},
		"ociRepositories": []map[string]interface{}{
			{
				"name": "dockerhub",
				"url":  "oci://registry-1.docker.io/bitnamicharts",
			},
		},
	}

	// Install chart
	installChart(t, kubeconfigPath, chartSpec)

	// Verify release
	if !verifyReleaseE2E(t, kubeconfigPath, "oci-chart-test", "oci-test") {
		t.Fatal("Release not found or not deployed")
	}

	// Verify deployment is running
	verifyDeployment(t, kubeconfigPath, "oci-test", "oci-chart-test")

	t.Log("SUCCESS: OCI source test passed")
}

// TestE2E_ValueReferences tests ConfigMap/Secret value references end-to-end
func TestE2E_ValueReferences(t *testing.T) {
	if os.Getenv("SKIP_E2E_TESTS") != "" {
		t.Skip("Skipping E2E test (SKIP_E2E_TESTS is set)")
	}

	kubeconfigPath, cleanup := setupE2ETestCluster(t)
	defer cleanup()

	// Create namespace
	createNamespace(t, kubeconfigPath, "valuerefs-test")

	// Create ConfigMap with values
	configMapManifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: helm-values
  namespace: valuerefs-test
data:
  values.yaml: |
    replicaCount: 2
    service:
      type: NodePort
`
	applyManifestE2E(t, kubeconfigPath, configMapManifest)

	// Create Secret with sensitive values
	secretManifest := `apiVersion: v1
kind: Secret
metadata:
  name: helm-secrets
  namespace: valuerefs-test
type: Opaque
stringData:
  password.yaml: |
    auth:
      password: "supersecret"
`
	applyManifestE2E(t, kubeconfigPath, secretManifest)

	// Create ChartSpec with ValueReferences
	chartSpec := map[string]interface{}{
		"releaseName": "valuerefs-test",
		"namespace":   "valuerefs-test",
		"chart": map[string]interface{}{
			"spec": map[string]interface{}{
				"sourceRef": map[string]interface{}{
					"name": "bitnami",
				},
				"chart":   "nginx",
				"version": "15.0.0",
			},
		},
		"helmRepositories": []map[string]interface{}{
			{
				"name": "bitnami",
				"url":  "https://charts.bitnami.com/bitnami",
			},
		},
		"valuesFrom": []map[string]interface{}{
			{
				"kind": "ConfigMap",
				"name": "helm-values",
			},
			{
				"kind":     "Secret",
				"name":     "helm-secrets",
				"optional": false,
			},
		},
	}

	// Install chart
	installChart(t, kubeconfigPath, chartSpec)

	// Verify release
	if !verifyReleaseE2E(t, kubeconfigPath, "valuerefs-test", "valuerefs-test") {
		t.Fatal("Release not found or not deployed")
	}

	// Verify deployment has 2 replicas (from ConfigMap)
	verifyReplicaCount(t, kubeconfigPath, "valuerefs-test", "valuerefs-test", 2)

	t.Log("SUCCESS: ValueReferences test passed")
}

// TestE2E_NestedValues tests complex nested YAML values end-to-end
func TestE2E_NestedValues(t *testing.T) {
	if os.Getenv("SKIP_E2E_TESTS") != "" {
		t.Skip("Skipping E2E test (SKIP_E2E_TESTS is set)")
	}

	kubeconfigPath, cleanup := setupE2ETestCluster(t)
	defer cleanup()

	// Create namespace
	createNamespace(t, kubeconfigPath, "nested-values-test")

	// Create ChartSpec with complex nested values
	chartSpec := map[string]interface{}{
		"releaseName": "nested-values-test",
		"namespace":   "nested-values-test",
		"chart": map[string]interface{}{
			"spec": map[string]interface{}{
				"sourceRef": map[string]interface{}{
					"name": "bitnami",
				},
				"chart":   "nginx",
				"version": "15.0.0",
			},
		},
		"helmRepositories": []map[string]interface{}{
			{
				"name": "bitnami",
				"url":  "https://charts.bitnami.com/bitnami",
			},
		},
		"values": map[string]interface{}{
			"replicaCount": 3,
			"service": map[string]interface{}{
				"type": "NodePort",
				"ports": map[string]interface{}{
					"http": 8080,
				},
			},
			"resources": map[string]interface{}{
				"limits": map[string]interface{}{
					"cpu":    "200m",
					"memory": "256Mi",
				},
				"requests": map[string]interface{}{
					"cpu":    "100m",
					"memory": "128Mi",
				},
			},
		},
	}

	// Install chart
	installChart(t, kubeconfigPath, chartSpec)

	// Verify release
	if !verifyReleaseE2E(t, kubeconfigPath, "nested-values-test", "nested-values-test") {
		t.Fatal("Release not found or not deployed")
	}

	// Verify deployment has 3 replicas (from nested values)
	verifyReplicaCount(t, kubeconfigPath, "nested-values-test", "nested-values-test", 3)

	t.Log("SUCCESS: Nested values test passed")
}

// TestE2E_MultipleCharts tests multiple charts in one ChartSpec
func TestE2E_MultipleCharts(t *testing.T) {
	if os.Getenv("SKIP_E2E_TESTS") != "" {
		t.Skip("Skipping E2E test (SKIP_E2E_TESTS is set)")
	}

	kubeconfigPath, cleanup := setupE2ETestCluster(t)
	defer cleanup()

	// Create namespace
	createNamespace(t, kubeconfigPath, "multi-chart-test")

	// Create ChartSpec with multiple charts
	chartSpec := map[string]interface{}{
		"charts": []map[string]interface{}{
			{
				"releaseName": "nginx-1",
				"namespace":   "multi-chart-test",
				"chart": map[string]interface{}{
					"spec": map[string]interface{}{
						"sourceRef": map[string]interface{}{
							"name": "bitnami",
						},
						"chart":   "nginx",
						"version": "15.0.0",
					},
				},
			},
			{
				"releaseName": "nginx-2",
				"namespace":   "multi-chart-test",
				"chart": map[string]interface{}{
					"spec": map[string]interface{}{
						"sourceRef": map[string]interface{}{
							"name": "bitnami",
						},
						"chart":   "nginx",
						"version": "15.0.0",
					},
				},
			},
		},
		"helmRepositories": []map[string]interface{}{
			{
				"name": "bitnami",
				"url":  "https://charts.bitnami.com/bitnami",
			},
		},
	}

	// Install charts
	installChart(t, kubeconfigPath, chartSpec)

	// Verify both releases
	if !verifyReleaseE2E(t, kubeconfigPath, "nginx-1", "multi-chart-test") {
		t.Fatal("Release nginx-1 not found or not deployed")
	}
	if !verifyReleaseE2E(t, kubeconfigPath, "nginx-2", "multi-chart-test") {
		t.Fatal("Release nginx-2 not found or not deployed")
	}

	t.Log("SUCCESS: Multiple charts test passed")
}

// TestE2E_CompleteLifecycle tests create, verify, and delete lifecycle
func TestE2E_CompleteLifecycle(t *testing.T) {
	if os.Getenv("SKIP_E2E_TESTS") != "" {
		t.Skip("Skipping E2E test (SKIP_E2E_TESTS is set)")
	}

	kubeconfigPath, cleanup := setupE2ETestCluster(t)
	defer cleanup()

	// Create namespace
	createNamespace(t, kubeconfigPath, "lifecycle-test")

	// Create ChartSpec
	chartSpec := map[string]interface{}{
		"releaseName": "lifecycle-test",
		"namespace":   "lifecycle-test",
		"chart": map[string]interface{}{
			"spec": map[string]interface{}{
				"sourceRef": map[string]interface{}{
					"name": "bitnami",
				},
				"chart":   "nginx",
				"version": "15.0.0",
			},
		},
		"helmRepositories": []map[string]interface{}{
			{
				"name": "bitnami",
				"url":  "https://charts.bitnami.com/bitnami",
			},
		},
	}

	// 1. Install chart
	installChart(t, kubeconfigPath, chartSpec)

	// 2. Verify release exists
	if !verifyReleaseE2E(t, kubeconfigPath, "lifecycle-test", "lifecycle-test") {
		t.Fatal("Release not found after installation")
	}

	// 3. Uninstall chart
	uninstallChart(t, kubeconfigPath, "lifecycle-test", "lifecycle-test")

	// 4. Verify release is gone
	if verifyReleaseE2E(t, kubeconfigPath, "lifecycle-test", "lifecycle-test") {
		t.Fatal("Release still exists after uninstallation")
	}

	t.Log("SUCCESS: Complete lifecycle test passed")
}

// Helper functions

func setupE2ETestCluster(t *testing.T) (kubeconfigPath string, cleanup func()) {
	t.Helper()

	// Create a temporary directory for test artifacts
	tempDir := t.TempDir()
	kubeconfigPath = filepath.Join(tempDir, "kubeconfig")

	// Generate unique cluster name
	clusterName := fmt.Sprintf("e2e-helm-install-%s", strings.ToLower(t.Name()))
	clusterName = strings.ReplaceAll(clusterName, "/", "-")
	clusterName = strings.ReplaceAll(clusterName, "_", "-")

	// Create cluster using testenv-kind
	createCmd := exec.Command("testenv-kind", "--mcp")
	createInput := map[string]interface{}{
		"method": "tools/call",
		"params": map[string]interface{}{
			"name": "create",
			"arguments": map[string]interface{}{
				"name":           clusterName,
				"kubeconfigPath": kubeconfigPath,
			},
		},
	}

	inputJSON, err := json.Marshal(createInput)
	if err != nil {
		t.Fatalf("Failed to marshal create input: %v", err)
	}

	createCmd.Stdin = bytes.NewReader(inputJSON)
	var stdout, stderr bytes.Buffer
	createCmd.Stdout = &stdout
	createCmd.Stderr = &stderr

	if err := createCmd.Run(); err != nil {
		t.Fatalf("Failed to create test cluster: %v\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}

	// Verify kubeconfig was created
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		t.Fatalf("Kubeconfig not created at %s", kubeconfigPath)
	}

	cleanup = func() {
		// Delete the cluster
		deleteCmd := exec.Command("testenv-kind", "--mcp")
		deleteInput := map[string]interface{}{
			"method": "tools/call",
			"params": map[string]interface{}{
				"name": "delete",
				"arguments": map[string]interface{}{
					"name": clusterName,
				},
			},
		}

		deleteJSON, err := json.Marshal(deleteInput)
		if err != nil {
			t.Logf("Failed to marshal delete input: %v", err)
			return
		}

		deleteCmd.Stdin = bytes.NewReader(deleteJSON)
		if err := deleteCmd.Run(); err != nil {
			t.Logf("Failed to delete test cluster: %v", err)
		}
	}

	return kubeconfigPath, cleanup
}

func createNamespace(t *testing.T, kubeconfigPath, namespace string) {
	t.Helper()

	cmd := exec.Command("kubectl", "create", "namespace", namespace, "--kubeconfig", kubeconfigPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Namespace might already exist, that's okay
		if !strings.Contains(stderr.String(), "already exists") {
			t.Fatalf("Failed to create namespace: %v\nStderr: %s", err, stderr.String())
		}
	}
}

func installChart(t *testing.T, kubeconfigPath string, chartSpec map[string]interface{}) {
	t.Helper()

	// Convert ChartSpec to JSON
	chartSpecJSON, err := json.Marshal(chartSpec)
	if err != nil {
		t.Fatalf("Failed to marshal ChartSpec: %v", err)
	}

	// Save ChartSpec to temp file
	tempDir := t.TempDir()
	chartSpecPath := filepath.Join(tempDir, "chartspec.json")
	if err := os.WriteFile(chartSpecPath, chartSpecJSON, 0o644); err != nil {
		t.Fatalf("Failed to write ChartSpec: %v", err)
	}

	// Create MCP input for testenv-helm-install
	mcpInput := map[string]interface{}{
		"method": "tools/call",
		"params": map[string]interface{}{
			"name": "install",
			"arguments": map[string]interface{}{
				"kubeconfigPath": kubeconfigPath,
				"chartSpecPath":  chartSpecPath,
			},
		},
	}

	inputJSON, err := json.Marshal(mcpInput)
	if err != nil {
		t.Fatalf("Failed to marshal MCP input: %v", err)
	}

	// Execute testenv-helm-install
	cmd := exec.Command("testenv-helm-install", "--mcp")
	cmd.Stdin = bytes.NewReader(inputJSON)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to install chart: %v\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}

	// Wait for deployment to be ready (up to 2 minutes)
	time.Sleep(5 * time.Second)
}

func verifyReleaseE2E(t *testing.T, kubeconfigPath, releaseName, namespace string) bool {
	t.Helper()

	cmd := exec.Command("helm", "list", "-n", namespace, "-o", "json", "--kubeconfig", kubeconfigPath)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Logf("Failed to list releases: %v", err)
		return false
	}

	var releases []map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &releases); err != nil {
		t.Logf("Failed to parse helm list output: %v", err)
		return false
	}

	for _, release := range releases {
		if name, ok := release["name"].(string); ok && name == releaseName {
			status, _ := release["status"].(string)
			return status == "deployed"
		}
	}

	return false
}

func verifyDeployment(t *testing.T, kubeconfigPath, namespace, deploymentName string) {
	t.Helper()

	// Wait up to 2 minutes for deployment to be ready
	for i := 0; i < 24; i++ {
		cmd := exec.Command("kubectl", "get", "deployment", deploymentName,
			"-n", namespace, "--kubeconfig", kubeconfigPath, "-o", "json")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err == nil {
			var deployment map[string]interface{}
			if err := json.Unmarshal(stdout.Bytes(), &deployment); err == nil {
				status, ok := deployment["status"].(map[string]interface{})
				if ok {
					readyReplicas, _ := status["readyReplicas"].(float64)
					replicas, _ := status["replicas"].(float64)
					if readyReplicas > 0 && readyReplicas == replicas {
						t.Logf("Deployment %s is ready (%v/%v replicas)", deploymentName, readyReplicas, replicas)
						return
					}
				}
			}
		}

		time.Sleep(5 * time.Second)
	}

	t.Fatalf("Deployment %s did not become ready within 2 minutes", deploymentName)
}

func verifyReplicaCount(t *testing.T, kubeconfigPath, namespace, deploymentName string, expectedReplicas int) {
	t.Helper()

	cmd := exec.Command("kubectl", "get", "deployment", deploymentName,
		"-n", namespace, "--kubeconfig", kubeconfigPath, "-o", "json")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to get deployment: %v", err)
	}

	var deployment map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &deployment); err != nil {
		t.Fatalf("Failed to parse deployment JSON: %v", err)
	}

	spec, ok := deployment["spec"].(map[string]interface{})
	if !ok {
		t.Fatal("Deployment spec not found")
	}

	replicas, ok := spec["replicas"].(float64)
	if !ok {
		t.Fatal("Replicas field not found in deployment spec")
	}

	if int(replicas) != expectedReplicas {
		t.Fatalf("Expected %d replicas, got %d", expectedReplicas, int(replicas))
	}
}

func uninstallChart(t *testing.T, kubeconfigPath, releaseName, namespace string) {
	t.Helper()

	cmd := exec.Command("helm", "uninstall", releaseName, "-n", namespace, "--kubeconfig", kubeconfigPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to uninstall chart: %v\nStderr: %s", err, stderr.String())
	}
}

func applyManifestE2E(t *testing.T, kubeconfigPath, manifest string) {
	t.Helper()

	cmd := exec.Command("kubectl", "apply", "-f", "-", "--kubeconfig", kubeconfigPath)
	cmd.Stdin = strings.NewReader(manifest)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to apply manifest: %v\nStderr: %s", err, stderr.String())
	}
}

func createTestGitRepo(t *testing.T) string {
	t.Helper()

	// Create temp directory for Git repo
	repoPath := t.TempDir()

	// Initialize Git repo
	runGitCommand(t, repoPath, "init")
	runGitCommand(t, repoPath, "config", "user.email", "test@example.com")
	runGitCommand(t, repoPath, "config", "user.name", "Test User")

	// Create Helm chart in charts/test-chart
	chartPath := filepath.Join(repoPath, "charts", "test-chart")
	if err := os.MkdirAll(chartPath, 0o755); err != nil {
		t.Fatalf("Failed to create chart directory: %v", err)
	}

	// Create Chart.yaml
	chartYAML := `apiVersion: v2
name: test-chart
description: A test Helm chart
type: application
version: 1.0.0
appVersion: "1.0"
`
	if err := os.WriteFile(filepath.Join(chartPath, "Chart.yaml"), []byte(chartYAML), 0o644); err != nil {
		t.Fatalf("Failed to write Chart.yaml: %v", err)
	}

	// Create templates directory with a simple ConfigMap
	templatesDir := filepath.Join(chartPath, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatalf("Failed to create templates directory: %v", err)
	}

	configMap := `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}
data:
  message: "Hello from Git chart"
`
	if err := os.WriteFile(filepath.Join(templatesDir, "configmap.yaml"), []byte(configMap), 0o644); err != nil {
		t.Fatalf("Failed to write configmap.yaml: %v", err)
	}

	// Commit the chart
	runGitCommand(t, repoPath, "add", ".")
	runGitCommand(t, repoPath, "commit", "-m", "Initial commit")

	// Create main branch (Git 2.28+ uses 'main' by default, older versions use 'master')
	runGitCommand(t, repoPath, "branch", "-M", "main")

	return repoPath
}

func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("Git command failed: git %v\nError: %v\nStderr: %s", args, err, stderr.String())
	}
}
