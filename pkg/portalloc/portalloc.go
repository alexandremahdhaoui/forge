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

package portalloc

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"syscall"
	"time"
)

// identifierRe validates that an identifier contains only alphanumeric characters, hyphens, and underscores.
var identifierRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// PortAllocations holds the persisted state of all port allocations.
type PortAllocations struct {
	Version     int                       `json:"version"`
	Allocations map[string]PortAllocation `json:"allocations"`
}

// PortAllocation represents a single allocated port.
type PortAllocation struct {
	Address     string    `json:"address"`
	Identifier  string    `json:"identifier"`
	Port        int       `json:"port"`
	AllocatedAt time.Time `json:"allocatedAt"`
}

// PortAllocator manages port allocations with file-based persistence and flock-based concurrency.
type PortAllocator struct {
	filePath string
	file     *os.File
	state    *PortAllocations
	dirty    bool
}

// New creates a new PortAllocator that will persist state to the given file path.
func New(filePath string) *PortAllocator {
	return &PortAllocator{
		filePath: filePath,
	}
}

// Open opens the state file, acquires an exclusive flock, and loads the state.
// If the file does not exist it is created. If the file contains invalid JSON,
// the state is re-initialized to an empty allocation map.
func (a *PortAllocator) Open() error {
	if err := os.MkdirAll(filepath.Dir(a.filePath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", a.filePath, err)
	}

	file, err := os.OpenFile(a.filePath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", a.filePath, err)
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		_ = file.Close()
		return fmt.Errorf("failed to acquire lock on %s: %w", a.filePath, err)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
		return fmt.Errorf("failed to read file %s: %w", a.filePath, err)
	}

	state := &PortAllocations{
		Version:     1,
		Allocations: make(map[string]PortAllocation),
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, state); err != nil {
			// Invalid JSON: re-initialize to empty state.
			state = &PortAllocations{
				Version:     1,
				Allocations: make(map[string]PortAllocation),
			}
		}
	}

	a.file = file
	a.state = state
	a.dirty = false

	return nil
}

// Close persists state to disk if dirty, releases the flock, and closes the file.
// Close is idempotent: calling it on an allocator that was never opened or already
// closed returns nil.
func (a *PortAllocator) Close() error {
	if a.file == nil {
		return nil
	}

	if a.dirty {
		data, err := json.MarshalIndent(a.state, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal state: %w", err)
		}

		if err := a.file.Truncate(0); err != nil {
			return fmt.Errorf("failed to truncate file: %w", err)
		}

		if _, err := a.file.Seek(0, 0); err != nil {
			return fmt.Errorf("failed to seek file: %w", err)
		}

		if _, err := a.file.Write(data); err != nil {
			return fmt.Errorf("failed to write state: %w", err)
		}
	}

	_ = syscall.Flock(int(a.file.Fd()), syscall.LOCK_UN)
	_ = a.file.Close()
	a.file = nil

	return nil
}

// Allocate returns a port string for the given address and identifier.
// If the same addr/id pair was previously allocated and the port is still available,
// the same port is returned (idempotent). If the previously allocated port is no
// longer available, a new port is allocated. The allocation is persisted on Close.
func (a *PortAllocator) Allocate(addr, id string) (string, error) {
	if a.state == nil {
		return "", fmt.Errorf("allocator not opened")
	}

	if net.ParseIP(addr) == nil {
		return "", fmt.Errorf("invalid IP address: %q", addr)
	}

	if !identifierRe.MatchString(id) {
		return "", fmt.Errorf("invalid identifier %q: must match ^[a-zA-Z0-9_-]+$", id)
	}

	key := addr + "/" + id

	// If we already have an allocation for this key, check if the port is still available.
	if existing, ok := a.state.Allocations[key]; ok {
		listener, err := net.Listen("tcp", net.JoinHostPort(existing.Address, fmt.Sprintf("%d", existing.Port)))
		if err == nil {
			// Port is still available; close the probe listener and return the existing port.
			_ = listener.Close()
			return fmt.Sprintf("%d", existing.Port), nil
		}
		// Port is taken by another process; delete stale entry and fall through.
		delete(a.state.Allocations, key)
	}

	// Allocate a new port by binding to port 0.
	listener, err := net.Listen("tcp", net.JoinHostPort(addr, "0"))
	if err != nil {
		return "", fmt.Errorf("failed to listen on %s:0: %w", addr, err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	a.state.Allocations[key] = PortAllocation{
		Address:     addr,
		Identifier:  id,
		Port:        port,
		AllocatedAt: time.Now().UTC(),
	}
	a.dirty = true

	return fmt.Sprintf("%d", port), nil
}

// AllocateInRange returns a port string for the given address and identifier,
// constrained to [minPort, maxPort]. This is useful for Kubernetes NodePort
// services that require ports in 30000-32767.
func (a *PortAllocator) AllocateInRange(addr, id string, minPort, maxPort int) (string, error) {
	if a.state == nil {
		return "", fmt.Errorf("allocator not opened")
	}

	if net.ParseIP(addr) == nil {
		return "", fmt.Errorf("invalid IP address: %q", addr)
	}

	if !identifierRe.MatchString(id) {
		return "", fmt.Errorf("invalid identifier %q: must match ^[a-zA-Z0-9_-]+$", id)
	}

	if minPort < 1 || maxPort > 65535 || minPort > maxPort {
		return "", fmt.Errorf("invalid port range [%d, %d]", minPort, maxPort)
	}

	key := addr + "/" + id

	// If we already have an allocation for this key, check if the port is still
	// available and within the requested range.
	if existing, ok := a.state.Allocations[key]; ok {
		if existing.Port >= minPort && existing.Port <= maxPort {
			listener, err := net.Listen("tcp", net.JoinHostPort(existing.Address, fmt.Sprintf("%d", existing.Port)))
			if err == nil {
				_ = listener.Close()
				return fmt.Sprintf("%d", existing.Port), nil
			}
		}
		delete(a.state.Allocations, key)
	}

	// Collect all ports currently allocated by other identifiers to avoid conflicts.
	usedPorts := make(map[int]bool)
	for _, alloc := range a.state.Allocations {
		usedPorts[alloc.Port] = true
	}

	// Scan the range for an available port. Start at a pseudo-random offset
	// derived from the identifier to spread allocations across the range.
	rangeSize := maxPort - minPort + 1
	offset := 0
	for _, b := range []byte(id) {
		offset = (offset*31 + int(b)) % rangeSize
	}

	for i := range rangeSize {
		candidate := minPort + (offset+i)%rangeSize
		if usedPorts[candidate] {
			continue
		}
		listener, err := net.Listen("tcp", net.JoinHostPort(addr, fmt.Sprintf("%d", candidate)))
		if err != nil {
			continue
		}
		_ = listener.Close()

		a.state.Allocations[key] = PortAllocation{
			Address:     addr,
			Identifier:  id,
			Port:        candidate,
			AllocatedAt: time.Now().UTC(),
		}
		a.dirty = true
		return fmt.Sprintf("%d", candidate), nil
	}

	return "", fmt.Errorf("no available port in range [%d, %d] on %s", minPort, maxPort, addr)
}

// Release removes the allocation for the given address and identifier.
// Release is idempotent: no error is returned if the key does not exist.
func (a *PortAllocator) Release(addr, id string) error {
	if a.state == nil {
		return fmt.Errorf("allocator not opened")
	}

	key := addr + "/" + id
	if _, ok := a.state.Allocations[key]; ok {
		delete(a.state.Allocations, key)
		a.dirty = true
	}

	return nil
}
