package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const (
	// DefaultLeasePath is the default path for the lease file.
	DefaultLeasePath = "/tmp/testenv-lcr.json"
	// DefaultLockTimeout is the default timeout for acquiring the file lock.
	DefaultLockTimeout = 10 * time.Second
	// DefaultMaxRetry is the default maximum number of retry attempts.
	DefaultMaxRetry = 3
	// NodePortMin is the minimum NodePort value.
	NodePortMin = 30000
	// NodePortMax is the maximum NodePort value.
	NodePortMax = 32767
)

// PortLeaseManager manages port leases for testenv-lcr instances.
// It uses a JSON file to track which ports are leased by which clusters,
// with file locking to prevent race conditions.
type PortLeaseManager struct {
	leasePath  string
	timeout    time.Duration
	maxRetry   int
	kindGetCmd func() ([]string, error) // For testing: mock kind get clusters
}

// NewPortLeaseManager creates a new PortLeaseManager with default settings.
func NewPortLeaseManager() *PortLeaseManager {
	return &PortLeaseManager{
		leasePath:  DefaultLeasePath,
		timeout:    DefaultLockTimeout,
		maxRetry:   DefaultMaxRetry,
		kindGetCmd: getKindClusters,
	}
}

// NewPortLeaseManagerWithPath creates a new PortLeaseManager with a custom lease file path.
// This is useful for testing.
func NewPortLeaseManagerWithPath(leasePath string) *PortLeaseManager {
	return &PortLeaseManager{
		leasePath:  leasePath,
		timeout:    DefaultLockTimeout,
		maxRetry:   DefaultMaxRetry,
		kindGetCmd: getKindClusters,
	}
}

// PortLeases represents the JSON structure of the lease file.
// Keys are port numbers (as strings), values are cluster names.
type PortLeases map[string]string

var (
	errAcquiringPort     = errors.New("acquiring port")
	errReleasingPort     = errors.New("releasing port")
	errLockTimeout       = errors.New("lock acquisition timeout")
	errMaxRetriesReached = errors.New("max retries reached: could not find available port")
)

// AcquirePort acquires a port lease for the given cluster name.
// It is idempotent: if the cluster already has a lease, it returns the existing port.
// The port is selected randomly from the NodePort range (30000-32767) and verified
// to be available on the host before being leased.
func (m *PortLeaseManager) AcquirePort(clusterName string) (int32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	bannedPorts := make(map[int32]bool)

	for attempt := 0; attempt < m.maxRetry; attempt++ {
		port, err := m.tryAcquirePort(ctx, clusterName, bannedPorts)
		if err == nil {
			return port, nil
		}

		// If we got an error because the port was already leased, add it to banned list
		if port > 0 {
			bannedPorts[port] = true
		}

		// Check if context is done (timeout or cancellation)
		if ctx.Err() != nil {
			return 0, fmt.Errorf("%w: %v", errLockTimeout, ctx.Err())
		}
	}

	return 0, errMaxRetriesReached
}

// tryAcquirePort attempts to acquire a port once.
// Returns the port number if successful, or the port that was tried (for banning) with an error.
func (m *PortLeaseManager) tryAcquirePort(ctx context.Context, clusterName string, bannedPorts map[int32]bool) (int32, error) {
	// Open or create the lease file with exclusive lock
	file, err := os.OpenFile(m.leasePath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return 0, fmt.Errorf("%w: failed to open lease file: %v", errAcquiringPort, err)
	}
	defer func() { _ = file.Close() }()

	// Acquire file lock with timeout
	if err := m.lockFileWithTimeout(ctx, file); err != nil {
		return 0, err
	}
	defer m.unlockFile(file)

	// Read existing leases
	leases, err := m.readLeases(file)
	if err != nil {
		return 0, fmt.Errorf("%w: failed to read leases: %v", errAcquiringPort, err)
	}

	// Cleanup stale leases (best effort)
	m.cleanupStaleLeases(leases)

	// Check if cluster already has a lease (idempotent behavior)
	for portStr, cluster := range leases {
		if cluster == clusterName {
			var port int32
			if _, err := fmt.Sscanf(portStr, "%d", &port); err == nil {
				return port, nil
			}
		}
	}

	// Find a random available port
	port := m.findRandomPort(bannedPorts, leases)
	if port == 0 {
		return 0, fmt.Errorf("%w: no available port found", errAcquiringPort)
	}

	// Check if port is available on the host
	if !m.isPortAvailable(port) {
		return port, fmt.Errorf("%w: port %d is not available on host", errAcquiringPort, port)
	}

	// Check again if port is already leased (race condition protection)
	portStr := fmt.Sprintf("%d", port)
	if _, exists := leases[portStr]; exists {
		return port, fmt.Errorf("%w: port %d is already leased", errAcquiringPort, port)
	}

	// Add the lease
	leases[portStr] = clusterName

	// Write updated leases
	if err := m.writeLeases(file, leases); err != nil {
		return port, fmt.Errorf("%w: failed to write leases: %v", errAcquiringPort, err)
	}

	return port, nil
}

// ReleasePort releases the port lease for the given cluster name.
// It is a no-op if no lease exists for the cluster.
func (m *PortLeaseManager) ReleasePort(clusterName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	file, err := os.OpenFile(m.leasePath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("%w: failed to open lease file: %v", errReleasingPort, err)
	}
	defer func() { _ = file.Close() }()

	// Acquire file lock with timeout
	if err := m.lockFileWithTimeout(ctx, file); err != nil {
		return err
	}
	defer m.unlockFile(file)

	// Read existing leases
	leases, err := m.readLeases(file)
	if err != nil {
		return fmt.Errorf("%w: failed to read leases: %v", errReleasingPort, err)
	}

	// Find and remove the lease for this cluster
	found := false
	for portStr, cluster := range leases {
		if cluster == clusterName {
			delete(leases, portStr)
			found = true
			break
		}
	}

	// If no lease was found, that's okay (idempotent)
	if !found {
		return nil
	}

	// Write updated leases
	if err := m.writeLeases(file, leases); err != nil {
		return fmt.Errorf("%w: failed to write leases: %v", errReleasingPort, err)
	}

	return nil
}

// lockFileWithTimeout acquires an exclusive lock on the file with timeout.
func (m *PortLeaseManager) lockFileWithTimeout(ctx context.Context, file *os.File) error {
	// Use a ticker to poll for lock with non-blocking flock
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w: %v", errLockTimeout, ctx.Err())
		case <-ticker.C:
			// Try non-blocking lock
			err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
			if err == nil {
				return nil
			}
			if err != syscall.EWOULDBLOCK {
				return fmt.Errorf("%w: flock failed: %v", errAcquiringPort, err)
			}
			// EWOULDBLOCK means someone else has the lock, keep trying
		}
	}
}

// unlockFile releases the file lock.
func (m *PortLeaseManager) unlockFile(file *os.File) {
	_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}

// readLeases reads the lease data from the file.
func (m *PortLeaseManager) readLeases(file *os.File) (PortLeases, error) {
	// Seek to beginning
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// Empty file or new file
	if stat.Size() == 0 {
		return make(PortLeases), nil
	}

	data := make([]byte, stat.Size())
	n, err := file.Read(data)
	if err != nil {
		return nil, err
	}
	data = data[:n]

	// If file is empty or whitespace only, return empty leases
	if len(strings.TrimSpace(string(data))) == 0 {
		return make(PortLeases), nil
	}

	var leases PortLeases
	if err := json.Unmarshal(data, &leases); err != nil {
		// If JSON is invalid, treat as empty (overwrite with new)
		return make(PortLeases), nil
	}

	return leases, nil
}

// writeLeases writes the lease data to the file.
func (m *PortLeaseManager) writeLeases(file *os.File, leases PortLeases) error {
	// Truncate and seek to beginning
	if err := file.Truncate(0); err != nil {
		return err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}

	data, err := json.MarshalIndent(leases, "", "  ")
	if err != nil {
		return err
	}

	_, err = file.Write(data)
	return err
}

// cleanupStaleLeases removes leases for clusters that no longer exist.
// This is a best-effort cleanup - errors are ignored.
func (m *PortLeaseManager) cleanupStaleLeases(leases PortLeases) {
	activeClusters, err := m.kindGetCmd()
	if err != nil {
		// If we can't get clusters, skip cleanup (best effort)
		return
	}

	// Create a set of active clusters
	activeSet := make(map[string]bool)
	for _, cluster := range activeClusters {
		activeSet[cluster] = true
	}

	// Remove leases for non-existent clusters
	for portStr, clusterName := range leases {
		if !activeSet[clusterName] {
			delete(leases, portStr)
		}
	}
}

// findRandomPort finds a random available port in the NodePort range.
// It avoids ports in the banned list and already leased ports.
func (m *PortLeaseManager) findRandomPort(bannedPorts map[int32]bool, leases PortLeases) int32 {
	// Create a set of leased ports
	leasedPorts := make(map[int32]bool)
	for portStr := range leases {
		var port int32
		if _, err := fmt.Sscanf(portStr, "%d", &port); err == nil {
			leasedPorts[port] = true
		}
	}

	// Try random ports
	portRange := NodePortMax - NodePortMin + 1
	for i := 0; i < portRange; i++ {
		//nolint:gosec // Math/rand is fine for port selection, not security-critical
		port := int32(rand.Intn(portRange) + NodePortMin)

		if bannedPorts[port] {
			continue
		}
		if leasedPorts[port] {
			continue
		}
		return port
	}

	return 0
}

// isPortAvailable checks if a port is available on the host by attempting to listen on it.
func (m *PortLeaseManager) isPortAvailable(port int32) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}

// getKindClusters returns the list of active Kind clusters.
func getKindClusters() ([]string, error) {
	cmd := exec.Command("kind", "get", "clusters")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var clusters []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			clusters = append(clusters, line)
		}
	}

	return clusters, nil
}
