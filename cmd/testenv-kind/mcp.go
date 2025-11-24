package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

// runMCPServer starts the testenv-kind MCP server with stdio transport.
func runMCPServer() error {
	server := mcpserver.New("testenv-kind", Version)

	config := engineframework.TestEnvSubengineConfig{
		Name:       "testenv-kind",
		Version:    Version,
		CreateFunc: createKindCluster,
		DeleteFunc: deleteKindCluster,
	}

	if err := engineframework.RegisterTestEnvSubengineTools(server, config); err != nil {
		return err
	}

	return server.RunDefault()
}

// createKindCluster implements the CreateFunc for creating kind clusters.
func createKindCluster(ctx context.Context, input engineframework.CreateInput) (*engineframework.TestEnvArtifact, error) {
	log.Printf("Creating kind cluster: testID=%s, stage=%s", input.TestID, input.Stage)

	// Read forge.yaml configuration
	config, err := forge.ReadSpec()
	if err != nil {
		return nil, fmt.Errorf("failed to read forge spec: %w", err)
	}

	// Read environment variables
	envs, err := readEnvs()
	if err != nil {
		return nil, fmt.Errorf("failed to read environment variables: %w", err)
	}

	// Generate cluster name and kubeconfig path
	clusterName := fmt.Sprintf("%s-%s", config.Name, input.TestID)
	kubeconfigPath := filepath.Join(input.TmpDir, "kubeconfig")

	// Update config with cluster-specific values
	config.Name = clusterName
	config.Kindenv.KubeconfigPath = kubeconfigPath

	// Create the kind cluster
	if err := doSetup(config, envs); err != nil {
		return nil, fmt.Errorf("failed to create kind cluster: %w", err)
	}

	// Prepare files map (relative paths within tmpDir)
	files := map[string]string{
		"testenv-kind.kubeconfig": "kubeconfig",
	}

	// Prepare metadata
	metadata := map[string]string{
		"testenv-kind.clusterName":    clusterName,
		"testenv-kind.kubeconfigPath": kubeconfigPath,
	}

	// Prepare managed resources (for cleanup)
	managedResources := []string{
		kubeconfigPath,
	}

	// Return artifact
	return &engineframework.TestEnvArtifact{
		TestID:           input.TestID,
		Files:            files,
		Metadata:         metadata,
		ManagedResources: managedResources,
		Env: map[string]string{
			"KUBECONFIG": kubeconfigPath,
		},
	}, nil
}

// deleteKindCluster implements the DeleteFunc for deleting kind clusters.
func deleteKindCluster(ctx context.Context, input engineframework.DeleteInput) error {
	log.Printf("Deleting kind cluster: testID=%s", input.TestID)

	// Read forge.yaml configuration
	config, err := forge.ReadSpec()
	if err != nil {
		return fmt.Errorf("failed to read forge spec: %w", err)
	}

	// Read environment variables
	envs, err := readEnvs()
	if err != nil {
		return fmt.Errorf("failed to read environment variables: %w", err)
	}

	// Reconstruct cluster name from testID
	clusterName := fmt.Sprintf("%s-%s", config.Name, input.TestID)
	config.Name = clusterName

	// Delete the kind cluster (best-effort)
	if err := doTeardown(config, envs); err != nil {
		// Log warning but don't fail - best effort cleanup
		log.Printf("Warning: failed to delete kind cluster: %v", err)
		return nil
	}

	return nil
}
