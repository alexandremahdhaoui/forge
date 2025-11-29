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

// TestLocalContainerRegistryPushPull verifies that we can push and pull images
// from the local container registry created by testenv-lcr.
func TestLocalContainerRegistryPushPull(t *testing.T) {
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

	// Test push and pull with a minimal alpine image
	testPushPullAlpineImage(t, registryFQDN)
}

// testPushPullAlpineImage tests pushing and pulling a minimal alpine image.
func testPushPullAlpineImage(t *testing.T, registryFQDN string) {
	t.Run("PushPullAlpineImage", func(t *testing.T) {
		// Get kubeconfig for port-forward
		kubeconfigPath := os.Getenv("FORGE_ARTIFACT_TESTENV_KIND_KUBECONFIG")
		if kubeconfigPath == "" {
			t.Fatal("FORGE_ARTIFACT_TESTENV_KIND_KUBECONFIG not set")
		}

		// Get registry namespace
		namespace := os.Getenv("FORGE_METADATA_TESTENV_LCR_NAMESPACE")
		if namespace == "" {
			namespace = "testenv-lcr" // default
		}

		// Get credentials path
		credentialPath := os.Getenv("FORGE_ARTIFACT_TESTENV_LCR_CREDENTIALS_YAML")
		if credentialPath == "" {
			t.Fatal("FORGE_ARTIFACT_TESTENV_LCR_CREDENTIALS_YAML not set")
		}

		// Establish port-forward to the registry
		t.Log("Establishing port-forward to registry...")
		portForwardCmd := exec.Command("kubectl", "port-forward",
			"-n", namespace,
			"svc/testenv-lcr", "5000:5000",
			"--kubeconfig", kubeconfigPath)

		// Start port-forward in background
		if err := portForwardCmd.Start(); err != nil {
			t.Fatalf("Failed to start port-forward: %v", err)
		}
		defer func() {
			if portForwardCmd.Process != nil {
				portForwardCmd.Process.Kill()
			}
		}()

		// Wait a bit for port-forward to establish
		t.Log("Waiting for port-forward to be ready...")
		time.Sleep(2 * time.Second)

		// Log in to the registry
		t.Log("Logging in to registry...")
		// Read credentials
		credBytes, err := os.ReadFile(credentialPath)
		if err != nil {
			t.Fatalf("Failed to read credentials: %v", err)
		}

		// Parse credentials (simple YAML parsing - look for username: and password: lines)
		var creds struct {
			Username string
			Password string
		}
		lines := strings.Split(string(credBytes), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "username:") {
				creds.Username = strings.TrimSpace(strings.TrimPrefix(line, "username:"))
			} else if strings.HasPrefix(line, "password:") {
				creds.Password = strings.TrimSpace(strings.TrimPrefix(line, "password:"))
			}
		}

		if creds.Username == "" || creds.Password == "" {
			t.Fatalf("Failed to parse credentials from file")
		}

		// Login using stdin for password
		loginCmd := exec.Command("docker", "login", registryFQDN, "--username", creds.Username, "--password-stdin")
		loginCmd.Stdin = strings.NewReader(creds.Password)
		output, err := loginCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to login to registry: %v\nOutput: %s", err, string(output))
		}
		t.Logf("Successfully logged in to registry")

		// Pull alpine image from public registry
		t.Log("Pulling alpine:latest from public registry...")
		cmd := exec.Command("docker", "pull", "alpine:latest")
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to pull alpine:latest: %v\nOutput: %s", err, string(output))
		}
		t.Logf("Successfully pulled alpine:latest")

		// Tag image for local registry
		localImage := fmt.Sprintf("%s/alpine:test", registryFQDN)
		t.Logf("Tagging image as %s...", localImage)
		cmd = exec.Command("docker", "tag", "alpine:latest", localImage)
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to tag image: %v\nOutput: %s", err, string(output))
		}

		// Push image to local registry
		t.Logf("Pushing image to local registry...")
		cmd = exec.Command("docker", "push", localImage)
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to push image: %v\nOutput: %s", err, string(output))
		}
		t.Logf("Successfully pushed image to local registry")

		// Remove local image to test pull
		t.Log("Removing local image to test pull...")
		cmd = exec.Command("docker", "rmi", localImage)
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Logf("Warning: Failed to remove local image (non-fatal): %v", err)
		}

		// Pull image from local registry
		t.Logf("Pulling image from local registry...")
		cmd = exec.Command("docker", "pull", localImage)
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to pull image from local registry: %v\nOutput: %s", err, string(output))
		}
		t.Logf("Successfully pulled image from local registry")

		// Cleanup
		t.Log("Cleaning up test images...")
		exec.Command("docker", "rmi", localImage).Run()
		exec.Command("docker", "rmi", "alpine:latest").Run()
		exec.Command("docker", "logout", registryFQDN).Run()
	})
}

// TestLocalContainerRegistryImagePullSecrets verifies that image pull secrets
// are automatically created in the configured namespaces.
func TestLocalContainerRegistryImagePullSecrets(t *testing.T) {
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
		cmd := exec.Command("kubectl", "get", "secret", secretName,
			"-n", namespace,
			"--kubeconfig", kubeconfigPath,
			"-o", "jsonpath={.type}")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to get secret: %v\nOutput: %s", err, string(output))
		}

		secretType := strings.TrimSpace(string(output))
		expectedType := "kubernetes.io/dockerconfigjson"
		if secretType != expectedType {
			t.Errorf("Expected secret type %s, got %s", expectedType, secretType)
		}

		t.Logf("✅ Image pull secret verified: %s/%s (type: %s)", namespace, secretName, secretType)

		// Verify secret has the correct data keys
		cmd = exec.Command("kubectl", "get", "secret", secretName,
			"-n", namespace,
			"--kubeconfig", kubeconfigPath,
			"-o", "jsonpath={.data}")
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to get secret data: %v\nOutput: %s", err, string(output))
		}

		if !strings.Contains(string(output), ".dockerconfigjson") {
			t.Errorf("Secret does not contain .dockerconfigjson key")
		}

		// Verify secret has the correct label
		cmd = exec.Command("kubectl", "get", "secret", secretName,
			"-n", namespace,
			"--kubeconfig", kubeconfigPath,
			"-o", "jsonpath={.metadata.labels.app\\.kubernetes\\.io/managed-by}")
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to get secret label: %v\nOutput: %s", err, string(output))
		}

		managedBy := strings.TrimSpace(string(output))
		if managedBy != "testenv-lcr" {
			t.Errorf("Expected managed-by label to be 'testenv-lcr', got '%s'", managedBy)
		}
	})
}

// TestLocalContainerRegistryDeployment verifies that the local container registry
// deployment is running in the cluster.
func TestLocalContainerRegistryDeployment(t *testing.T) {
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
		cmd := exec.Command("kubectl", "get", "deployment", "testenv-lcr",
			"-n", namespace,
			"--kubeconfig", kubeconfigPath,
			"-o", "jsonpath={.status.availableReplicas}")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to get deployment: %v\nOutput: %s", err, string(output))
		}

		availableReplicas := strings.TrimSpace(string(output))
		if availableReplicas != "1" {
			t.Errorf("Expected 1 available replica, got %s", availableReplicas)
		}

		t.Logf("✅ Registry deployment is running with %s replica(s)", availableReplicas)
	})

	t.Run("RegistryService", func(t *testing.T) {
		// Check service exists
		cmd := exec.Command("kubectl", "get", "service", "testenv-lcr",
			"-n", namespace,
			"--kubeconfig", kubeconfigPath,
			"-o", "jsonpath={.spec.ports[0].port}")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to get service: %v\nOutput: %s", err, string(output))
		}

		port := strings.TrimSpace(string(output))
		if port != "5000" {
			t.Errorf("Expected service port 5000, got %s", port)
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

	// Use the alpine image that should have been pushed to LCR by testenv-lcr
	// The full image path is preserved when pushing to LCR:
	// docker.io/library/alpine:3.18 -> {registryFQDN}/docker.io/library/alpine:3.18
	testImage := fmt.Sprintf("%s/docker.io/library/alpine:3.18", registryFQDN)
	testNamespace := "default"
	testPodName := "test-lcr-image-pull"

	t.Run("CreateAndVerifyPodImagePull", func(t *testing.T) {
		// Cleanup: Delete pod if it exists from previous test run
		cleanupCmd := exec.Command("kubectl", "delete", "pod", testPodName,
			"-n", testNamespace,
			"--kubeconfig", kubeconfigPath,
			"--ignore-not-found=true")
		cleanupCmd.Run() // Ignore errors from cleanup

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
		applyCmd := exec.Command("kubectl", "apply", "-f", "-",
			"--kubeconfig", kubeconfigPath)
		applyCmd.Stdin = strings.NewReader(podYAML)
		output, err := applyCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to create pod: %v\nOutput: %s", err, string(output))
		}
		t.Logf("Pod created successfully")

		// Wait for pod to pull image (check for image pull success)
		// We wait up to 60 seconds for the image to be pulled
		var lastStatus string
		var imagePulled bool
		for i := 0; i < 30; i++ {
			// Get pod status
			statusCmd := exec.Command("kubectl", "get", "pod", testPodName,
				"-n", testNamespace,
				"--kubeconfig", kubeconfigPath,
				"-o", "jsonpath={.status.phase},{.status.containerStatuses[0].state}")
			statusOutput, err := statusCmd.CombinedOutput()
			if err != nil {
				t.Logf("Waiting for pod... (attempt %d/30)", i+1)
				time.Sleep(2 * time.Second)
				continue
			}

			lastStatus = strings.TrimSpace(string(statusOutput))
			t.Logf("Pod status (attempt %d/30): %s", i+1, lastStatus)

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
				eventsCmd := exec.Command("kubectl", "describe", "pod", testPodName,
					"-n", testNamespace,
					"--kubeconfig", kubeconfigPath)
				eventsOutput, _ := eventsCmd.CombinedOutput()
				t.Fatalf("Image pull failed: %s\nPod details:\n%s", lastStatus, string(eventsOutput))
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
			eventsCmd := exec.Command("kubectl", "describe", "pod", testPodName,
				"-n", testNamespace,
				"--kubeconfig", kubeconfigPath)
			eventsOutput, _ := eventsCmd.CombinedOutput()
			t.Fatalf("Image pull did not complete in time. Last status: %s\nPod details:\n%s",
				lastStatus, string(eventsOutput))
		}

		t.Logf("✅ Pod successfully pulled image from LCR: %s", testImage)

		// Cleanup: Delete the test pod
		deleteCmd := exec.Command("kubectl", "delete", "pod", testPodName,
			"-n", testNamespace,
			"--kubeconfig", kubeconfigPath,
			"--ignore-not-found=true")
		deleteCmd.Run() // Best effort cleanup
	})
}
