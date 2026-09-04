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
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/stretchr/testify/require"
)

func TestTheCommandSeesTheTestenvVariablesTheForgeMetadataAndTheSpecEnv(t *testing.T) {
	input := mcptypes.RunInput{
		Stage: "integration",
		Name:  "integration",
		TestenvEnv: map[string]string{
			"KUBECONFIG":   "/tmp/kubeconfig",
			"SECRET_TOKEN": "secret",
		},
		EnvPropagation:  &forge.EnvPropagation{Blacklist: []string{"SECRET_TOKEN"}},
		TestenvTmpDir:   "/tmp/testenv-123",
		ArtifactFiles:   map[string]string{"kind.kubeconfig": "kubeconfig"},
		TestenvMetadata: map[string]string{"cluster-name": "kind-test"},
		Env:             map[string]string{"RUNNER_VAR": "runner"},
	}
	spec := &Spec{
		Command: "sh",
		Args: []string{"-c", `
test "$KUBECONFIG" = /tmp/kubeconfig &&
test -z "$SECRET_TOKEN" &&
test "$FORGE_TESTENV_TMPDIR" = /tmp/testenv-123 &&
test "$FORGE_ARTIFACT_KIND_KUBECONFIG" = /tmp/testenv-123/kubeconfig &&
test "$FORGE_METADATA_CLUSTER_NAME" = kind-test &&
test "$RUNNER_VAR" = spec
`},
		Env: map[string]string{"RUNNER_VAR": "spec"},
	}

	report, err := Run(context.Background(), input, spec)
	require.NoError(t, err)
	require.Equal(t, "passed", report.Status, report.ErrorMessage)
}

func TestAFailingCommandYieldsAFailedReportAndNoError(t *testing.T) {
	report, err := Run(context.Background(), mcptypes.RunInput{Stage: "unit"}, &Spec{Command: "sh", Args: []string{"-c", "exit 3"}})
	require.NoError(t, err)
	require.Equal(t, "failed", report.Status)
	require.Contains(t, report.ErrorMessage, "3")
}
