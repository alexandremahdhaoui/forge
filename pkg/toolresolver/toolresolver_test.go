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

//go:build unit

package toolresolver

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func noPath(string) (string, error) { return "", errors.New("not found") }

func TestOverrideWinsOverEverything(t *testing.T) {
	r := Resolver{
		Override: "/opt/custom/tool",
		LookPath: func(string) (string, error) { return "/usr/bin/tool", nil },
	}

	inv, err := r.Resolve(Ref{Name: "tool", Module: "example.com/x/cmd/tool", Version: "v1.0.0"})
	if err != nil {
		t.Fatal(err)
	}

	if inv.Path != "/opt/custom/tool" || inv.Source != SourceOverride {
		t.Fatalf("override lost: %+v", inv)
	}
}

func TestSourceDirWinsOverPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "cmd", "tool"), 0o755); err != nil {
		t.Fatal(err)
	}

	r := Resolver{
		SourceDir: dir,
		LookPath:  func(string) (string, error) { return "/usr/bin/tool", nil },
	}

	inv, err := r.Resolve(Ref{Name: "tool"})
	if err != nil {
		t.Fatal(err)
	}

	if inv.Source != SourceWorkspace || inv.Path != "go" || inv.Args[1] != "./cmd/tool" {
		t.Fatalf("source dir lost: %+v", inv)
	}
}

func TestStoreWinsOverPath(t *testing.T) {
	r := Resolver{
		StoreLookup: func(name string) (string, bool) { return "/store/bin/" + name, true },
		LookPath:    func(string) (string, error) { return "/usr/bin/tool", nil },
	}

	inv, err := r.Resolve(Ref{Name: "tool"})
	if err != nil {
		t.Fatal(err)
	}

	if inv.Path != "/store/bin/tool" || inv.Source != SourceStore {
		t.Fatalf("store lost: %+v", inv)
	}
}

func TestPathWinsWhenNothingPins(t *testing.T) {
	r := Resolver{LookPath: func(string) (string, error) { return "/usr/bin/tool", nil }}

	inv, err := r.Resolve(Ref{Name: "tool"})
	if err != nil {
		t.Fatal(err)
	}

	if inv.Path != "/usr/bin/tool" || inv.Source != SourcePath {
		t.Fatalf("path lost: %+v", inv)
	}
}

func TestGoRunUsesTheRefVersionAndStripsDirty(t *testing.T) {
	r := Resolver{LookPath: noPath}

	inv, err := r.Resolve(Ref{Name: "tool", Module: "example.com/x/cmd/tool", Version: "v1.2.3-dirty"})
	if err != nil {
		t.Fatal(err)
	}

	if inv.Source != SourceGoRun || inv.Args[1] != "example.com/x/cmd/tool@v1.2.3" {
		t.Fatalf("go run form wrong: %+v", inv)
	}
}

func TestGoRunConsultsThePinWhenTheRefCarriesNone(t *testing.T) {
	r := Resolver{
		LookPath:   noPath,
		PinVersion: func(string) string { return "v2.0.0" },
	}

	inv, err := r.Resolve(Ref{Name: "tool", Module: "example.com/x/cmd/tool"})
	if err != nil {
		t.Fatal(err)
	}

	if inv.Args[1] != "example.com/x/cmd/tool@v2.0.0" {
		t.Fatalf("pin lost: %+v", inv)
	}
}

// Latest is never a fallback: an unpinnable ref fails loud instead.
func TestNothingPinsIsAnErrorNeverLatest(t *testing.T) {
	for _, ref := range []Ref{
		{Name: "tool"}, // no module at all
		{Name: "tool", Module: "example.com/x/cmd/tool"},                 // pin answers nothing
		{Name: "tool", Module: "example.com/x/cmd/tool", Version: "dev"}, // dev is not a pin
	} {
		r := Resolver{LookPath: noPath, PinVersion: func(string) string { return "" }}

		_, err := r.Resolve(ref)
		if err == nil {
			t.Fatalf("ref %+v resolved without a pin", ref)
		}

		if !strings.Contains(err.Error(), "nothing resolves tool") {
			t.Fatalf("error does not name the fix: %v", err)
		}
	}
}

func TestEmptyNameFailsLoud(t *testing.T) {
	if _, err := (Resolver{}).Resolve(Ref{}); err == nil {
		t.Fatal("an empty ref resolved")
	}
}

func TestForgeFactoryDelegationIsPinned(t *testing.T) {
	// Off PATH and outside any workspace, delegation must land on go run
	// at the companion pin - never latest.
	t.Setenv("FORGE_RUN_LOCAL_ENABLED", "")

	r := Resolver{
		LookPath: noPath,
		PinVersion: func(module string) string {
			if v := DepVersion(module); v != "" {
				return v
			}

			return companionForgeFactory
		},
	}

	inv, err := r.Resolve(Ref{Name: "forge-factory", Module: ForgeFactoryModule})
	if err != nil {
		t.Fatal(err)
	}

	if inv.Source != SourceGoRun {
		t.Fatalf("expected go-run, got %+v", inv)
	}

	if strings.Contains(inv.Args[1], "@latest") {
		t.Fatalf("delegation fell back to latest: %+v", inv)
	}

	if !strings.HasPrefix(inv.Args[1], ForgeFactoryModule+"@") {
		t.Fatalf("delegation names the wrong module: %+v", inv)
	}
}

// A released binary stamped with a companion revision finds its toolchain
// in the store; a dev build (no revision) answers nothing and falls
// through.
func TestDefaultStoreLookupConsultsTheCompanionRevision(t *testing.T) {
	store := t.TempDir()
	t.Setenv("FORGE_STORE_DIR", store)

	saved := CompanionRevision
	t.Cleanup(func() { CompanionRevision = saved })

	CompanionRevision = ""
	if _, ok := DefaultStoreLookup("forge-factory"); ok {
		t.Fatal("a dev build must not consult the store")
	}

	CompanionRevision = "abc123def456"
	if _, ok := DefaultStoreLookup("forge-factory"); ok {
		t.Fatal("an empty store must answer nothing")
	}

	viewBin := filepath.Join(store, "rev", "abc123def456",
		runtime.GOOS+"-"+runtime.GOARCH, "bin")
	if err := os.MkdirAll(viewBin, 0o755); err != nil {
		t.Fatal(err)
	}

	binary := filepath.Join(viewBin, "forge-factory")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\n"), 0o555); err != nil {
		t.Fatal(err)
	}

	path, ok := DefaultStoreLookup("forge-factory")
	if !ok || path != binary {
		t.Fatalf("store lookup answered %q %v, want %q", path, ok, binary)
	}

	// The store view outranks PATH in the shared precedence.
	r := Resolver{
		StoreLookup: DefaultStoreLookup,
		LookPath:    func(string) (string, error) { return "/usr/bin/forge-factory", nil },
	}

	inv, err := r.Resolve(Ref{Name: "forge-factory", Module: ForgeFactoryModule})
	if err != nil {
		t.Fatal(err)
	}

	if inv.Path != binary || inv.Source != SourceStore {
		t.Fatalf("the store lost to PATH: %+v", inv)
	}
}

func TestDepVersionAnswersEmptyForUnknownModules(t *testing.T) {
	if v := DepVersion("example.com/never/depended/on"); v != "" {
		t.Fatalf("unexpected version %q", v)
	}
}

// EnsureWorkspaceBin puts the enclosing workspace's pinned tooling on
// PATH for the whole process tree, so nothing depends on .envrc having
// been sourced.
func TestEnsureWorkspaceBinPrependsThePinnedView(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, ".forge", "bin")

	if err := os.MkdirAll(bin, 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, "forge-factory.yaml"), []byte("name: w\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	nested := filepath.Join(root, "member", "cmd", "tool")
	if err := os.MkdirAll(nested, 0o750); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", "/usr/bin")

	if got := EnsureWorkspaceBin(nested); got != bin {
		t.Fatalf("expected %s, got %q", bin, got)
	}

	if path := os.Getenv("PATH"); !strings.HasPrefix(path, bin) {
		t.Fatalf("PATH must start with the workspace bin, got %q", path)
	}

	// Idempotent: a second call changes nothing.
	before := os.Getenv("PATH")

	if got := EnsureWorkspaceBin(nested); got != "" {
		t.Fatalf("a dir already on PATH must answer empty, got %q", got)
	}

	if os.Getenv("PATH") != before {
		t.Fatal("a second call must not change PATH")
	}
}

func TestEnsureWorkspaceBinOutsideAWorkspaceChangesNothing(t *testing.T) {
	t.Setenv("PATH", "/usr/bin")

	if got := EnsureWorkspaceBin(t.TempDir()); got != "" {
		t.Fatalf("no workspace means no change, got %q", got)
	}

	if os.Getenv("PATH") != "/usr/bin" {
		t.Fatal("PATH must be untouched outside a workspace")
	}
}

// A factory file without a provisioned bin dir is a workspace that never
// synced tooling: nothing to expose, nothing to change.
func TestEnsureWorkspaceBinNeedsTheBinDir(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "forge-factory.yaml"), []byte("name: w\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", "/usr/bin")

	if got := EnsureWorkspaceBin(root); got != "" {
		t.Fatalf("no .forge/bin means no change, got %q", got)
	}
}

// A PATH hit served by a workspace's .forge/bin is the pinned view, not a
// user install, and the source says so.
func TestAWorkspaceBinHitIsLabelledHonestly(t *testing.T) {
	r := Resolver{
		LookPath: func(string) (string, error) {
			return "/w/.forge/bin/forge-ci", nil
		},
	}

	inv, err := r.Resolve(Ref{Name: "forge-ci"})
	if err != nil {
		t.Fatal(err)
	}

	if inv.Source != SourceWorkspaceBin {
		t.Fatalf("expected %s, got %s", SourceWorkspaceBin, inv.Source)
	}

	if inv.Path != "/w/.forge/bin/forge-ci" {
		t.Fatalf("unexpected path %s", inv.Path)
	}
}
