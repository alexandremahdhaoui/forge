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

package engineframework

import (
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/stretchr/testify/require"
)

func testenvEnvFixture() map[string]string {
	return map[string]string{
		"KUBECONFIG":       "/tmp/kubeconfig",
		"TESTENV_LCR_FQDN": "localhost:5000",
		"SECRET_TOKEN":     "secret",
	}
}

func TestEveryTestenvVariableReachesTheTestsWhenNoPolicyIsSet(t *testing.T) {
	env := BuildTestEnv(mcptypes.RunInput{TestenvEnv: testenvEnvFixture()}, nil)

	require.Equal(t, testenvEnvFixture(), env)
}

func TestAWhitelistKeepsOnlyTheNamedVariablesAndIgnoresNamesTheTestenvNeverSet(t *testing.T) {
	env := BuildTestEnv(mcptypes.RunInput{
		TestenvEnv:     testenvEnvFixture(),
		EnvPropagation: &forge.EnvPropagation{Whitelist: []string{"KUBECONFIG", "NOT_SET"}},
	}, nil)

	require.Equal(t, map[string]string{"KUBECONFIG": "/tmp/kubeconfig"}, env)
}

func TestABlacklistDropsOnlyTheNamedVariables(t *testing.T) {
	env := BuildTestEnv(mcptypes.RunInput{
		TestenvEnv:     testenvEnvFixture(),
		EnvPropagation: &forge.EnvPropagation{Blacklist: []string{"SECRET_TOKEN"}},
	}, nil)

	require.Equal(t, map[string]string{
		"KUBECONFIG":       "/tmp/kubeconfig",
		"TESTENV_LCR_FQDN": "localhost:5000",
	}, env)
}

func TestADisabledPolicyPassesNoTestenvVariableButStillExportsTheForgeMetadata(t *testing.T) {
	env := BuildTestEnv(mcptypes.RunInput{
		TestenvEnv:      testenvEnvFixture(),
		EnvPropagation:  &forge.EnvPropagation{Disabled: true},
		TestenvTmpDir:   "/tmp/testenv",
		TestenvMetadata: map[string]string{"cluster-name": "kind"},
	}, nil)

	require.Equal(t, map[string]string{
		"FORGE_TESTENV_TMPDIR":        "/tmp/testenv",
		"FORGE_METADATA_CLUSTER_NAME": "kind",
	}, env)
}

func TestADisabledPolicyBeatsANonEmptyWhitelist(t *testing.T) {
	env := BuildTestEnv(mcptypes.RunInput{
		TestenvEnv:     testenvEnvFixture(),
		EnvPropagation: &forge.EnvPropagation{Disabled: true, Whitelist: []string{"KUBECONFIG"}},
	}, nil)

	require.Empty(t, env)
}

func TestAnEmptyWhitelistAndAnEmptyBlacklistMeanNoFiltering(t *testing.T) {
	env := BuildTestEnv(mcptypes.RunInput{
		TestenvEnv:     testenvEnvFixture(),
		EnvPropagation: &forge.EnvPropagation{Whitelist: []string{}, Blacklist: []string{}},
	}, nil)

	require.Equal(t, testenvEnvFixture(), env)
}

func TestArtifactFilesAreExportedAsAbsolutePathsUnderTheTestenvTmpDir(t *testing.T) {
	env := BuildTestEnv(mcptypes.RunInput{
		TestenvTmpDir: "/tmp/testenv",
		ArtifactFiles: map[string]string{"kubeconfig.yaml": "kind/kubeconfig"},
	}, nil)

	require.Equal(t, map[string]string{
		"FORGE_TESTENV_TMPDIR":           "/tmp/testenv",
		"FORGE_ARTIFACT_KUBECONFIG_YAML": "/tmp/testenv/kind/kubeconfig",
	}, env)
}

func TestArtifactFilesStayRelativeWhenThereIsNoTestenvTmpDir(t *testing.T) {
	env := BuildTestEnv(mcptypes.RunInput{
		ArtifactFiles: map[string]string{"ca": "certs/ca.crt"},
	}, nil)

	require.Equal(t, map[string]string{"FORGE_ARTIFACT_CA": "certs/ca.crt"}, env)
}

func TestTheRunnerEnvOverridesTheTestenvAndTheSpecEnvOverridesBoth(t *testing.T) {
	env := BuildTestEnv(mcptypes.RunInput{
		TestenvEnv:      map[string]string{"SHARED": "from-testenv", "ONLY_TESTENV": "testenv"},
		TestenvMetadata: map[string]string{"shared": "from-metadata"},
		Env:             map[string]string{"SHARED": "from-runner", "FORGE_METADATA_SHARED": "from-runner", "ONLY_RUNNER": "runner"},
	}, map[string]string{"SHARED": "from-spec", "ONLY_SPEC": "spec"})

	require.Equal(t, map[string]string{
		"SHARED":                "from-spec",
		"FORGE_METADATA_SHARED": "from-runner",
		"ONLY_TESTENV":          "testenv",
		"ONLY_RUNNER":           "runner",
		"ONLY_SPEC":             "spec",
	}, env)
}

func TestNormalizeEnvKeyUppercasesLettersAndReplacesEverythingElseWithAnUnderscore(t *testing.T) {
	require.Equal(t, "CLUSTER_NAME_V1_2", NormalizeEnvKey("cluster-name.v1 2"))
	require.Equal(t, "ABC123", NormalizeEnvKey("abc123"))
	require.Equal(t, "", NormalizeEnvKey(""))
}
