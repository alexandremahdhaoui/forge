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

package portalloc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/templateutil"
)

func TestNew(t *testing.T) {
	path := "/tmp/test-portalloc.json"
	a := New(path)
	if a == nil {
		t.Fatal("New returned nil")
	}
	if a.filePath != path {
		t.Errorf("expected filePath=%q, got %q", path, a.filePath)
	}
	if a.file != nil {
		t.Error("expected file to be nil")
	}
	if a.state != nil {
		t.Error("expected state to be nil")
	}
	if a.dirty {
		t.Error("expected dirty to be false")
	}
}

func TestOpenClose_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	a := New(path)
	if err := a.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Verify state is initialized with Version 1 and empty Allocations.
	if a.state == nil {
		t.Fatal("expected state to be non-nil after Open")
	}
	if a.state.Version != 1 {
		t.Errorf("expected Version=1, got %d", a.state.Version)
	}
	if len(a.state.Allocations) != 0 {
		t.Errorf("expected empty Allocations, got %d entries", len(a.state.Allocations))
	}

	if err := a.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Since dirty is false, the file should not have been written.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("expected file to be empty (size=0), got size=%d", info.Size())
	}
}

func TestOpenClose_ExistingState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	// Write valid JSON state to the file.
	existing := PortAllocations{
		Version: 1,
		Allocations: map[string]PortAllocation{
			"127.0.0.1/my-api": {
				Address:     "127.0.0.1",
				Identifier:  "my-api",
				Port:        8080,
				AllocatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent failed: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	a := New(path)
	if err := a.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Verify the state was loaded correctly.
	if a.state.Version != 1 {
		t.Errorf("expected Version=1, got %d", a.state.Version)
	}
	if len(a.state.Allocations) != 1 {
		t.Fatalf("expected 1 allocation, got %d", len(a.state.Allocations))
	}
	alloc, ok := a.state.Allocations["127.0.0.1/my-api"]
	if !ok {
		t.Fatal("expected allocation for key '127.0.0.1/my-api'")
	}
	if alloc.Port != 8080 {
		t.Errorf("expected Port=8080, got %d", alloc.Port)
	}
	if alloc.Address != "127.0.0.1" {
		t.Errorf("expected Address='127.0.0.1', got %q", alloc.Address)
	}
	if alloc.Identifier != "my-api" {
		t.Errorf("expected Identifier='my-api', got %q", alloc.Identifier)
	}

	if err := a.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Since dirty is false, the file should be unchanged.
	afterData, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(afterData) != string(data) {
		t.Errorf("file content changed after Close with dirty=false")
	}
}

func TestOpenClose_DirtyWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	a := New(path)
	if err := a.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Manually set dirty and add an entry.
	a.dirty = true
	a.state.Allocations["127.0.0.1/test-svc"] = PortAllocation{
		Address:     "127.0.0.1",
		Identifier:  "test-svc",
		Port:        9090,
		AllocatedAt: time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC),
	}

	if err := a.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Read the file back and verify the JSON was written.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var written PortAllocations
	if err := json.Unmarshal(data, &written); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if written.Version != 1 {
		t.Errorf("expected Version=1, got %d", written.Version)
	}
	if len(written.Allocations) != 1 {
		t.Fatalf("expected 1 allocation, got %d", len(written.Allocations))
	}
	alloc, ok := written.Allocations["127.0.0.1/test-svc"]
	if !ok {
		t.Fatal("expected allocation for key '127.0.0.1/test-svc'")
	}
	if alloc.Port != 9090 {
		t.Errorf("expected Port=9090, got %d", alloc.Port)
	}
	if alloc.Address != "127.0.0.1" {
		t.Errorf("expected Address='127.0.0.1', got %q", alloc.Address)
	}
	if alloc.Identifier != "test-svc" {
		t.Errorf("expected Identifier='test-svc', got %q", alloc.Identifier)
	}
}

func TestOpenClose_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	// Write garbage to the file.
	if err := os.WriteFile(path, []byte("this is not json{{{"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	a := New(path)
	if err := a.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// State should be re-initialized to empty.
	if a.state.Version != 1 {
		t.Errorf("expected Version=1, got %d", a.state.Version)
	}
	if len(a.state.Allocations) != 0 {
		t.Errorf("expected empty Allocations, got %d entries", len(a.state.Allocations))
	}

	if err := a.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestOpen_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "nested", "state.json")

	a := New(path)
	if err := a.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Verify the directory was created.
	info, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected a directory")
	}

	if err := a.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestClose_Idempotent(t *testing.T) {
	// Close on a freshly constructed (never opened) allocator should not error.
	a := New("/tmp/nonexistent.json")
	if err := a.Close(); err != nil {
		t.Fatalf("Close on never-opened allocator failed: %v", err)
	}

	// Close twice after opening should also not error.
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	a2 := New(path)
	if err := a2.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if err := a2.Close(); err != nil {
		t.Fatalf("first Close failed: %v", err)
	}
	if err := a2.Close(); err != nil {
		t.Fatalf("second Close failed: %v", err)
	}
}

func TestAllocate_NewPort(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	a := New(path)
	if err := a.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer a.Close()

	portStr, err := a.Allocate("127.0.0.1", "test-api")
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}

	if portStr == "" {
		t.Fatal("expected non-empty port string")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("port string %q is not a valid integer: %v", portStr, err)
	}
	if port < 1 || port > 65535 {
		t.Errorf("port %d out of valid range 1-65535", port)
	}

	// Verify the state has the entry.
	alloc, ok := a.state.Allocations["127.0.0.1/test-api"]
	if !ok {
		t.Fatal("expected allocation for key '127.0.0.1/test-api'")
	}
	if alloc.Port != port {
		t.Errorf("expected state port=%d, got %d", port, alloc.Port)
	}
	if alloc.Address != "127.0.0.1" {
		t.Errorf("expected Address='127.0.0.1', got %q", alloc.Address)
	}
	if alloc.Identifier != "test-api" {
		t.Errorf("expected Identifier='test-api', got %q", alloc.Identifier)
	}
}

func TestAllocate_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	a := New(path)
	if err := a.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer a.Close()

	port1, err := a.Allocate("127.0.0.1", "test-api")
	if err != nil {
		t.Fatalf("first Allocate failed: %v", err)
	}

	port2, err := a.Allocate("127.0.0.1", "test-api")
	if err != nil {
		t.Fatalf("second Allocate failed: %v", err)
	}

	if port1 != port2 {
		t.Errorf("expected same port on idempotent call, got %q and %q", port1, port2)
	}
}

func TestAllocate_DifferentIDs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	a := New(path)
	if err := a.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer a.Close()

	port1, err := a.Allocate("127.0.0.1", "api")
	if err != nil {
		t.Fatalf("Allocate api failed: %v", err)
	}

	port2, err := a.Allocate("127.0.0.1", "metrics")
	if err != nil {
		t.Fatalf("Allocate metrics failed: %v", err)
	}

	if port1 == port2 {
		t.Errorf("expected different ports for different IDs, both got %q", port1)
	}

	if len(a.state.Allocations) != 2 {
		t.Errorf("expected 2 allocations, got %d", len(a.state.Allocations))
	}
}

func TestAllocate_InvalidAddr(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	a := New(path)
	if err := a.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer a.Close()

	_, err := a.Allocate("not-an-ip", "test")
	if err == nil {
		t.Fatal("expected error for invalid IP address")
	}
	if !strings.Contains(err.Error(), "invalid IP address") {
		t.Errorf("expected error to contain 'invalid IP address', got %q", err.Error())
	}
}

func TestAllocate_InvalidID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	a := New(path)
	if err := a.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer a.Close()

	// Test with slash in identifier.
	_, err := a.Allocate("127.0.0.1", "bad/id")
	if err == nil {
		t.Fatal("expected error for invalid identifier with slash")
	}
	if !strings.Contains(err.Error(), "invalid identifier") {
		t.Errorf("expected error to contain 'invalid identifier', got %q", err.Error())
	}

	// Test with empty string.
	_, err = a.Allocate("127.0.0.1", "")
	if err == nil {
		t.Fatal("expected error for empty identifier")
	}
	if !strings.Contains(err.Error(), "invalid identifier") {
		t.Errorf("expected error to contain 'invalid identifier', got %q", err.Error())
	}
}

func TestAllocate_IPv6(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	a := New(path)
	if err := a.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer a.Close()

	portStr, err := a.Allocate("::1", "test")
	if err != nil {
		t.Fatalf("Allocate with IPv6 failed: %v", err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("port string %q is not a valid integer: %v", portStr, err)
	}
	if port < 1 || port > 65535 {
		t.Errorf("port %d out of valid range 1-65535", port)
	}

	// Verify key format uses the raw IPv6 address.
	if _, ok := a.state.Allocations["::1/test"]; !ok {
		t.Error("expected allocation with key '::1/test'")
	}
}

func TestRelease_Existing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	a := New(path)
	if err := a.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer a.Close()

	_, err := a.Allocate("127.0.0.1", "test-api")
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}

	if len(a.state.Allocations) != 1 {
		t.Fatalf("expected 1 allocation, got %d", len(a.state.Allocations))
	}

	if err := a.Release("127.0.0.1", "test-api"); err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	if len(a.state.Allocations) != 0 {
		t.Errorf("expected 0 allocations after release, got %d", len(a.state.Allocations))
	}
}

func TestRelease_NonExistent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	a := New(path)
	if err := a.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer a.Close()

	// Releasing a key that doesn't exist should not error (idempotent).
	if err := a.Release("127.0.0.1", "nonexistent"); err != nil {
		t.Fatalf("Release on nonexistent key failed: %v", err)
	}
}

func TestAllocate_NotOpened(t *testing.T) {
	a := New("/tmp/not-opened.json")

	_, err := a.Allocate("127.0.0.1", "test")
	if err == nil {
		t.Fatal("expected error when allocator is not opened")
	}
	if !strings.Contains(err.Error(), "allocator not opened") {
		t.Errorf("expected error to contain 'allocator not opened', got %q", err.Error())
	}
}

func TestAllocate_PersistsAcrossOpenClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	// First session: allocate a port.
	a := New(path)
	if err := a.Open(); err != nil {
		t.Fatalf("first Open failed: %v", err)
	}

	port1, err := a.Allocate("127.0.0.1", "persist-test")
	if err != nil {
		t.Fatalf("first Allocate failed: %v", err)
	}

	if err := a.Close(); err != nil {
		t.Fatalf("first Close failed: %v", err)
	}

	// Second session: reopen and allocate with same addr/id.
	a2 := New(path)
	if err := a2.Open(); err != nil {
		t.Fatalf("second Open failed: %v", err)
	}
	defer a2.Close()

	port2, err := a2.Allocate("127.0.0.1", "persist-test")
	if err != nil {
		t.Fatalf("second Allocate failed: %v", err)
	}

	if port1 != port2 {
		t.Errorf("expected same port across open/close cycles, got %q and %q", port1, port2)
	}
}

// TestAllocateOpenPort_EndToEnd exercises the full pipeline as used in orchestrateCreate:
// PortAllocator.Open -> build FuncMap -> ExpandTemplates with allocateOpenPort -> Close.
// This proves the feature works end-to-end, not just in unit isolation.
func TestAllocateOpenPort_EndToEnd(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "port-allocations.json")

	// Simulate what orchestrateCreate does: create allocator, open, build FuncMap, expand, close.
	allocator := New(stateFile)
	if err := allocator.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	funcMap := template.FuncMap{
		"allocateOpenPort": allocator.Allocate,
	}

	// Simulate a forge.yaml spec with allocateOpenPort templates
	spec := map[string]interface{}{
		"apiPort":     `{{ allocateOpenPort "127.0.0.1" "my-api" }}`,
		"metricsPort": `{{ allocateOpenPort "127.0.0.1" "my-metrics" }}`,
		"name":        "{{ .Env.APP_NAME }}",
		"staticValue": "no-template-here",
	}
	env := map[string]string{
		"APP_NAME": "test-app",
	}

	result, err := templateutil.ExpandTemplates(spec, env, templateutil.WithFuncMap(funcMap))
	if err != nil {
		t.Fatalf("ExpandTemplates failed: %v", err)
	}

	if err := allocator.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify apiPort is a valid port number
	apiPortStr, ok := result["apiPort"].(string)
	if !ok {
		t.Fatalf("apiPort is not a string: %v", result["apiPort"])
	}
	apiPort, err := strconv.Atoi(apiPortStr)
	if err != nil {
		t.Fatalf("apiPort is not a number: %q", apiPortStr)
	}
	if apiPort < 1 || apiPort > 65535 {
		t.Errorf("apiPort out of range: %d", apiPort)
	}

	// Verify metricsPort is a valid and different port
	metricsPortStr, ok := result["metricsPort"].(string)
	if !ok {
		t.Fatalf("metricsPort is not a string: %v", result["metricsPort"])
	}
	metricsPort, err := strconv.Atoi(metricsPortStr)
	if err != nil {
		t.Fatalf("metricsPort is not a number: %q", metricsPortStr)
	}
	if metricsPort < 1 || metricsPort > 65535 {
		t.Errorf("metricsPort out of range: %d", metricsPort)
	}
	if apiPort == metricsPort {
		t.Errorf("apiPort and metricsPort should be different, both are %d", apiPort)
	}

	// Verify env template still works alongside allocateOpenPort
	if result["name"] != "test-app" {
		t.Errorf("expected name='test-app', got %q", result["name"])
	}

	// Verify static values pass through unchanged
	if result["staticValue"] != "no-template-here" {
		t.Errorf("expected staticValue='no-template-here', got %q", result["staticValue"])
	}

	// Verify state was persisted to disk
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("failed to read state file: %v", err)
	}
	var state PortAllocations
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("failed to unmarshal state: %v", err)
	}
	if len(state.Allocations) != 2 {
		t.Errorf("expected 2 allocations in state, got %d", len(state.Allocations))
	}
	if _, ok := state.Allocations["127.0.0.1/my-api"]; !ok {
		t.Error("missing allocation for 127.0.0.1/my-api")
	}
	if _, ok := state.Allocations["127.0.0.1/my-metrics"]; !ok {
		t.Error("missing allocation for 127.0.0.1/my-metrics")
	}

	// Verify idempotency: re-open, re-expand, same ports returned
	allocator2 := New(stateFile)
	if err := allocator2.Open(); err != nil {
		t.Fatalf("second Open failed: %v", err)
	}
	funcMap2 := template.FuncMap{
		"allocateOpenPort": allocator2.Allocate,
	}
	result2, err := templateutil.ExpandTemplates(spec, env, templateutil.WithFuncMap(funcMap2))
	if err != nil {
		t.Fatalf("second ExpandTemplates failed: %v", err)
	}
	if err := allocator2.Close(); err != nil {
		t.Fatalf("second Close failed: %v", err)
	}

	if result2["apiPort"] != apiPortStr {
		t.Errorf("idempotency failed for apiPort: first=%q, second=%q", apiPortStr, result2["apiPort"])
	}
	if result2["metricsPort"] != metricsPortStr {
		t.Errorf("idempotency failed for metricsPort: first=%q, second=%q", metricsPortStr, result2["metricsPort"])
	}
}
