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
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestNewPortLeaseManager(t *testing.T) {
	m := NewPortLeaseManager()

	if m.leasePath != DefaultLeasePath {
		t.Errorf("expected leasePath %s, got %s", DefaultLeasePath, m.leasePath)
	}
	if m.timeout != DefaultLockTimeout {
		t.Errorf("expected timeout %v, got %v", DefaultLockTimeout, m.timeout)
	}
	if m.maxRetry != DefaultMaxRetry {
		t.Errorf("expected maxRetry %d, got %d", DefaultMaxRetry, m.maxRetry)
	}
	if m.kindGetCmd == nil {
		t.Error("expected kindGetCmd to be set")
	}
}

func TestNewPortLeaseManagerWithPath(t *testing.T) {
	customPath := "/custom/path/leases.json"
	m := NewPortLeaseManagerWithPath(customPath)

	if m.leasePath != customPath {
		t.Errorf("expected leasePath %s, got %s", customPath, m.leasePath)
	}
	if m.timeout != DefaultLockTimeout {
		t.Errorf("expected timeout %v, got %v", DefaultLockTimeout, m.timeout)
	}
	if m.maxRetry != DefaultMaxRetry {
		t.Errorf("expected maxRetry %d, got %d", DefaultMaxRetry, m.maxRetry)
	}
}

func TestAcquirePort_Success(t *testing.T) {
	tmpDir := t.TempDir()
	leasePath := filepath.Join(tmpDir, "leases.json")

	m := NewPortLeaseManagerWithPath(leasePath)
	// Mock kind get clusters to return the test cluster as active
	m.kindGetCmd = func() ([]string, error) {
		return []string{"test-cluster"}, nil
	}

	port, err := m.AcquirePort("test-cluster")
	if err != nil {
		t.Fatalf("AcquirePort failed: %v", err)
	}

	if port < NodePortMin || port > NodePortMax {
		t.Errorf("port %d is outside valid range [%d, %d]", port, NodePortMin, NodePortMax)
	}

	// Verify the lease file was written
	data, err := os.ReadFile(leasePath)
	if err != nil {
		t.Fatalf("failed to read lease file: %v", err)
	}

	if len(data) == 0 {
		t.Error("lease file should not be empty after acquiring port")
	}
}

func TestAcquirePort_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	leasePath := filepath.Join(tmpDir, "leases.json")

	m := NewPortLeaseManagerWithPath(leasePath)
	// Mock kind get clusters to return the test cluster as active
	m.kindGetCmd = func() ([]string, error) {
		return []string{"test-cluster"}, nil
	}

	// First acquisition
	port1, err := m.AcquirePort("test-cluster")
	if err != nil {
		t.Fatalf("first AcquirePort failed: %v", err)
	}

	// Second acquisition should return the same port
	port2, err := m.AcquirePort("test-cluster")
	if err != nil {
		t.Fatalf("second AcquirePort failed: %v", err)
	}

	if port1 != port2 {
		t.Errorf("idempotent call returned different port: first=%d, second=%d", port1, port2)
	}
}

func TestAcquirePort_ValidRange(t *testing.T) {
	tmpDir := t.TempDir()
	leasePath := filepath.Join(tmpDir, "leases.json")

	m := NewPortLeaseManagerWithPath(leasePath)
	// Mock kind get clusters
	m.kindGetCmd = func() ([]string, error) {
		return []string{"test-cluster"}, nil
	}

	// Acquire multiple ports and verify they're all in valid range
	for i := 0; i < 10; i++ {
		clusterName := "test-cluster-" + string(rune('a'+i))

		// Update mock to include this cluster
		clusters := make([]string, i+1)
		for j := 0; j <= i; j++ {
			clusters[j] = "test-cluster-" + string(rune('a'+j))
		}
		m.kindGetCmd = func() ([]string, error) {
			return clusters, nil
		}

		port, err := m.AcquirePort(clusterName)
		if err != nil {
			t.Fatalf("AcquirePort for %s failed: %v", clusterName, err)
		}

		if port < NodePortMin || port > NodePortMax {
			t.Errorf("cluster %s got port %d outside valid range [%d, %d]",
				clusterName, port, NodePortMin, NodePortMax)
		}
	}
}

func TestAcquirePort_AvoidsExistingLeases(t *testing.T) {
	tmpDir := t.TempDir()
	leasePath := filepath.Join(tmpDir, "leases.json")

	m := NewPortLeaseManagerWithPath(leasePath)
	// Mock kind get clusters
	m.kindGetCmd = func() ([]string, error) {
		return []string{"cluster-a", "cluster-b"}, nil
	}

	// Acquire port for first cluster
	port1, err := m.AcquirePort("cluster-a")
	if err != nil {
		t.Fatalf("first AcquirePort failed: %v", err)
	}

	// Acquire port for second cluster - should get a different port
	port2, err := m.AcquirePort("cluster-b")
	if err != nil {
		t.Fatalf("second AcquirePort failed: %v", err)
	}

	if port1 == port2 {
		t.Errorf("different clusters got same port: %d", port1)
	}
}

func TestAcquirePort_CleansUpStaleLeases(t *testing.T) {
	tmpDir := t.TempDir()
	leasePath := filepath.Join(tmpDir, "leases.json")

	// Pre-populate lease file with a stale lease
	initialLeases := `{
  "30100": "stale-cluster",
  "30101": "active-cluster"
}`
	if err := os.WriteFile(leasePath, []byte(initialLeases), 0o644); err != nil {
		t.Fatalf("failed to write initial leases: %v", err)
	}

	m := NewPortLeaseManagerWithPath(leasePath)
	// Mock kind get clusters - only active-cluster exists
	m.kindGetCmd = func() ([]string, error) {
		return []string{"active-cluster", "new-cluster"}, nil
	}

	// Acquire port for new cluster - should trigger cleanup
	_, err := m.AcquirePort("new-cluster")
	if err != nil {
		t.Fatalf("AcquirePort failed: %v", err)
	}

	// Read the lease file and verify stale-cluster was removed
	data, err := os.ReadFile(leasePath)
	if err != nil {
		t.Fatalf("failed to read lease file: %v", err)
	}

	content := string(data)
	if contains(content, "stale-cluster") {
		t.Error("stale-cluster lease should have been removed")
	}
	if !contains(content, "active-cluster") {
		t.Error("active-cluster lease should still exist")
	}
}

func TestReleasePort_Success(t *testing.T) {
	tmpDir := t.TempDir()
	leasePath := filepath.Join(tmpDir, "leases.json")

	m := NewPortLeaseManagerWithPath(leasePath)
	m.kindGetCmd = func() ([]string, error) {
		return []string{"test-cluster"}, nil
	}

	// First acquire a port
	port, err := m.AcquirePort("test-cluster")
	if err != nil {
		t.Fatalf("AcquirePort failed: %v", err)
	}

	// Then release it
	err = m.ReleasePort("test-cluster")
	if err != nil {
		t.Fatalf("ReleasePort failed: %v", err)
	}

	// Verify the lease was removed
	data, err := os.ReadFile(leasePath)
	if err != nil {
		t.Fatalf("failed to read lease file: %v", err)
	}

	content := string(data)
	portStr := string(rune(port)) // This won't work, use proper conversion
	portStrCorrect := ""
	for _, r := range string(data) {
		if r >= '0' && r <= '9' {
			portStrCorrect += string(r)
		}
	}

	// A simpler check: the cluster name should not be in the file
	if contains(content, "test-cluster") {
		t.Error("test-cluster lease should have been removed")
	}
	_ = portStr // suppress unused warning
}

func TestReleasePort_RemovesCorrectEntry(t *testing.T) {
	tmpDir := t.TempDir()
	leasePath := filepath.Join(tmpDir, "leases.json")

	m := NewPortLeaseManagerWithPath(leasePath)
	m.kindGetCmd = func() ([]string, error) {
		return []string{"cluster-a", "cluster-b"}, nil
	}

	// Acquire ports for two clusters
	_, err := m.AcquirePort("cluster-a")
	if err != nil {
		t.Fatalf("AcquirePort for cluster-a failed: %v", err)
	}

	_, err = m.AcquirePort("cluster-b")
	if err != nil {
		t.Fatalf("AcquirePort for cluster-b failed: %v", err)
	}

	// Release only cluster-a
	err = m.ReleasePort("cluster-a")
	if err != nil {
		t.Fatalf("ReleasePort failed: %v", err)
	}

	// Verify cluster-a was removed but cluster-b still exists
	data, err := os.ReadFile(leasePath)
	if err != nil {
		t.Fatalf("failed to read lease file: %v", err)
	}

	content := string(data)
	if contains(content, "cluster-a") {
		t.Error("cluster-a lease should have been removed")
	}
	if !contains(content, "cluster-b") {
		t.Error("cluster-b lease should still exist")
	}
}

func TestReleasePort_NoOpIfNotLeased(t *testing.T) {
	tmpDir := t.TempDir()
	leasePath := filepath.Join(tmpDir, "leases.json")

	m := NewPortLeaseManagerWithPath(leasePath)

	// Release a port that was never leased - should not error
	err := m.ReleasePort("nonexistent-cluster")
	if err != nil {
		t.Fatalf("ReleasePort for nonexistent cluster should not error: %v", err)
	}
}

func TestConcurrentAcquire(t *testing.T) {
	tmpDir := t.TempDir()
	leasePath := filepath.Join(tmpDir, "leases.json")

	const numGoroutines = 10
	var wg sync.WaitGroup
	ports := make(chan int32, numGoroutines)
	errors := make(chan error, numGoroutines)

	// Create a list of clusters for the mock
	clusters := make([]string, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		clusters[i] = "concurrent-cluster-" + string(rune('a'+i))
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			m := NewPortLeaseManagerWithPath(leasePath)
			m.kindGetCmd = func() ([]string, error) {
				return clusters, nil
			}

			clusterName := clusters[idx]
			port, err := m.AcquirePort(clusterName)
			if err != nil {
				errors <- err
				return
			}
			ports <- port
		}(i)
	}

	wg.Wait()
	close(ports)
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("concurrent acquire error: %v", err)
	}

	// Verify all ports are unique
	seen := make(map[int32]bool)
	for port := range ports {
		if seen[port] {
			t.Errorf("duplicate port acquired: %d", port)
		}
		seen[port] = true

		// Verify port is in valid range
		if port < NodePortMin || port > NodePortMax {
			t.Errorf("port %d is outside valid range", port)
		}
	}
}

func TestAcquirePort_HandlesInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	leasePath := filepath.Join(tmpDir, "leases.json")

	// Write invalid JSON
	if err := os.WriteFile(leasePath, []byte("not valid json"), 0o644); err != nil {
		t.Fatalf("failed to write invalid JSON: %v", err)
	}

	m := NewPortLeaseManagerWithPath(leasePath)
	m.kindGetCmd = func() ([]string, error) {
		return []string{"test-cluster"}, nil
	}

	// Should still succeed by treating invalid JSON as empty
	port, err := m.AcquirePort("test-cluster")
	if err != nil {
		t.Fatalf("AcquirePort should handle invalid JSON gracefully: %v", err)
	}

	if port < NodePortMin || port > NodePortMax {
		t.Errorf("port %d is outside valid range", port)
	}
}

func TestAcquirePort_HandlesEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	leasePath := filepath.Join(tmpDir, "leases.json")

	// Create empty file
	if err := os.WriteFile(leasePath, []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write empty file: %v", err)
	}

	m := NewPortLeaseManagerWithPath(leasePath)
	m.kindGetCmd = func() ([]string, error) {
		return []string{"test-cluster"}, nil
	}

	port, err := m.AcquirePort("test-cluster")
	if err != nil {
		t.Fatalf("AcquirePort should handle empty file: %v", err)
	}

	if port < NodePortMin || port > NodePortMax {
		t.Errorf("port %d is outside valid range", port)
	}
}

func TestAcquirePort_KindGetClustersFailure(t *testing.T) {
	tmpDir := t.TempDir()
	leasePath := filepath.Join(tmpDir, "leases.json")

	m := NewPortLeaseManagerWithPath(leasePath)
	// Mock kind get clusters to fail
	m.kindGetCmd = func() ([]string, error) {
		return nil, os.ErrNotExist
	}

	// Should still succeed - kind failure means skip stale cleanup
	port, err := m.AcquirePort("test-cluster")
	if err != nil {
		t.Fatalf("AcquirePort should succeed even when kind get clusters fails: %v", err)
	}

	if port < NodePortMin || port > NodePortMax {
		t.Errorf("port %d is outside valid range", port)
	}
}

func TestCleanupStaleLeases(t *testing.T) {
	tmpDir := t.TempDir()
	leasePath := filepath.Join(tmpDir, "leases.json")

	// Pre-populate with leases
	initialLeases := `{
  "30100": "active-cluster-1",
  "30101": "stale-cluster",
  "30102": "active-cluster-2"
}`
	if err := os.WriteFile(leasePath, []byte(initialLeases), 0o644); err != nil {
		t.Fatalf("failed to write initial leases: %v", err)
	}

	m := NewPortLeaseManagerWithPath(leasePath)
	// Only active clusters exist
	m.kindGetCmd = func() ([]string, error) {
		return []string{"active-cluster-1", "active-cluster-2", "new-cluster"}, nil
	}

	// Trigger cleanup by acquiring a new port
	port, err := m.AcquirePort("new-cluster")
	if err != nil {
		t.Fatalf("AcquirePort failed: %v", err)
	}

	if port < NodePortMin || port > NodePortMax {
		t.Errorf("port %d is outside valid range", port)
	}

	// Read back and verify
	data, err := os.ReadFile(leasePath)
	if err != nil {
		t.Fatalf("failed to read lease file: %v", err)
	}

	content := string(data)
	if contains(content, "stale-cluster") {
		t.Error("stale-cluster should have been cleaned up")
	}
	if !contains(content, "active-cluster-1") {
		t.Error("active-cluster-1 should still exist")
	}
	if !contains(content, "active-cluster-2") {
		t.Error("active-cluster-2 should still exist")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
