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
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/stretchr/testify/require"
)

func TestGoTestHandsTheTestsTheFilteredTestenvPlusTheForgeMetadataPlusTheSpecEnv(t *testing.T) {
	input := mcptypes.RunInput{
		Stage: "integration",
		TestenvEnv: map[string]string{
			"KUBECONFIG":   "/tmp/kubeconfig",
			"SECRET_TOKEN": "secret",
		},
		EnvPropagation: &forge.EnvPropagation{Blacklist: []string{"SECRET_TOKEN"}},
		TestenvTmpDir:  "/tmp/test-123",
		ArtifactFiles:  map[string]string{"testenv-kind.kubeconfig": "kubeconfig"},
		TestenvMetadata: map[string]string{
			"cluster.name": "kind-test",
		},
		Env: map[string]string{"RUNNER_VAR": "runner"},
	}
	spec := &Spec{Env: map[string]string{"RUNNER_VAR": "spec"}}

	env := engineframework.BuildTestEnv(input, spec.Env)

	require.Equal(t, map[string]string{
		"KUBECONFIG":                             "/tmp/kubeconfig",
		"FORGE_TESTENV_TMPDIR":                   "/tmp/test-123",
		"FORGE_ARTIFACT_TESTENV_KIND_KUBECONFIG": "/tmp/test-123/kubeconfig",
		"FORGE_METADATA_CLUSTER_NAME":            "kind-test",
		"RUNNER_VAR":                             "spec",
	}, env)
}
