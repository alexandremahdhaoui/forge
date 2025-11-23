package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// buildKubectlGetCommand builds the kubectl get command arguments for fetching a resource.
// Returns the full command arguments including kubeconfig, resource type, name, namespace, and output format.
func buildKubectlGetCommand(kubeconfigPath, resource, resName, namespace string) []string {
	return []string{
		"--kubeconfig", kubeconfigPath,
		"get", resource, resName,
		"-n", namespace,
		"-o", "json",
	}
}

// parseConfigMapJSON parses kubectl JSON output and extracts the .data field.
// Returns a map of key-value pairs from the ConfigMap data.
func parseConfigMapJSON(jsonData string) (map[string]string, error) {
	var resource struct {
		Data map[string]string `json:"data"`
	}

	if err := json.Unmarshal([]byte(jsonData), &resource); err != nil {
		return nil, fmt.Errorf("failed to parse ConfigMap JSON: %w", err)
	}

	// Return empty map if data is nil (instead of nil map)
	if resource.Data == nil {
		return make(map[string]string), nil
	}

	return resource.Data, nil
}

// parseSecretJSON parses kubectl JSON output for a Secret and extracts the .data field.
// Secret data values are base64-encoded, so this function decodes them before returning.
// Returns a map of key-value pairs with decoded values.
func parseSecretJSON(jsonData string) (map[string]string, error) {
	var resource struct {
		Data map[string]string `json:"data"`
	}

	if err := json.Unmarshal([]byte(jsonData), &resource); err != nil {
		return nil, fmt.Errorf("failed to parse Secret JSON: %w", err)
	}

	// Return empty map if data is nil
	if resource.Data == nil {
		return make(map[string]string), nil
	}

	// Decode base64 values
	decodedData := make(map[string]string)
	for key, encodedValue := range resource.Data {
		decoded, err := base64.StdEncoding.DecodeString(encodedValue)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 value for key %q: %w", key, err)
		}
		decodedData[key] = string(decoded)
	}

	return decodedData, nil
}

// fetchConfigMap fetches a ConfigMap by name from the specified namespace.
// Returns the ConfigMap data as a map of key-value pairs.
func fetchConfigMap(kubeconfigPath, namespace, name string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := buildKubectlGetCommand(kubeconfigPath, "configmap", name, namespace)
	cmd := exec.CommandContext(ctx, "kubectl", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("kubectl get configmap timed out after 30 seconds")
		}
		return nil, fmt.Errorf("failed to fetch ConfigMap %s/%s: %w, output: %s", namespace, name, err, string(output))
	}

	return parseConfigMapJSON(string(output))
}

// fetchSecret fetches a Secret by name from the specified namespace.
// Returns the Secret data as a map of key-value pairs with base64-decoded values.
func fetchSecret(kubeconfigPath, namespace, name string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := buildKubectlGetCommand(kubeconfigPath, "secret", name, namespace)
	cmd := exec.CommandContext(ctx, "kubectl", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("kubectl get secret timed out after 30 seconds")
		}
		return nil, fmt.Errorf("failed to fetch Secret %s/%s: %w, output: %s", namespace, name, err, string(output))
	}

	return parseSecretJSON(string(output))
}
