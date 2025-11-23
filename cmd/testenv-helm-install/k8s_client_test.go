//go:build unit

package main

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBuildKubectlGetCommand(t *testing.T) {
	tests := []struct {
		name           string
		kubeconfigPath string
		resource       string
		resName        string
		namespace      string
		wantArgs       []string
	}{
		{
			name:           "get ConfigMap",
			kubeconfigPath: "/tmp/kubeconfig",
			resource:       "configmap",
			resName:        "my-config",
			namespace:      "default",
			wantArgs:       []string{"--kubeconfig", "/tmp/kubeconfig", "get", "configmap", "my-config", "-n", "default", "-o", "json"},
		},
		{
			name:           "get Secret",
			kubeconfigPath: "/etc/kubernetes/admin.conf",
			resource:       "secret",
			resName:        "my-secret",
			namespace:      "kube-system",
			wantArgs:       []string{"--kubeconfig", "/etc/kubernetes/admin.conf", "get", "secret", "my-secret", "-n", "kube-system", "-o", "json"},
		},
		{
			name:           "get ConfigMap in custom namespace",
			kubeconfigPath: "/home/user/.kube/config",
			resource:       "configmap",
			resName:        "app-config",
			namespace:      "production",
			wantArgs:       []string{"--kubeconfig", "/home/user/.kube/config", "get", "configmap", "app-config", "-n", "production", "-o", "json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildKubectlGetCommand(tt.kubeconfigPath, tt.resource, tt.resName, tt.namespace)

			if len(args) != len(tt.wantArgs) {
				t.Errorf("buildKubectlGetCommand() returned %d args, want %d", len(args), len(tt.wantArgs))
				t.Errorf("got:  %v", args)
				t.Errorf("want: %v", tt.wantArgs)
				return
			}

			for i := range args {
				if args[i] != tt.wantArgs[i] {
					t.Errorf("buildKubectlGetCommand() args[%d] = %q, want %q", i, args[i], tt.wantArgs[i])
				}
			}
		})
	}
}

func TestParseConfigMapJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantData map[string]string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid ConfigMap JSON",
			jsonData: `{"data":{"key1":"value1","key2":"value2"}}`,
			wantData: map[string]string{"key1": "value1", "key2": "value2"},
			wantErr:  false,
		},
		{
			name:     "ConfigMap with empty data",
			jsonData: `{"data":{}}`,
			wantData: map[string]string{},
			wantErr:  false,
		},
		{
			name:     "ConfigMap with YAML value",
			jsonData: `{"data":{"config.yaml":"server:\n  port: 8080\n"}}`,
			wantData: map[string]string{"config.yaml": "server:\n  port: 8080\n"},
			wantErr:  false,
		},
		{
			name:     "invalid JSON",
			jsonData: `{invalid json}`,
			wantErr:  true,
			errMsg:   "failed to parse",
		},
		{
			name:     "missing data field",
			jsonData: `{"metadata":{"name":"test"}}`,
			wantData: map[string]string{},
			wantErr:  false,
		},
		{
			name:     "empty JSON",
			jsonData: `{}`,
			wantData: map[string]string{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := parseConfigMapJSON(tt.jsonData)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseConfigMapJSON() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("parseConfigMapJSON() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("parseConfigMapJSON() unexpected error: %v", err)
				return
			}

			if len(data) != len(tt.wantData) {
				t.Errorf("parseConfigMapJSON() returned %d keys, want %d", len(data), len(tt.wantData))
			}

			for key, wantValue := range tt.wantData {
				gotValue, ok := data[key]
				if !ok {
					t.Errorf("parseConfigMapJSON() missing key %q", key)
					continue
				}
				if gotValue != wantValue {
					t.Errorf("parseConfigMapJSON() data[%q] = %q, want %q", key, gotValue, wantValue)
				}
			}
		})
	}
}

func TestParseSecretJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantData map[string]string
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid Secret JSON with base64 values",
			jsonData: `{"data":{"username":"` + base64.StdEncoding.EncodeToString([]byte("admin")) + `","password":"` +
				base64.StdEncoding.EncodeToString([]byte("secret123")) + `"}}`,
			wantData: map[string]string{"username": "admin", "password": "secret123"},
			wantErr:  false,
		},
		{
			name:     "Secret with empty data",
			jsonData: `{"data":{}}`,
			wantData: map[string]string{},
			wantErr:  false,
		},
		{
			name: "Secret with YAML config",
			jsonData: `{"data":{"config.yaml":"` +
				base64.StdEncoding.EncodeToString([]byte("database:\n  host: localhost\n")) + `"}}`,
			wantData: map[string]string{"config.yaml": "database:\n  host: localhost\n"},
			wantErr:  false,
		},
		{
			name:     "invalid JSON",
			jsonData: `{invalid}`,
			wantErr:  true,
			errMsg:   "failed to parse",
		},
		{
			name:     "invalid base64 in data",
			jsonData: `{"data":{"key":"not-valid-base64!!!"}}`,
			wantErr:  true,
			errMsg:   "failed to decode base64",
		},
		{
			name:     "missing data field",
			jsonData: `{"metadata":{"name":"test"}}`,
			wantData: map[string]string{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := parseSecretJSON(tt.jsonData)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseSecretJSON() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("parseSecretJSON() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("parseSecretJSON() unexpected error: %v", err)
				return
			}

			if len(data) != len(tt.wantData) {
				t.Errorf("parseSecretJSON() returned %d keys, want %d", len(data), len(tt.wantData))
			}

			for key, wantValue := range tt.wantData {
				gotValue, ok := data[key]
				if !ok {
					t.Errorf("parseSecretJSON() missing key %q", key)
					continue
				}
				if gotValue != wantValue {
					t.Errorf("parseSecretJSON() data[%q] = %q, want %q", key, gotValue, wantValue)
				}
			}
		})
	}
}
