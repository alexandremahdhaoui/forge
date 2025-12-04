//go:build integration

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
)

// setupTestCluster creates a test Kubernetes cluster using testenv-kind
// Returns the kubeconfig path and a cleanup function
func setupTestCluster(t *testing.T) (kubeconfigPath string, cleanup func()) {
	t.Helper()

	// Check if SKIP_INTEGRATION_TESTS is set
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration test (SKIP_INTEGRATION_TESTS is set)")
	}

	// Create a temporary directory for test artifacts
	tempDir := t.TempDir()
	kubeconfigPath = filepath.Join(tempDir, "kubeconfig")

	// Generate unique cluster name
	clusterName := fmt.Sprintf("testenv-helm-install-%s", t.Name())
	clusterName = strings.ReplaceAll(clusterName, "/", "-")
	clusterName = strings.ToLower(clusterName)

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

// createTestChart creates a simple Helm chart in the specified directory
// Returns the path to the chart directory
func createTestChart(t *testing.T, chartPath string) string {
	t.Helper()

	// Create chart directory
	if err := os.MkdirAll(chartPath, 0o755); err != nil {
		t.Fatalf("Failed to create chart directory: %v", err)
	}

	// Create Chart.yaml
	chartYAML := `apiVersion: v2
name: test-chart
description: A Helm chart for testing
type: application
version: 0.1.0
appVersion: "1.0"
`
	if err := os.WriteFile(filepath.Join(chartPath, "Chart.yaml"), []byte(chartYAML), 0o644); err != nil {
		t.Fatalf("Failed to write Chart.yaml: %v", err)
	}

	// Create templates directory
	templatesDir := filepath.Join(chartPath, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatalf("Failed to create templates directory: %v", err)
	}

	// Create a simple deployment template
	deployment := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Release.Name }}
  labels:
    app: {{ .Release.Name }}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{ .Release.Name }}
  template:
    metadata:
      labels:
        app: {{ .Release.Name }}
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
`
	if err := os.WriteFile(filepath.Join(templatesDir, "deployment.yaml"), []byte(deployment), 0o644); err != nil {
		t.Fatalf("Failed to write deployment.yaml: %v", err)
	}

	// Create a simple service template
	service := `apiVersion: v1
kind: Service
metadata:
  name: {{ .Release.Name }}
spec:
  type: ClusterIP
  ports:
  - port: 80
    targetPort: 80
    protocol: TCP
  selector:
    app: {{ .Release.Name }}
`
	if err := os.WriteFile(filepath.Join(templatesDir, "service.yaml"), []byte(service), 0o644); err != nil {
		t.Fatalf("Failed to write service.yaml: %v", err)
	}

	return chartPath
}

// verifyRelease checks if a Helm release exists in the cluster
// Returns true if the release is found and deployed
func verifyRelease(t *testing.T, kubeconfigPath, releaseName, namespace string) bool {
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

// applyManifest applies a Kubernetes manifest to the cluster
func applyManifest(t *testing.T, kubeconfigPath, manifest string) {
	t.Helper()

	cmd := exec.Command("kubectl", "apply", "-f", "-", "--kubeconfig", kubeconfigPath)
	cmd.Stdin = strings.NewReader(manifest)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to apply manifest: %v\nStderr: %s", err, stderr.String())
	}
}
