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

package runnerexec

import "testing"

func TestRunPropagatesTheExitCode(t *testing.T) {
	t.Parallel()

	code, err := Run(Spec{Artifact: "/bin/sh", Extra: []string{"-c", "exit 3"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 3 {
		t.Fatalf("exit code must pass through verbatim, got %d", code)
	}
}

func TestRunWithNothingToExecuteFails(t *testing.T) {
	t.Parallel()

	if _, err := Run(Spec{}); err == nil {
		t.Fatal("an empty spec must fail")
	}
}

func TestASpecCommandWrapsTheArtifact(t *testing.T) {
	t.Parallel()

	code, err := Run(Spec{
		Artifact: "/tmp/the-artifact",
		Spec: map[string]interface{}{
			"command": "sh",
			"args":    []interface{}{"-c", `test "$FORGE_ARTIFACT" = /tmp/the-artifact`},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Fatal("FORGE_ARTIFACT must reach the wrapped command")
	}
}

func TestSpecEnvReachesTheProcess(t *testing.T) {
	t.Parallel()

	code, err := Run(Spec{
		Artifact: "/bin/sh",
		Extra:    []string{"-c", `test "$GREETING" = hello`},
		Env:      map[string]string{"GREETING": "hello"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Fatal("spec env must reach the process")
	}
}
