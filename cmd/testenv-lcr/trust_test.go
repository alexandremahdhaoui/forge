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
