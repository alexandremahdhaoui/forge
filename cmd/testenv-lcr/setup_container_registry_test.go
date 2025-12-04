//go:build unit

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
	"context"
	"strings"
	"testing"
	"text/template"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSetDynamicPort(t *testing.T) {
	tests := []struct {
		name     string
		port     int32
		wantPort int32
	}{
		{
			name:     "set port to 30123",
			port:     30123,
			wantPort: 30123,
		},
		{
			name:     "set port to 32000",
			port:     32000,
			wantPort: 32000,
		},
		{
			name:     "set port to min nodeport range",
			port:     30000,
			wantPort: 30000,
		},
		{
			name:     "set port to max nodeport range",
			port:     32767,
			wantPort: 32767,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ContainerRegistry{}
			r.SetDynamicPort(tt.port)

			if r.dynamicPort != tt.wantPort {
				t.Errorf("SetDynamicPort(%d) = %d, want %d", tt.port, r.dynamicPort, tt.wantPort)
			}
		})
	}
}

func TestPort_ReturnsDynamicPort(t *testing.T) {
	tests := []struct {
		name        string
		dynamicPort int32
		wantPort    int32
	}{
		{
			name:        "returns dynamic port when set",
			dynamicPort: 30123,
			wantPort:    30123,
		},
		{
			name:        "returns default port when not set",
			dynamicPort: 0,
			wantPort:    containerRegistryPort, // 5000
		},
		{
			name:        "returns custom dynamic port",
			dynamicPort: 31500,
			wantPort:    31500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ContainerRegistry{}
			if tt.dynamicPort != 0 {
				r.SetDynamicPort(tt.dynamicPort)
			}

			got := r.Port()
			if got != tt.wantPort {
				t.Errorf("Port() = %d, want %d", got, tt.wantPort)
			}
		})
	}
}

func TestCreateService_NodePort(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	cl := fake.NewClientBuilder().WithScheme(scheme).Build()

	r := &ContainerRegistry{
		client:    cl,
		namespace: "test-ns",
	}
	r.SetDynamicPort(30123)

	labels := map[string]string{"app": Name}
	ctx := context.Background()

	err := r.createService(ctx, labels)
	if err != nil {
		t.Fatalf("createService() error = %v", err)
	}

	// Retrieve the created service
	svc := &corev1.Service{}
	err = cl.Get(ctx, client.ObjectKey{Namespace: "test-ns", Name: Name}, svc)
	if err != nil {
		t.Fatalf("Failed to get created service: %v", err)
	}

	// Verify service type is NodePort
	if svc.Spec.Type != corev1.ServiceTypeNodePort {
		t.Errorf("Service type = %v, want %v", svc.Spec.Type, corev1.ServiceTypeNodePort)
	}
}

func TestCreateService_AllPortsSame(t *testing.T) {
	tests := []struct {
		name        string
		dynamicPort int32
	}{
		{
			name:        "ports set to 30123",
			dynamicPort: 30123,
		},
		{
			name:        "ports set to 31500",
			dynamicPort: 31500,
		},
		{
			name:        "ports set to 32000",
			dynamicPort: 32000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)

			cl := fake.NewClientBuilder().WithScheme(scheme).Build()

			r := &ContainerRegistry{
				client:    cl,
				namespace: "test-ns",
			}
			r.SetDynamicPort(tt.dynamicPort)

			labels := map[string]string{"app": Name}
			ctx := context.Background()

			err := r.createService(ctx, labels)
			if err != nil {
				t.Fatalf("createService() error = %v", err)
			}

			// Retrieve the created service
			svc := &corev1.Service{}
			err = cl.Get(ctx, client.ObjectKey{Namespace: "test-ns", Name: Name}, svc)
			if err != nil {
				t.Fatalf("Failed to get created service: %v", err)
			}

			if len(svc.Spec.Ports) != 1 {
				t.Fatalf("Expected 1 port, got %d", len(svc.Spec.Ports))
			}

			port := svc.Spec.Ports[0]

			// Verify all ports are the same dynamic value
			if port.Port != tt.dynamicPort {
				t.Errorf("Service port = %d, want %d", port.Port, tt.dynamicPort)
			}

			if port.TargetPort.IntVal != tt.dynamicPort {
				t.Errorf("Service targetPort = %d, want %d", port.TargetPort.IntVal, tt.dynamicPort)
			}

			if port.NodePort != tt.dynamicPort {
				t.Errorf("Service nodePort = %d, want %d", port.NodePort, tt.dynamicPort)
			}
		})
	}
}

func TestConfigMap_DynamicPort(t *testing.T) {
	tests := []struct {
		name        string
		dynamicPort int32
		wantAddr    string
		wantHost    string
	}{
		{
			name:        "port 30123",
			dynamicPort: 30123,
			wantAddr:    "addr: 0.0.0.0:30123",
			wantHost:    "host: https://testenv-lcr.test-ns.svc.cluster.local:30123",
		},
		{
			name:        "port 31500",
			dynamicPort: 31500,
			wantAddr:    "addr: 0.0.0.0:31500",
			wantHost:    "host: https://testenv-lcr.test-ns.svc.cluster.local:31500",
		},
		{
			name:        "default port when not set",
			dynamicPort: 0, // Will use default 5000
			wantAddr:    "addr: 0.0.0.0:5000",
			wantHost:    "host: https://testenv-lcr.test-ns.svc.cluster.local:5000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ContainerRegistry{
				namespace: "test-ns",
			}
			if tt.dynamicPort != 0 {
				r.SetDynamicPort(tt.dynamicPort)
			}

			// Template the registry config directly to verify port usage
			config := registryConfig{
				FQDN:           r.FQDN(),
				Port:           r.Port(),
				CredentialPath: "/auth/htpasswd",
				CACertPath:     "/certs/ca.crt",
				ServerCertPath: "/certs/tls.crt",
				ServerKeyPath:  "/certs/tls.key",
			}

			buf := bytes.NewBuffer(make([]byte, 0))
			tmpl, err := template.New("").Parse(registryConfigTemplate)
			if err != nil {
				t.Fatalf("Failed to parse template: %v", err)
			}

			if err := tmpl.Execute(buf, config); err != nil {
				t.Fatalf("Failed to execute template: %v", err)
			}

			result := buf.String()

			if !strings.Contains(result, tt.wantAddr) {
				t.Errorf("ConfigMap config missing expected addr:\nwant: %q\ngot:\n%s", tt.wantAddr, result)
			}

			if !strings.Contains(result, tt.wantHost) {
				t.Errorf("ConfigMap config missing expected host:\nwant: %q\ngot:\n%s", tt.wantHost, result)
			}
		})
	}
}

func TestCreateDeployment_ContainerPort(t *testing.T) {
	// This test verifies that the deployment creation uses the dynamic port.
	// Since createDeployment requires eventualconfig, we test the port value
	// that would be used in the deployment.
	tests := []struct {
		name        string
		dynamicPort int32
		wantPort    int32
	}{
		{
			name:        "dynamic port 30123",
			dynamicPort: 30123,
			wantPort:    30123,
		},
		{
			name:        "dynamic port 31500",
			dynamicPort: 31500,
			wantPort:    31500,
		},
		{
			name:        "default port when not set",
			dynamicPort: 0,
			wantPort:    containerRegistryPort, // 5000
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ContainerRegistry{}
			if tt.dynamicPort != 0 {
				r.SetDynamicPort(tt.dynamicPort)
			}

			// Verify that Port() returns the expected value
			// This is the same value used in createDeployment for ContainerPort
			got := r.Port()
			if got != tt.wantPort {
				t.Errorf("Port() for deployment = %d, want %d", got, tt.wantPort)
			}
		})
	}
}

func TestService_PortNameIsHttps(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	cl := fake.NewClientBuilder().WithScheme(scheme).Build()

	r := &ContainerRegistry{
		client:    cl,
		namespace: "test-ns",
	}
	r.SetDynamicPort(30123)

	labels := map[string]string{"app": Name}
	ctx := context.Background()

	err := r.createService(ctx, labels)
	if err != nil {
		t.Fatalf("createService() error = %v", err)
	}

	// Retrieve the created service
	svc := &corev1.Service{}
	err = cl.Get(ctx, client.ObjectKey{Namespace: "test-ns", Name: Name}, svc)
	if err != nil {
		t.Fatalf("Failed to get created service: %v", err)
	}

	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("Expected 1 port, got %d", len(svc.Spec.Ports))
	}

	if svc.Spec.Ports[0].Name != "https" {
		t.Errorf("Service port name = %q, want %q", svc.Spec.Ports[0].Name, "https")
	}
}
