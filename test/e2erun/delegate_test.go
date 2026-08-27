//go:build e2erun

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

package e2erun

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A workspace's provisioned tooling must reach a delegated verb with no
// .envrc sourced and no network: forge itself puts the enclosing root's
// .forge/bin on PATH. This is both the direnv-less proof and the airgap
// proof - the fake binary is the only forge-ci in the world.
func TestDelegationFindsTheWorkspaceBinWithNoEnvrc(t *testing.T) {
	bin, _ := binaries(t)

	root := t.TempDir()
	marker := filepath.Join(root, "ran.txt")

	if err := os.WriteFile(filepath.Join(root, "forge-factory.yaml"),
		[]byte("version: \"1\"\nname: w\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	wsBin := filepath.Join(root, ".forge", "bin")
	if err := os.MkdirAll(wsBin, 0o750); err != nil {
		t.Fatal(err)
	}

	fake := "#!/bin/sh\necho \"$@\" > " + marker + "\n"
	if err := os.WriteFile(filepath.Join(wsBin, "forge-ci"), []byte(fake), 0o750); err != nil { //nolint:gosec // an executable fixture
		t.Fatal(err)
	}

	sub := filepath.Join(root, "member", "deep")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(filepath.Join(bin, "forge"), "ci", "apply", "--config", "x.yaml")
	cmd.Dir = sub
	// A deliberately minimal PATH: no go-installed tooling, nothing but
	// the shell's own directories. The workspace bin is all there is.
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + t.TempDir()}

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("forge ci through the workspace bin: %v: %s", err, out)
	}

	raw, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("the fake forge-ci never ran: %v", err)
	}

	if got := strings.TrimSpace(string(raw)); got != "apply --config x.yaml" {
		t.Fatalf("args must pass through verbatim, got %q", got)
	}
}

// Outside any workspace, with nothing installed, the delegated verb fails
// by naming what provisions it - never a go run at a guessed version.
func TestAnUnprovisionedDelegationFailsActionably(t *testing.T) {
	bin, _ := binaries(t)

	cmd := exec.Command(filepath.Join(bin, "forge"), "register", "status")
	cmd.Dir = t.TempDir()
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + t.TempDir()}

	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("an unprovisioned delegation must fail, got: %s", out)
	}

	if !strings.Contains(string(out), "not provisioned") ||
		!strings.Contains(string(out), "forge factory sync") {
		t.Fatalf("the failure must name what provisions it, got: %s", out)
	}
}
