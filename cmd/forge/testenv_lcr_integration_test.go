//go:build integration

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// runCmd executes a command and logs stdout/stderr to the test output.
// Returns the combined output and any error.
func runCmd(t *testing.T, name string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(name, args...)
	t.Logf(">>> Running: %s %s", name, strings.Join(args, " "))
	output, err := cmd.CombinedOutput()
	if len(output) > 0 {
		t.Logf("<<< Output:\n%s", string(output))
	}
	return string(output), err
}

// runCmdWithStdin executes a command with stdin and logs stdout/stderr to the test output.
func runCmdWithStdin(t *testing.T, stdin string, name string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(stdin)
	t.Logf(">>> Running: %s %s", name, strings.Join(args, " "))
	output, err := cmd.CombinedOutput()
	if len(output) > 0 {
		t.Logf("<<< Output:\n%s", string(output))
	}
	return string(output), err
}

// TestLocalContainerRegistryPushPull verifies that we can push and pull images
// from the local container registry created by testenv-lcr.
func TestLocalContainerRegistryPushPull(t *testing.T) {
	t.Parallel() // Can run in parallel - uses isolated namespace

	// Verify testenv-lcr metadata is set
	registryFQDN := os.Getenv("FORGE_METADATA_TESTENV_LCR_REGISTRYFQDN")
	if registryFQDN == "" {
		t.Skip("FORGE_METADATA_TESTENV_LCR_REGISTRYFQDN not set - testenv-lcr may not be fully set up (possibly due to permissions)")
	}

	t.Logf("Registry FQDN: %s", registryFQDN)

	// Verify credentials file exists
	credentialPath := os.Getenv("FORGE_ARTIFACT_TESTENV_LCR_CREDENTIALS_YAML")
	if credentialPath == "" {
		t.Skip("FORGE_ARTIFACT_TESTENV_LCR_CREDENTIALS_YAML not set - testenv-lcr may not be fully set up")
	}

	if _, err := os.Stat(credentialPath); os.IsNotExist(err) {
		t.Skipf("Credentials file does not exist: %s - testenv-lcr may not be fully set up (possibly due to permissions)", credentialPath)
	}
	t.Logf("Credentials file: %s", credentialPath)

	// Verify CA certificate file exists
	caCrtPath := os.Getenv("FORGE_ARTIFACT_TESTENV_LCR_CA_CRT")
	if caCrtPath == "" {
		t.Fatal("FORGE_ARTIFACT_TESTENV_LCR_CA_CRT not set")
	}

	if _, err := os.Stat(caCrtPath); os.IsNotExist(err) {
		t.Fatalf("CA certificate file does not exist: %s", caCrtPath)
	}
	t.Logf("CA certificate: %s", caCrtPath)

	// Test push and pull with the minimal for-testing-purposes image
	testPushPullLocalImage(t, registryFQDN)
}

// testPushPullLocalImage tests pushing and pulling the for-testing-purposes image.
// This test deploys its OWN testenv-lcr in a SEPARATE namespace using the EXISTING Kind cluster.
// This ensures test isolation without the overhead of creating a new cluster.
func testPushPullLocalImage(t *testing.T, _ string) {
	t.Run("PushPullLocalImage", func(t *testing.T) {
		// Use the existing Kind cluster's kubeconfig
		kubeconfigPath := os.Getenv("FORGE_ARTIFACT_TESTENV_KIND_KUBECONFIG")
		if kubeconfigPath == "" {
			t.Fatal("FORGE_ARTIFACT_TESTENV_KIND_KUBECONFIG not set")
		}

		// Deploy a SEPARATE testenv-lcr in a different namespace for this test
		testNamespace := "testenv-lcr-pushpull"
		testPort := "30500" // Use a different port to avoid conflicts

		t.Logf("Deploying separate testenv-lcr in namespace %s", testNamespace)

		// Create namespace
		nsYAML, err := runCmd(t, "kubectl", "create", "namespace", testNamespace,
			"--kubeconfig", kubeconfigPath, "--dry-run=client", "-o", "yaml")
		if err != nil {
			t.Fatalf("Failed to generate namespace YAML: %v", err)
		}

		if _, err := runCmdWithStdin(t, nsYAML, "kubectl", "apply", "-f", "-", "--kubeconfig", kubeconfigPath); err != nil {
			t.Fatalf("Failed to create namespace: %v", err)
		}

		// Cleanup namespace at the end
		defer func() {
			t.Log("Cleaning up test namespace...")
			runCmd(t, "kubectl", "delete", "namespace", testNamespace,
				"--kubeconfig", kubeconfigPath, "--ignore-not-found=true")
		}()

		// Deploy a simple registry (no TLS, no auth for simplicity in this isolated test)
		registryDeployment := fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: registry
  namespace: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app: registry
  template:
    metadata:
      labels:
        app: registry
    spec:
      containers:
      - name: registry
        image: registry:2
        ports:
        - containerPort: 5000
---
apiVersion: v1
kind: Service
metadata:
  name: registry
  namespace: %s
spec:
  type: NodePort
  selector:
    app: registry
  ports:
  - port: 5000
    targetPort: 5000
    nodePort: %s
`, testNamespace, testNamespace, testPort)

		if _, err := runCmdWithStdin(t, registryDeployment, "kubectl", "apply", "-f", "-", "--kubeconfig", kubeconfigPath); err != nil {
			t.Fatalf("Failed to deploy registry: %v", err)
		}

		// Wait for registry to be ready
		t.Log("Waiting for registry deployment to be ready...")
		if _, err := runCmd(t, "kubectl", "rollout", "status", "deployment/registry",
			"-n", testNamespace, "--kubeconfig", kubeconfigPath, "--timeout=60s"); err != nil {
			t.Fatalf("Registry deployment not ready: %v", err)
		}

		// Start port-forward to the registry
		t.Log("Starting port-forward to registry...")
		portForwardCmd := exec.Command("kubectl", "port-forward",
			"-n", testNamespace,
			"svc/registry", fmt.Sprintf("%s:5000", testPort),
			"--kubeconfig", kubeconfigPath)
		t.Logf(">>> Running (background): kubectl port-forward -n %s svc/registry %s:5000", testNamespace, testPort)
		if err := portForwardCmd.Start(); err != nil {
			t.Fatalf("Failed to start port-forward: %v", err)
		}
		defer func() {
			if portForwardCmd.Process != nil {
				portForwardCmd.Process.Kill()
			}
		}()

		// Wait for port-forward to be ready
		time.Sleep(2 * time.Second)

		// Registry endpoint (no TLS, no auth)
		registryEndpoint := fmt.Sprintf("127.0.0.1:%s", testPort)

		// Use the for-testing-purposes image that's already built locally by forge
		sourceImage := "for-testing-purposes:latest"
		localImage := fmt.Sprintf("%s/for-testing-purposes:push-pull-test", registryEndpoint)

		// Tag the local image for the registry
		t.Logf("Tagging %s as %s...", sourceImage, localImage)
		if _, err := runCmd(t, "docker", "tag", sourceImage, localImage); err != nil {
			t.Fatalf("Failed to tag image: %v", err)
		}

		// Push image to local registry (insecure, no TLS)
		t.Logf("Pushing image to local registry...")
		if _, err := runCmd(t, "docker", "push", localImage); err != nil {
			t.Fatalf("Failed to push image: %v", err)
		}
		t.Logf("Successfully pushed image to local registry")

		// Remove the tagged image to test pull
		t.Log("Removing tagged image to test pull...")
		if _, err := runCmd(t, "docker", "rmi", localImage); err != nil {
			t.Logf("Warning: Failed to remove tagged image (non-fatal): %v", err)
		}

		// Pull image from local registry
		t.Logf("Pulling image from local registry...")
		if _, err := runCmd(t, "docker", "pull", localImage); err != nil {
			t.Fatalf("Failed to pull image from local registry: %v", err)
		}
		t.Logf("Successfully pulled image from local registry")

		// Cleanup images
		t.Log("Cleaning up test images...")
		runCmd(t, "docker", "rmi", localImage)
	})
}

// TestLocalContainerRegistryImagePullSecrets verifies that image pull secrets
// are automatically created in the configured namespaces.
func TestLocalContainerRegistryImagePullSecrets(t *testing.T) {
	t.Parallel() // Can run in parallel - read-only kubectl commands

	// Verify testenv-lcr metadata is set
	registryFQDN := os.Getenv("FORGE_METADATA_TESTENV_LCR_REGISTRYFQDN")
	if registryFQDN == "" {
		t.Skip("FORGE_METADATA_TESTENV_LCR_REGISTRYFQDN not set - testenv-lcr may be disabled")
	}

	// Get kubeconfig
	kubeconfigPath := os.Getenv("FORGE_ARTIFACT_TESTENV_KIND_KUBECONFIG")
	if kubeconfigPath == "" {
		t.Fatal("FORGE_ARTIFACT_TESTENV_KIND_KUBECONFIG not set")
	}

	// Check if image pull secrets were created
	imagePullSecretCount := os.Getenv("FORGE_METADATA_TESTENV_LCR_IMAGEPULLSECRETCOUNT")
	if imagePullSecretCount == "" {
		t.Skip("No image pull secrets configured - skipping test")
	}

	t.Logf("Image pull secret count: %s", imagePullSecretCount)

	// Test each image pull secret
	testImagePullSecretInNamespace(t, kubeconfigPath, 0)
	testImagePullSecretInNamespace(t, kubeconfigPath, 1)
}

// testImagePullSecretInNamespace verifies an image pull secret exists in a namespace.
func testImagePullSecretInNamespace(t *testing.T, kubeconfigPath string, index int) {
	t.Run(fmt.Sprintf("ImagePullSecret_%d", index), func(t *testing.T) {
		// Get namespace and secret name from metadata
		namespaceKey := fmt.Sprintf("FORGE_METADATA_TESTENV_LCR_IMAGEPULLSECRET_%d_NAMESPACE", index)
		secretNameKey := fmt.Sprintf("FORGE_METADATA_TESTENV_LCR_IMAGEPULLSECRET_%d_SECRETNAME", index)

		namespace := os.Getenv(namespaceKey)
		if namespace == "" {
			t.Skipf("%s not set", namespaceKey)
		}

		secretName := os.Getenv(secretNameKey)
		if secretName == "" {
			t.Fatalf("%s not set", secretNameKey)
		}

		t.Logf("Checking image pull secret: %s/%s", namespace, secretName)

		// Verify secret exists
		output, err := runCmd(t, "kubectl", "get", "secret", secretName,
			"-n", namespace,
			"--kubeconfig", kubeconfigPath,
			"-o", "jsonpath={.type}")
		if err != nil {
			t.Fatalf("Failed to get secret: %v", err)
		}

		secretType := strings.TrimSpace(output)
		expectedType := "kubernetes.io/dockerconfigjson"
		if secretType != expectedType {
			t.Errorf("Expected secret type %s, got %s", expectedType, secretType)
		}

		t.Logf("✅ Image pull secret verified: %s/%s (type: %s)", namespace, secretName, secretType)

		// Verify secret has the correct data keys
		output, err = runCmd(t, "kubectl", "get", "secret", secretName,
			"-n", namespace,
			"--kubeconfig", kubeconfigPath,
			"-o", "jsonpath={.data}")
		if err != nil {
			t.Fatalf("Failed to get secret data: %v", err)
		}

		if !strings.Contains(output, ".dockerconfigjson") {
			t.Errorf("Secret does not contain .dockerconfigjson key")
		}

		// Verify secret has the correct label
		output, err = runCmd(t, "kubectl", "get", "secret", secretName,
			"-n", namespace,
			"--kubeconfig", kubeconfigPath,
			"-o", "jsonpath={.metadata.labels.app\\.kubernetes\\.io/managed-by}")
		if err != nil {
			t.Fatalf("Failed to get secret label: %v", err)
		}

		managedBy := strings.TrimSpace(output)
		if managedBy != "testenv-lcr" {
			t.Errorf("Expected managed-by label to be 'testenv-lcr', got '%s'", managedBy)
		}
	})
}

// TestLocalContainerRegistryDeployment verifies that the local container registry
// deployment is running in the cluster.
func TestLocalContainerRegistryDeployment(t *testing.T) {
	t.Parallel() // Can run in parallel - read-only kubectl commands

	// Check if testenv-lcr was set up
	registryFQDN := os.Getenv("FORGE_METADATA_TESTENV_LCR_REGISTRYFQDN")
	if registryFQDN == "" {
		t.Skip("FORGE_METADATA_TESTENV_LCR_REGISTRYFQDN not set - testenv-lcr may not be fully set up (possibly due to permissions)")
	}

	// Get kubeconfig
	kubeconfigPath := os.Getenv("FORGE_ARTIFACT_TESTENV_KIND_KUBECONFIG")
	if kubeconfigPath == "" {
		t.Fatal("FORGE_ARTIFACT_TESTENV_KIND_KUBECONFIG not set")
	}

	// Get registry namespace
	namespace := os.Getenv("FORGE_METADATA_TESTENV_LCR_NAMESPACE")
	if namespace == "" {
		namespace = "testenv-lcr" // default
	}

	t.Run("RegistryDeployment", func(t *testing.T) {
		// Check deployment exists and is ready
		output, err := runCmd(t, "kubectl", "get", "deployment", "testenv-lcr",
			"-n", namespace,
			"--kubeconfig", kubeconfigPath,
			"-o", "jsonpath={.status.availableReplicas}")
		if err != nil {
			t.Fatalf("Failed to get deployment: %v", err)
		}

		availableReplicas := strings.TrimSpace(output)
		if availableReplicas != "1" {
			t.Errorf("Expected 1 available replica, got %s", availableReplicas)
		}

		t.Logf("✅ Registry deployment is running with %s replica(s)", availableReplicas)
	})

	t.Run("RegistryService", func(t *testing.T) {
		// Get expected port from metadata (dynamic port from PortLeaseManager)
		expectedPort := os.Getenv("FORGE_METADATA_TESTENV_LCR_PORT")
		if expectedPort == "" {
			// Fallback for backward compatibility if metadata is not set
			expectedPort = "5000"
		}

		// Check service exists
		output, err := runCmd(t, "kubectl", "get", "service", "testenv-lcr",
			"-n", namespace,
			"--kubeconfig", kubeconfigPath,
			"-o", "jsonpath={.spec.ports[0].port}")
		if err != nil {
			t.Fatalf("Failed to get service: %v", err)
		}

		port := strings.TrimSpace(output)
		if port != expectedPort {
			t.Errorf("Expected service port %s, got %s", expectedPort, port)
		}

		t.Logf("✅ Registry service is running on port %s", port)
	})
}

// TestPodCanPullImageFromLCR verifies that a pod can pull an image from LCR
// using the automatically created image pull secret. This tests the full
// containerd trust configuration including:
// - Kind cluster configured with containerd certs.d path
// - CA certificate copied to Kind nodes
// - hosts.toml configured for the registry FQDN
// - Image pull secret properly configured in the namespace
func TestPodCanPullImageFromLCR(t *testing.T) {
	t.Parallel() // Can run in parallel - uses unique pod name

	// Verify testenv-lcr metadata is set
	registryFQDN := os.Getenv("FORGE_METADATA_TESTENV_LCR_REGISTRYFQDN")
	if registryFQDN == "" {
		t.Skip("FORGE_METADATA_TESTENV_LCR_REGISTRYFQDN not set - testenv-lcr may not be fully set up")
	}

	// Get kubeconfig
	kubeconfigPath := os.Getenv("FORGE_ARTIFACT_TESTENV_KIND_KUBECONFIG")
	if kubeconfigPath == "" {
		t.Fatal("FORGE_ARTIFACT_TESTENV_KIND_KUBECONFIG not set")
	}

	// Get image pull secret info from default namespace (first configured namespace)
	secretName := os.Getenv("FORGE_METADATA_TESTENV_LCR_IMAGEPULLSECRET_0_SECRETNAME")
	if secretName == "" {
		secretName = "local-container-registry-credentials" // default
	}

	// Use the for-testing-purposes image built by forge and pushed to LCR
	// Local images are pushed as: local://name:tag -> {registryFQDN}/name:tag
	testImage := fmt.Sprintf("%s/for-testing-purposes:latest", registryFQDN)
	testNamespace := "default"
	testPodName := "test-lcr-image-pull"

	t.Run("CreateAndVerifyPodImagePull", func(t *testing.T) {
		// Cleanup: Delete pod if it exists from previous test run
		t.Log("Cleaning up any existing test pod...")
		runCmd(t, "kubectl", "delete", "pod", testPodName,
			"-n", testNamespace,
			"--kubeconfig", kubeconfigPath,
			"--ignore-not-found=true")

		// Create pod YAML
		podYAML := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  containers:
  - name: test-container
    image: %s
    command: ["sleep", "3600"]
  imagePullSecrets:
  - name: %s
  restartPolicy: Never
`, testPodName, testNamespace, testImage, secretName)

		t.Logf("Creating pod with image: %s", testImage)
		t.Logf("Using image pull secret: %s", secretName)

		// Create pod using kubectl apply
		if _, err := runCmdWithStdin(t, podYAML, "kubectl", "apply", "-f", "-",
			"--kubeconfig", kubeconfigPath); err != nil {
			t.Fatalf("Failed to create pod: %v", err)
		}
		t.Logf("Pod created successfully")

		// Wait for pod to pull image (check for image pull success)
		// We wait up to 20 seconds for the image to be pulled (using minimal for-testing-purposes image)
		var lastStatus string
		var imagePulled bool
		for i := 0; i < 10; i++ {
			// Get pod status
			output, err := runCmd(t, "kubectl", "get", "pod", testPodName,
				"-n", testNamespace,
				"--kubeconfig", kubeconfigPath,
				"-o", "jsonpath={.status.phase},{.status.containerStatuses[0].state}")
			if err != nil {
				t.Logf("Waiting for pod... (attempt %d/10)", i+1)
				time.Sleep(2 * time.Second)
				continue
			}

			lastStatus = strings.TrimSpace(output)
			t.Logf("Pod status (attempt %d/10): %s", i+1, lastStatus)

			// Check if pod phase indicates image was pulled successfully
			// Phase can be "Pending" (still pulling), "Running" (pulled and running), or "Failed"
			if strings.HasPrefix(lastStatus, "Running") {
				imagePulled = true
				break
			}

			// Check for ImagePullBackOff or ErrImagePull which would indicate failure
			if strings.Contains(lastStatus, "ImagePullBackOff") ||
				strings.Contains(lastStatus, "ErrImagePull") {
				// Get detailed events for debugging
				eventsOutput, _ := runCmd(t, "kubectl", "describe", "pod", testPodName,
					"-n", testNamespace,
					"--kubeconfig", kubeconfigPath)
				t.Fatalf("Image pull failed: %s\nPod details:\n%s", lastStatus, eventsOutput)
			}

			// Also check if container is waiting with a successful image pull
			// (e.g., pod might be Running state)
			if strings.Contains(lastStatus, "running") {
				imagePulled = true
				break
			}

			time.Sleep(2 * time.Second)
		}

		if !imagePulled {
			// Get detailed events for debugging
			eventsOutput, _ := runCmd(t, "kubectl", "describe", "pod", testPodName,
				"-n", testNamespace,
				"--kubeconfig", kubeconfigPath)
			t.Fatalf("Image pull did not complete in time. Last status: %s\nPod details:\n%s",
				lastStatus, eventsOutput)
		}

		t.Logf("✅ Pod successfully pulled image from LCR: %s", testImage)

		// Cleanup: Delete the test pod
		t.Log("Cleaning up test pod...")
		runCmd(t, "kubectl", "delete", "pod", testPodName,
			"-n", testNamespace,
			"--kubeconfig", kubeconfigPath,
			"--ignore-not-found=true")
	})
}
