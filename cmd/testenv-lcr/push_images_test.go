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
	"context"
	"crypto/x509"
	"encoding/pem"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeTLSServerCertToFile extracts the TLS certificate from an httptest.TLSServer
// and writes it as a PEM-encoded file to the given directory. Returns the file path.
func writeTLSServerCertToFile(t *testing.T, server *httptest.Server, dir string) string {
	t.Helper()

	derBytes := server.TLS.Certificates[0].Certificate[0]

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		t.Fatalf("Failed to parse server certificate: %v", err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{ //nolint:exhaustruct
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})

	certPath := filepath.Join(dir, "ca.crt")
	if err := os.WriteFile(certPath, pemBytes, 0o600); err != nil {
		t.Fatalf("Failed to write CA cert file: %v", err)
	}

	return certPath
}

// serverPort extracts the TCP port from an httptest.Server's listener address.
func serverPort(t *testing.T, server *httptest.Server) int32 {
	t.Helper()

	addr, ok := server.Listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatal("Failed to cast server listener address to *net.TCPAddr")
	}

	return int32(addr.Port)
}

func TestHttpsHealthCheck_SucceedsOn401(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	certPath := writeTLSServerCertToFile(t, server, t.TempDir())
	port := serverPort(t, server)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpsHealthCheck(ctx, port, certPath, "127.0.0.1"); err != nil {
		t.Errorf("httpsHealthCheck() returned error for 401 response: %v", err)
	}
}

func TestHttpsHealthCheck_SucceedsOn200(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	certPath := writeTLSServerCertToFile(t, server, t.TempDir())
	port := serverPort(t, server)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpsHealthCheck(ctx, port, certPath, "127.0.0.1"); err != nil {
		t.Errorf("httpsHealthCheck() returned error for 200 response: %v", err)
	}
}

func TestHttpsHealthCheck_TimeoutWhenNoServer(t *testing.T) {
	// Start a TLS server to get a valid cert, then close it immediately.
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	certPath := writeTLSServerCertToFile(t, server, t.TempDir())
	port := serverPort(t, server)

	// Close the server so nothing is listening on the port.
	server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := httpsHealthCheck(ctx, port, certPath, "127.0.0.1")
	if err == nil {
		t.Error("httpsHealthCheck() returned nil error when no server is listening")
	}
}

func TestHttpsHealthCheck_FailsWithBadCACert(t *testing.T) {
	certPath := filepath.Join(t.TempDir(), "bad-ca.crt")
	if err := os.WriteFile(certPath, []byte("not a cert"), 0o600); err != nil {
		t.Fatalf("Failed to write bad CA cert file: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := httpsHealthCheck(ctx, 19999, certPath, "localhost")
	if err == nil {
		t.Error("httpsHealthCheck() returned nil error with bad CA certificate")
	}
}
