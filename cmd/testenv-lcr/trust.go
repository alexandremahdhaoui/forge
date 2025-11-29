package main

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

var errListingKindNodes = errors.New("failed to list Kind nodes")

// listKindNodes returns the list of node names for a Kind cluster.
// It executes "kind get nodes --name {clusterName}" and parses the output.
// Each node name is returned as a separate string in the slice.
func listKindNodes(clusterName string) ([]string, error) {
	cmd := exec.Command("kind", "get", "nodes", "--name", clusterName)
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Join(errListingKindNodes, err)
	}

	// Parse output: one node name per line
	lines := strings.Split(string(output), "\n")
	nodes := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			nodes = append(nodes, trimmed)
		}
	}

	if len(nodes) == 0 {
		return nil, errors.Join(errListingKindNodes, errors.New("no nodes found for cluster "+clusterName))
	}

	return nodes, nil
}

// generateHostsToml generates the hosts.toml configuration content for containerd
// to trust a registry at the given FQDN. The generated configuration tells containerd
// where to find the CA certificate for TLS verification.
//
// Parameters:
//   - registryFQDN: Fully qualified domain name of the registry (e.g., "testenv-lcr.testenv-lcr.svc.cluster.local:5000")
//
// Returns:
//   - A string containing the hosts.toml content in TOML format
func generateHostsToml(registryFQDN string) string {
	const hostsTomlTemplate = `server = "https://%s"

[host."https://%s"]
capabilities = ["pull", "resolve"]
ca = "/etc/containerd/certs.d/%s/ca.crt"
`
	return fmt.Sprintf(hostsTomlTemplate, registryFQDN, registryFQDN, registryFQDN)
}

// configureNode configures containerd on a Kind node to trust the local registry's
// CA certificate. It performs the following steps:
//  1. Creates the containerd certs.d directory for the registry FQDN on the node
//  2. Copies the CA certificate from the host to the node
//  3. Generates and writes the hosts.toml configuration to the node
//  4. Adds DNS entry to /etc/hosts on the node for the registry FQDN
//
// Note: containerd uses inotify to auto-detect file changes, so no restart is needed.
//
// Parameters:
//   - nodeName: Name of the Kind node (e.g., "test-cluster-control-plane")
//   - registryFQDN: Registry FQDN (e.g., "testenv-lcr.testenv-lcr.svc.cluster.local:5000")
//   - caCrtPath: Host path to CA certificate file
//   - clusterIP: ClusterIP of the registry service (for /etc/hosts entry)
//
// Returns:
//   - error if any step fails
func configureNode(nodeName, registryFQDN, caCrtPath, clusterIP string) error {
	// Step 1: Create directory on node for registry certificates
	certsDir := fmt.Sprintf("/etc/containerd/certs.d/%s", registryFQDN)
	mkdirCmd := exec.Command("docker", "exec", nodeName, "mkdir", "-p", certsDir)
	if output, err := mkdirCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create certs directory on node %s: %w\noutput: %s", nodeName, err, string(output))
	}

	// Step 2: Copy CA certificate to node
	destPath := fmt.Sprintf("%s:/etc/containerd/certs.d/%s/ca.crt", nodeName, registryFQDN)
	cpCmd := exec.Command("docker", "cp", caCrtPath, destPath)
	if output, err := cpCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to copy CA cert to node %s: %w\noutput: %s", nodeName, err, string(output))
	}

	// Step 3: Generate hosts.toml content
	hostsTomlContent := generateHostsToml(registryFQDN)

	// Step 4: Write hosts.toml to node via docker exec with stdin
	hostsTomlPath := fmt.Sprintf("/etc/containerd/certs.d/%s/hosts.toml", registryFQDN)
	writeCmd := exec.Command("docker", "exec", "-i", nodeName, "sh", "-c", fmt.Sprintf("cat > %s", hostsTomlPath))
	writeCmd.Stdin = strings.NewReader(hostsTomlContent)
	if output, err := writeCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to write hosts.toml to node %s: %w\noutput: %s", nodeName, err, string(output))
	}

	// Step 5: Add DNS entry to /etc/hosts on the node
	// This is necessary because Kind nodes use host DNS, not Kubernetes CoreDNS,
	// so they cannot resolve cluster-internal FQDNs like service.namespace.svc.cluster.local
	if clusterIP != "" {
		// Extract hostname without port for /etc/hosts entry
		hostname := registryFQDN
		if idx := strings.LastIndex(registryFQDN, ":"); idx != -1 {
			hostname = registryFQDN[:idx]
		}

		hostsEntry := fmt.Sprintf("%s %s", clusterIP, hostname)
		// Check if entry already exists to avoid duplicates
		checkCmd := exec.Command("docker", "exec", nodeName, "grep", "-q", hostname, "/etc/hosts")
		if err := checkCmd.Run(); err != nil {
			// Entry doesn't exist, add it
			addCmd := exec.Command("docker", "exec", nodeName, "sh", "-c",
				fmt.Sprintf("echo '%s' >> /etc/hosts", hostsEntry))
			if output, err := addCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to add /etc/hosts entry on node %s: %w\noutput: %s", nodeName, err, string(output))
			}
			log.Printf("Added /etc/hosts entry on node %s: %s", nodeName, hostsEntry)
		}
	}

	return nil
}

// verifyNodeConfig verifies that the containerd trust configuration was applied
// correctly on a Kind node. It checks that both the CA certificate and hosts.toml
// files exist in the expected location.
//
// Parameters:
//   - nodeName: Name of the Kind node (e.g., "test-cluster-control-plane")
//   - registryFQDN: Registry FQDN (e.g., "testenv-lcr.testenv-lcr.svc.cluster.local:5000")
//
// Returns:
//   - nil if all verification checks pass
//   - error with descriptive message if any verification fails
func verifyNodeConfig(nodeName, registryFQDN string) error {
	certsDir := fmt.Sprintf("/etc/containerd/certs.d/%s", registryFQDN)

	// Check that CA certificate exists
	caCrtPath := fmt.Sprintf("%s/ca.crt", certsDir)
	cmd := exec.Command("docker", "exec", nodeName, "test", "-f", caCrtPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ca.crt not found on node %s at %s: %w", nodeName, caCrtPath, err)
	}

	// Check that hosts.toml exists
	hostsTomlPath := fmt.Sprintf("%s/hosts.toml", certsDir)
	cmd = exec.Command("docker", "exec", nodeName, "test", "-f", hostsTomlPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("hosts.toml not found on node %s at %s: %w", nodeName, hostsTomlPath, err)
	}

	// Check that hosts.toml contains the registry FQDN
	cmd = exec.Command("docker", "exec", nodeName, "grep", registryFQDN, hostsTomlPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("hosts.toml on node %s does not contain registry FQDN %s: %w", nodeName, registryFQDN, err)
	}

	return nil
}

// getServiceClusterIP retrieves the ClusterIP of a Kubernetes service using kubectl.
// It requires KUBECONFIG environment variable to be set or uses the default kubeconfig.
//
// Parameters:
//   - serviceName: Name of the service (e.g., "testenv-lcr")
//   - namespace: Namespace where the service is located
//   - kubeconfigPath: Path to kubeconfig file
//
// Returns:
//   - ClusterIP string on success
//   - error if the service doesn't exist or kubectl fails
func getServiceClusterIP(serviceName, namespace, kubeconfigPath string) (string, error) {
	args := []string{
		"get", "service", serviceName,
		"-n", namespace,
		"-o", "jsonpath={.spec.clusterIP}",
	}
	if kubeconfigPath != "" {
		args = append(args, "--kubeconfig", kubeconfigPath)
	}

	cmd := exec.Command("kubectl", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get ClusterIP for service %s/%s: %w", namespace, serviceName, err)
	}

	clusterIP := strings.TrimSpace(string(output))
	if clusterIP == "" || clusterIP == "None" {
		return "", fmt.Errorf("service %s/%s has no ClusterIP (might be headless)", namespace, serviceName)
	}

	return clusterIP, nil
}

// configureContainerdTrust orchestrates the complete containerd trust configuration
// across all nodes in a Kind cluster. It lists all nodes, then for each node:
// configures the containerd trust settings and verifies the configuration was applied.
//
// Parameters:
//   - clusterName: Name of the Kind cluster
//   - registryFQDN: Registry FQDN (e.g., "testenv-lcr.testenv-lcr.svc.cluster.local:5000")
//   - caCrtPath: Host path to CA certificate file
//   - kubeconfigPath: Path to kubeconfig file for kubectl commands
//
// Returns:
//   - nil if all nodes are successfully configured and verified
//   - error immediately if any node fails (fail-fast behavior)
func configureContainerdTrust(clusterName, registryFQDN, caCrtPath, kubeconfigPath string) error {
	// Step 1: Get all nodes in the cluster
	nodes, err := listKindNodes(clusterName)
	if err != nil {
		return fmt.Errorf("failed to list nodes for cluster %s: %w", clusterName, err)
	}

	// Step 2: listKindNodes already returns an error if no nodes found,
	// but double-check for safety
	if len(nodes) == 0 {
		return fmt.Errorf("no nodes found for cluster %s", clusterName)
	}

	// Step 3: Get the ClusterIP of the registry service
	// Extract service name and namespace from FQDN
	// FQDN format: service.namespace.svc.cluster.local:port
	fqdnWithoutPort := registryFQDN
	if idx := strings.LastIndex(registryFQDN, ":"); idx != -1 {
		fqdnWithoutPort = registryFQDN[:idx]
	}
	parts := strings.SplitN(fqdnWithoutPort, ".", 3)
	if len(parts) < 2 {
		return fmt.Errorf("invalid registry FQDN format: %s", registryFQDN)
	}
	serviceName := parts[0]
	namespace := parts[1]

	clusterIP, err := getServiceClusterIP(serviceName, namespace, kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to get registry ClusterIP: %w", err)
	}
	log.Printf("Registry service ClusterIP: %s", clusterIP)

	// Step 4: Configure and verify each node
	for _, node := range nodes {
		log.Printf("Configuring containerd trust on node %s", node)

		// Configure the node
		if err := configureNode(node, registryFQDN, caCrtPath, clusterIP); err != nil {
			return fmt.Errorf("failed to configure node %s: %w", node, err)
		}

		// Verify the configuration
		if err := verifyNodeConfig(node, registryFQDN); err != nil {
			return fmt.Errorf("failed to verify configuration on node %s: %w", node, err)
		}

		log.Printf("Verified containerd trust on node %s", node)
	}

	log.Printf("Successfully configured containerd trust on all %d nodes", len(nodes))
	return nil
}
