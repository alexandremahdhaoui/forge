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
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/stretchr/testify/require"
)

func storeWithThreeReports(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "artifact-store.yaml")
	store, err := forge.ReadOrCreateArtifactStore(path)
	require.NoError(t, err)

	forge.AddOrUpdateTestReport(&store, &forge.TestReport{ID: "report-in-env", Stage: "integration", Status: "passed", TestenvID: "env-1"})
	forge.AddOrUpdateTestReport(&store, &forge.TestReport{ID: "report-elsewhere", Stage: "integration", Status: "failed", TestenvID: "env-2"})
	forge.AddOrUpdateTestReport(&store, &forge.TestReport{ID: "report-legacy", Stage: "unit", Status: "passed"})
	require.NoError(t, forge.WriteArtifactStore(path, store))

	return path
}

func remainingReportIDs(t *testing.T, path string) []string {
	t.Helper()

	store, err := forge.ReadArtifactStore(path)
	require.NoError(t, err)

	var ids []string
	for _, report := range forge.ListTestReports(&store, "") {
		ids = append(ids, report.ID)
	}

	return ids
}

func TestDeleteEnvRefusesAndListsTheReportsThatRanInTheEnvironment(t *testing.T) {
	path := storeWithThreeReports(t)

	reports, err := guardEnvironmentReports(path, "env-1", false)

	require.Nil(t, reports)
	require.Error(t, err)
	require.Contains(t, err.Error(), "report-in-env  stage=integration  status=passed")
	require.Contains(t, err.Error(), "rerun with --force to remove the environment and its 1 reports")
	require.NotContains(t, err.Error(), "report-elsewhere")
	require.NotContains(t, err.Error(), "report-legacy")
	require.ElementsMatch(t, []string{"report-in-env", "report-elsewhere", "report-legacy"}, remainingReportIDs(t, path))
}

func TestDeleteEnvWithForceRemovesEveryReportOfTheEnvironmentAndNothingElse(t *testing.T) {
	path := storeWithThreeReports(t)

	reports, err := guardEnvironmentReports(path, "env-1", true)
	require.NoError(t, err)
	require.Len(t, reports, 1)

	require.NoError(t, removeEnvironmentReports(path, reports))

	require.ElementsMatch(t, []string{"report-elsewhere", "report-legacy"}, remainingReportIDs(t, path))
}

func TestDeleteEnvWithNoReportsInTheEnvironmentDeletesPlainly(t *testing.T) {
	path := storeWithThreeReports(t)

	reports, err := guardEnvironmentReports(path, "env-empty", false)

	require.NoError(t, err)
	require.Empty(t, reports)
	require.NoError(t, removeEnvironmentReports(path, reports))
	require.ElementsMatch(t, []string{"report-in-env", "report-elsewhere", "report-legacy"}, remainingReportIDs(t, path))
}

func TestAReportWithoutAnEnvironmentFieldIsNeverTouched(t *testing.T) {
	path := storeWithThreeReports(t)

	reports, err := guardEnvironmentReports(path, "", true)

	require.NoError(t, err)
	require.Empty(t, reports)
}

func TestTestAllStillRemovesTheEnvironmentWhenAReportRanInIt(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "artifact-store.yaml")
	store, err := forge.ReadOrCreateArtifactStore(storePath)
	require.NoError(t, err)
	forge.AddOrUpdateTestEnvironment(&store, &forge.TestEnvironment{ID: "env-1", Name: "integration"})
	forge.AddOrUpdateTestReport(&store, &forge.TestReport{ID: "report-in-env", Stage: "integration", Status: "failed", TestenvID: "env-1"})
	require.NoError(t, forge.WriteArtifactStore(storePath, store))

	writeFixtureForgeYaml(t, dir, storePath)

	require.NoError(t, cleanupTestEnvironmentByID(&forge.TestSpec{Name: "integration"}, "env-1"))

	after, err := forge.ReadArtifactStore(storePath)
	require.NoError(t, err)
	_, err = forge.GetTestEnvironment(&after, "env-1")
	require.Error(t, err)
	require.Empty(t, remainingReportIDs(t, storePath))
}

func writeFixtureForgeYaml(t *testing.T, dir, storePath string) {
	t.Helper()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "forge.yaml"),
		[]byte("name: fixture\nartifactStorePath: "+storePath+"\n"), 0o644))
	t.Chdir(dir)
}

func TestARunInAnEnvironmentRecordsTheEnvironmentOnItsReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "artifact-store.yaml")
	writeFixtureForgeYaml(t, dir, path)
	config := &forge.Spec{ArtifactStorePath: path}

	result := map[string]any{"id": "report-1", "stage": "integration", "status": "passed"}
	require.NoError(t, storeTestReportFromResult(result, config, "env-1"))

	store, err := forge.ReadArtifactStore(path)
	require.NoError(t, err)
	report, err := forge.GetTestReport(&store, "report-1")
	require.NoError(t, err)
	require.Equal(t, "env-1", report.TestenvID)
}
