//go:build unit

package main

import (
	"strings"
	"testing"
)

func TestGenerateHostsToml(t *testing.T) {
	tests := []struct {
		name         string
		registryFQDN string
		wantContains []string
	}{
		{
			name:         "standard registry FQDN",
			registryFQDN: "testenv-lcr.testenv-lcr.svc.cluster.local:5000",
			wantContains: []string{
				`server = "https://testenv-lcr.testenv-lcr.svc.cluster.local:5000"`,
				`[host."https://testenv-lcr.testenv-lcr.svc.cluster.local:5000"]`,
				`capabilities = ["pull", "resolve"]`,
				`ca = "/etc/containerd/certs.d/testenv-lcr.testenv-lcr.svc.cluster.local:5000/ca.crt"`,
			},
		},
		{
			name:         "custom namespace",
			registryFQDN: "my-registry.custom-ns.svc.cluster.local:5000",
			wantContains: []string{
				`server = "https://my-registry.custom-ns.svc.cluster.local:5000"`,
				`[host."https://my-registry.custom-ns.svc.cluster.local:5000"]`,
				`ca = "/etc/containerd/certs.d/my-registry.custom-ns.svc.cluster.local:5000/ca.crt"`,
			},
		},
		{
			name:         "different port",
			registryFQDN: "registry.ns.svc.cluster.local:8443",
			wantContains: []string{
				`server = "https://registry.ns.svc.cluster.local:8443"`,
				`[host."https://registry.ns.svc.cluster.local:8443"]`,
				`ca = "/etc/containerd/certs.d/registry.ns.svc.cluster.local:8443/ca.crt"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateHostsToml(tt.registryFQDN)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("generateHostsToml(%q) missing expected content:\nwant: %q\ngot:\n%s",
						tt.registryFQDN, want, result)
				}
			}
		})
	}
}

func TestGenerateHostsTomlFormat(t *testing.T) {
	// Test that the output is valid TOML-like format
	result := generateHostsToml("registry.ns.svc.cluster.local:5000")

	// Check it starts with server line
	if !strings.HasPrefix(result, "server = ") {
		t.Errorf("generateHostsToml should start with 'server = ', got: %s", result)
	}

	// Check it contains [host.] section
	if !strings.Contains(result, "[host.") {
		t.Errorf("generateHostsToml should contain [host.] section, got: %s", result)
	}

	// Check capabilities array format
	if !strings.Contains(result, `["pull", "resolve"]`) {
		t.Errorf("generateHostsToml should contain capabilities array, got: %s", result)
	}
}

func TestGenerateHostsTomlEmptyInput(t *testing.T) {
	// Even with empty input, function should return structured content
	// (though this would be an error case in practice)
	result := generateHostsToml("")

	// Should still have the structure, just with empty registry
	if !strings.Contains(result, "server = ") {
		t.Errorf("generateHostsToml with empty input should still have structure")
	}
}

func TestGenerateHostsToml_IncludesDynamicPort(t *testing.T) {
	// Test that generateHostsToml correctly includes dynamic NodePort in all paths.
	// This is critical for the NodePort + port-forward architecture where the
	// same dynamic port (30000-32767 range) is used everywhere.
	tests := []struct {
		name         string
		registryFQDN string
		port         string
	}{
		{
			name:         "NodePort lower bound",
			registryFQDN: "testenv-lcr.testenv-lcr.svc.cluster.local:30000",
			port:         "30000",
		},
		{
			name:         "NodePort typical value",
			registryFQDN: "testenv-lcr.testenv-lcr.svc.cluster.local:30123",
			port:         "30123",
		},
		{
			name:         "NodePort upper bound",
			registryFQDN: "testenv-lcr.testenv-lcr.svc.cluster.local:32767",
			port:         "32767",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateHostsToml(tt.registryFQDN)

			// Verify the port is included in server URL
			expectedServer := `server = "https://` + tt.registryFQDN + `"`
			if !strings.Contains(result, expectedServer) {
				t.Errorf("generateHostsToml(%q) missing server with port:\nwant contains: %q\ngot:\n%s",
					tt.registryFQDN, expectedServer, result)
			}

			// Verify the port is included in host section
			expectedHost := `[host."https://` + tt.registryFQDN + `"]`
			if !strings.Contains(result, expectedHost) {
				t.Errorf("generateHostsToml(%q) missing host section with port:\nwant contains: %q\ngot:\n%s",
					tt.registryFQDN, expectedHost, result)
			}

			// Verify the port is included in certs.d path
			expectedCertsPath := `/etc/containerd/certs.d/` + tt.registryFQDN + `/ca.crt`
			if !strings.Contains(result, expectedCertsPath) {
				t.Errorf("generateHostsToml(%q) missing certs.d path with port:\nwant contains: %q\ngot:\n%s",
					tt.registryFQDN, expectedCertsPath, result)
			}
		})
	}
}

// TestConfigureNode_SignatureNoClusterIP verifies that configureNode no longer
// requires a clusterIP parameter. The DNS leak mechanism means Kind nodes
// resolve the registry FQDN to 127.0.0.1 via Docker DNS forwarding to the host's
// /etc/hosts, and kube-proxy intercepts traffic to 127.0.0.1:NodePort.
func TestConfigureNode_SignatureNoClusterIP(t *testing.T) {
	// This test documents the API change: configureNode now takes only 3 parameters:
	// (nodeName, registryFQDN, caCrtPath) instead of 4 (with clusterIP).
	//
	// We cannot actually call configureNode in unit tests as it requires Docker,
	// but we verify the function signature by assigning it to a function variable
	// with the expected signature.
	var fn func(nodeName, registryFQDN, caCrtPath string) error
	fn = configureNode
	_ = fn // Prevent unused variable error

	// If this test compiles, the signature is correct (no clusterIP parameter)
}

// TestConfigureContainerdTrust_SignatureNoKubeconfigPath verifies that
// configureContainerdTrust no longer requires a kubeconfigPath parameter.
// The kubeconfigPath was only used to retrieve the ClusterIP, which is no longer needed.
func TestConfigureContainerdTrust_SignatureNoKubeconfigPath(t *testing.T) {
	// This test documents the API change: configureContainerdTrust now takes only 3 parameters:
	// (clusterName, registryFQDN, caCrtPath) instead of 4 (with kubeconfigPath).
	var fn func(clusterName, registryFQDN, caCrtPath string) error
	fn = configureContainerdTrust
	_ = fn // Prevent unused variable error

	// If this test compiles, the signature is correct (no kubeconfigPath parameter)
}
