package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcputil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"sigs.k8s.io/yaml"
)

// CreateImagePullSecretInput represents the input for the create-image-pull-secret tool.
type CreateImagePullSecretInput struct {
	TestID     string            `json:"testID"`               // Test environment ID (required)
	Namespace  string            `json:"namespace"`            // Kubernetes namespace where secret should be created (required)
	SecretName string            `json:"secretName,omitempty"` // Name of the secret (optional, defaults to "local-container-registry-credentials")
	Metadata   map[string]string `json:"metadata"`             // Metadata from testenv (optional, provides paths and registry FQDN)
}

// ListImagePullSecretsInput represents the input for the list-image-pull-secrets tool.
type ListImagePullSecretsInput struct {
	TestID    string            `json:"testID"`              // Test environment ID (required)
	Namespace string            `json:"namespace,omitempty"` // Optional namespace filter
	Metadata  map[string]string `json:"metadata"`            // Metadata from testenv (optional, provides kubeconfig path)
}

// runMCPServer starts the testenv-lcr MCP server with stdio transport.
func runMCPServer() error {
	server := mcpserver.New(Name, Version)

	// Use framework for standard create/delete tools
	config := engineframework.TestEnvSubengineConfig{
		Name:       Name,
		Version:    Version,
		CreateFunc: createLocalContainerRegistry,
		DeleteFunc: deleteLocalContainerRegistry,
	}

	if err := engineframework.RegisterTestEnvSubengineTools(server, config); err != nil {
		return err
	}

	// Manually register additional tools specific to testenv-lcr
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "create-image-pull-secret",
		Description: "Create an image pull secret in a specific namespace for the local container registry",
	}, handleCreateImagePullSecretTool)

	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "list-image-pull-secrets",
		Description: "List all image pull secrets created by testenv-lcr across all namespaces or in a specific namespace",
	}, handleListImagePullSecretsTool)

	return server.RunDefault()
}

// createLocalContainerRegistry implements the CreateFunc for creating a local container registry.
func createLocalContainerRegistry(ctx context.Context, input engineframework.CreateInput) (*engineframework.TestEnvArtifact, error) {
	log.Printf("Creating local container registry: testID=%s, stage=%s", input.TestID, input.Stage)

	// Redirect stdout to stderr (setup() writes to stdout, but MCP uses stdout for JSON-RPC)
	oldStdout := os.Stdout
	os.Stdout = os.Stderr
	defer func() { os.Stdout = oldStdout }()

	// Read forge.yaml configuration
	config, err := forge.ReadSpec()
	if err != nil {
		return nil, fmt.Errorf("failed to read forge spec: %w", err)
	}

	// Override config with spec values if provided
	if input.Spec != nil {
		if enabled, ok := input.Spec["enabled"].(bool); ok {
			config.LocalContainerRegistry.Enabled = enabled
		}
		if autoPush, ok := input.Spec["autoPushImages"].(bool); ok {
			config.LocalContainerRegistry.AutoPushImages = autoPush
		}
		if namespace, ok := input.Spec["namespace"].(string); ok {
			config.LocalContainerRegistry.Namespace = namespace
		}
		if imagePullSecretNamespaces, ok := input.Spec["imagePullSecretNamespaces"].([]interface{}); ok {
			namespaces := make([]string, 0, len(imagePullSecretNamespaces))
			for _, ns := range imagePullSecretNamespaces {
				if nsStr, ok := ns.(string); ok {
					namespaces = append(namespaces, nsStr)
				}
			}
			config.LocalContainerRegistry.ImagePullSecretNamespaces = namespaces
		}
		if imagePullSecretName, ok := input.Spec["imagePullSecretName"].(string); ok {
			config.LocalContainerRegistry.ImagePullSecretName = imagePullSecretName
		}
	}

	// Check if local container registry is enabled
	if !config.LocalContainerRegistry.Enabled {
		log.Printf("Local container registry is disabled, skipping setup")
		return &engineframework.TestEnvArtifact{
			TestID:           input.TestID,
			Files:            map[string]string{},
			Metadata:         map[string]string{"testenv-lcr.enabled": "false"},
			ManagedResources: []string{},
		}, nil
	}

	// Override kubeconfig path from environment (primary source, from testenv-kind)
	// Fallback to metadata for backward compatibility
	kubeconfigPath := ""
	if envKubeconfig, ok := input.Env["KUBECONFIG"]; ok && envKubeconfig != "" {
		kubeconfigPath = envKubeconfig
		log.Printf("Using KUBECONFIG from environment: %s", kubeconfigPath)
	} else if metadataKubeconfig, ok := input.Metadata["testenv-kind.kubeconfigPath"]; ok && metadataKubeconfig != "" {
		kubeconfigPath = metadataKubeconfig
		log.Printf("Using kubeconfig from metadata (backward compatibility): %s", kubeconfigPath)
	}

	if kubeconfigPath != "" {
		config.Kindenv.KubeconfigPath = kubeconfigPath
	}

	// Override file paths to use tmpDir
	caCrtPath := filepath.Join(input.TmpDir, "ca.crt")
	credentialPath := filepath.Join(input.TmpDir, "registry-credentials.yaml")

	config.LocalContainerRegistry.CaCrtPath = caCrtPath
	config.LocalContainerRegistry.CredentialPath = credentialPath

	// Call the existing setup logic with the overridden config
	if err := setupWithConfig(&config); err != nil {
		return nil, fmt.Errorf("failed to setup local container registry: %w", err)
	}

	// Create Kubernetes client to list created image pull secrets
	cl, err := createKubeClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kube client: %w", err)
	}

	// Prepare files map (relative paths within tmpDir)
	files := map[string]string{
		"testenv-lcr.ca.crt":           "ca.crt",
		"testenv-lcr.credentials.yaml": "registry-credentials.yaml",
	}

	// Prepare metadata
	registryFQDN := fmt.Sprintf("%s.%s.svc.cluster.local:5000", Name, config.LocalContainerRegistry.Namespace)
	metadata := map[string]string{
		"testenv-lcr.registryFQDN":   registryFQDN,
		"testenv-lcr.namespace":      config.LocalContainerRegistry.Namespace,
		"testenv-lcr.caCrtPath":      caCrtPath,
		"testenv-lcr.credentialPath": credentialPath,
		"testenv-lcr.enabled":        "true",
	}

	// Prepare managed resources (for cleanup)
	managedResources := []string{
		caCrtPath,
		credentialPath,
	}

	// Add image pull secret information if they were created
	if len(config.LocalContainerRegistry.ImagePullSecretNamespaces) > 0 {
		secrets, err := ListImagePullSecrets(ctx, cl, "")
		if err != nil {
			log.Printf("Warning: failed to list image pull secrets: %v", err)
		} else {
			secretCount := 0
			for _, secret := range secrets {
				// Add to metadata
				key := fmt.Sprintf("testenv-lcr.imagePullSecret.%d", secretCount)
				metadata[key+".namespace"] = secret.Namespace
				metadata[key+".secretName"] = secret.SecretName
				secretCount++
			}
			if secretCount > 0 {
				metadata["testenv-lcr.imagePullSecretCount"] = fmt.Sprintf("%d", secretCount)
			}
		}
	}

	// Prepare environment variables to export
	env := map[string]string{
		"REGISTRY_FQDN":      registryFQDN,
		"REGISTRY_NAMESPACE": config.LocalContainerRegistry.Namespace,
		"REGISTRY_CA_CERT":   caCrtPath,
	}

	return &engineframework.TestEnvArtifact{
		TestID:           input.TestID,
		Files:            files,
		Metadata:         metadata,
		ManagedResources: managedResources,
		Env:              env,
	}, nil
}

// deleteLocalContainerRegistry implements the DeleteFunc for deleting a local container registry.
func deleteLocalContainerRegistry(ctx context.Context, input engineframework.DeleteInput) error {
	log.Printf("Deleting local container registry: testID=%s", input.TestID)

	// Check if registry was enabled
	if enabled, ok := input.Metadata["testenv-lcr.enabled"]; ok && enabled == "false" {
		log.Printf("Local container registry was disabled, skipping teardown")
		return nil
	}

	// Read forge.yaml configuration
	config, err := forge.ReadSpec()
	if err != nil {
		log.Printf("Warning: failed to read forge spec: %v", err)
		return nil // Best-effort cleanup
	}

	// Check if local container registry is enabled
	if !config.LocalContainerRegistry.Enabled {
		log.Printf("Local container registry is disabled, skipping teardown")
		return nil
	}

	// Override kubeconfig path from metadata (if provided)
	// Note: DeleteInput doesn't have Env field, so we only use metadata
	if kubeconfigPath, ok := input.Metadata["testenv-kind.kubeconfigPath"]; ok {
		config.Kindenv.KubeconfigPath = kubeconfigPath
	}

	// Call the existing teardown logic (best-effort)
	if err := teardown(); err != nil {
		// Log error but don't fail - best effort cleanup
		log.Printf("Warning: failed to teardown local container registry: %v", err)
	}

	return nil
}

// handleCreateImagePullSecretTool handles the "create-image-pull-secret" tool call from MCP clients.
func handleCreateImagePullSecretTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input CreateImagePullSecretInput,
) (*mcp.CallToolResult, any, error) {
	log.Printf("Creating image pull secret: testID=%s, namespace=%s", input.TestID, input.Namespace)

	// Validate inputs
	if result := mcputil.ValidateRequiredWithPrefix("Create image pull secret failed", map[string]string{
		"testID":    input.TestID,
		"namespace": input.Namespace,
	}); result != nil {
		return result, nil, nil
	}

	// Read forge.yaml configuration
	config, err := forge.ReadSpec()
	if err != nil {
		return mcputil.ErrorResult(fmt.Sprintf("Create image pull secret failed: %v", err)), nil, nil
	}

	// Check if local container registry is enabled
	if !config.LocalContainerRegistry.Enabled {
		return mcputil.ErrorResult("Create image pull secret failed: local container registry is disabled"), nil, nil
	}

	// Override kubeconfig path from metadata (if provided by testenv-kind)
	if kubeconfigPath, ok := input.Metadata["testenv-kind.kubeconfigPath"]; ok {
		config.Kindenv.KubeconfigPath = kubeconfigPath
	}

	// Create Kubernetes client
	cl, err := createKubeClient(config)
	if err != nil {
		return mcputil.ErrorResult(fmt.Sprintf("Create image pull secret failed: failed to create kube client: %v", err)), nil, nil
	}

	// Get credential and CA cert from metadata or files
	caCrtPath := input.Metadata["testenv-lcr.caCrtPath"]
	if caCrtPath == "" {
		caCrtPath = config.LocalContainerRegistry.CaCrtPath
	}

	credentialPath := input.Metadata["testenv-lcr.credentialPath"]
	if credentialPath == "" {
		credentialPath = config.LocalContainerRegistry.CredentialPath
	}

	registryFQDN := input.Metadata["testenv-lcr.registryFQDN"]
	if registryFQDN == "" {
		return mcputil.ErrorResult("Create image pull secret failed: missing testenv-lcr.registryFQDN in metadata"), nil, nil
	}

	// Read CA certificate
	caCert, err := os.ReadFile(caCrtPath)
	if err != nil {
		return mcputil.ErrorResult(fmt.Sprintf("Create image pull secret failed: failed to read CA cert: %v", err)), nil, nil
	}

	// Read credentials from file
	credBytes, err := os.ReadFile(credentialPath)
	if err != nil {
		return mcputil.ErrorResult(fmt.Sprintf("Create image pull secret failed: failed to read credentials file: %v", err)), nil, nil
	}

	var credentials Credentials
	if err := yaml.Unmarshal(credBytes, &credentials); err != nil {
		return mcputil.ErrorResult(fmt.Sprintf("Create image pull secret failed: failed to parse credentials: %v", err)), nil, nil
	}

	// Use provided secret name or default
	secretName := input.SecretName
	if secretName == "" {
		secretName = config.LocalContainerRegistry.ImagePullSecretName
	}

	// Create image pull secret
	imagePullSecret := NewImagePullSecret(
		cl,
		secretName,
		registryFQDN,
		credentials.Username,
		credentials.Password,
		caCert,
	)

	secretFullName, err := imagePullSecret.CreateInNamespace(ctx, input.Namespace)
	if err != nil {
		return mcputil.ErrorResult(fmt.Sprintf("Create image pull secret failed: %v", err)), nil, nil
	}

	return mcputil.SuccessResult(fmt.Sprintf("Created image pull secret: %s", secretFullName)), nil, nil
}

// handleListImagePullSecretsTool handles the "list-image-pull-secrets" tool call from MCP clients.
func handleListImagePullSecretsTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListImagePullSecretsInput,
) (*mcp.CallToolResult, any, error) {
	log.Printf("Listing image pull secrets: testID=%s", input.TestID)

	// Validate inputs
	if result := mcputil.ValidateRequiredWithPrefix("List image pull secrets failed", map[string]string{
		"testID": input.TestID,
	}); result != nil {
		return result, nil, nil
	}

	// Read forge.yaml configuration
	config, err := forge.ReadSpec()
	if err != nil {
		return mcputil.ErrorResult(fmt.Sprintf("List image pull secrets failed: %v", err)), nil, nil
	}

	// Override kubeconfig path from metadata (if provided by testenv-kind)
	if kubeconfigPath, ok := input.Metadata["testenv-kind.kubeconfigPath"]; ok {
		config.Kindenv.KubeconfigPath = kubeconfigPath
	}

	// Create Kubernetes client
	cl, err := createKubeClient(config)
	if err != nil {
		return mcputil.ErrorResult(fmt.Sprintf("List image pull secrets failed: failed to create kube client: %v", err)), nil, nil
	}

	// List image pull secrets
	secrets, err := ListImagePullSecrets(ctx, cl, input.Namespace)
	if err != nil {
		return mcputil.ErrorResult(fmt.Sprintf("List image pull secrets failed: %v", err)), nil, nil
	}

	if len(secrets) == 0 {
		return mcputil.SuccessResult("No image pull secrets found"), nil, nil
	}

	// Build response message
	message := fmt.Sprintf("Found %d image pull secret(s):\n", len(secrets))
	for _, secret := range secrets {
		message += fmt.Sprintf("  - %s/%s (created: %v)\n", secret.Namespace, secret.SecretName, secret.CreatedAt)
	}

	return mcputil.SuccessResult(message), nil, nil
}
