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
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/stretchr/testify/require"
)

func answerWith(paths ...string) map[string]interface{} {
	files := []interface{}{}
	for _, path := range paths {
		files = append(files, map[string]interface{}{"path": path, "content": "// " + path + "\n"})
	}

	return map[string]interface{}{"files": files}
}

func generateCell(t *testing.T, dir string, answer map[string]interface{}) error {
	t.Helper()

	stub := &stubGeneratorCaller{answer: answer}
	withStubGenerator(t, stub)

	_, err := generate(context.Background(), mcptypes.BuildInput{
		Name: "fixture-gui", Src: dir, Engine: "forge://forge-dev", Force: true,
	})

	return err
}

func TestAnAnsweredPathThatIsNotNamedZzGeneratedFailsNamingThePathAndTheGenerator(t *testing.T) {
	dir := t.TempDir()
	writeKindFixture(t, dir, customKindYaml())

	err := generateCell(t, dir, answerWith("src/hand_written.rs"))

	require.Error(t, err)
	require.Contains(t, err.Error(), "src/hand_written.rs is not named zz_generated")
	require.Contains(t, err.Error(), "forge://example.com/org/gui-gen/engines/gui-gen")
}

func TestAnAnsweredPathInASubdirectoryIsJudgedOnItsBasename(t *testing.T) {
	dir := t.TempDir()
	writeKindFixture(t, dir, customKindYaml())

	require.NoError(t, generateCell(t, dir, answerWith("src/rest/zz_generated_driver.rs")))
	require.FileExists(t, filepath.Join(dir, "src", "rest", "zz_generated_driver.rs"))
}

func TestTheRunnableManifestRecordsEveryPathTheGeneratorAnswered(t *testing.T) {
	dir := t.TempDir()
	writeKindFixture(t, dir, customKindYaml())

	require.NoError(t, generateCell(t, dir, answerWith("zz_generated_a.rs", "src/zz_generated_b.rs")))

	recorded, err := ReadGeneratedFiles(filepath.Join(dir, GeneratedRunnableFile))
	require.NoError(t, err)
	require.Equal(t, []string{"zz_generated_a.rs", "src/zz_generated_b.rs"}, recorded)
}

func TestAFileTheNewAnswerDoesNotHoldIsRemovedOnTheNextRun(t *testing.T) {
	dir := t.TempDir()
	writeKindFixture(t, dir, customKindYaml())

	require.NoError(t, generateCell(t, dir, answerWith("zz_generated_a.rs", "zz_generated_b.rs")))
	require.FileExists(t, filepath.Join(dir, "zz_generated_b.rs"))

	require.NoError(t, generateCell(t, dir, answerWith("zz_generated_a.rs")))

	require.FileExists(t, filepath.Join(dir, "zz_generated_a.rs"))
	require.NoFileExists(t, filepath.Join(dir, "zz_generated_b.rs"))

	recorded, err := ReadGeneratedFiles(filepath.Join(dir, GeneratedRunnableFile))
	require.NoError(t, err)
	require.Equal(t, []string{"zz_generated_a.rs"}, recorded)
}

func TestAStaleFileAlreadyGoneLeavesTheRunQuiet(t *testing.T) {
	dir := t.TempDir()
	writeKindFixture(t, dir, customKindYaml())

	require.NoError(t, generateCell(t, dir, answerWith("zz_generated_a.rs", "zz_generated_b.rs")))
	require.NoError(t, os.Remove(filepath.Join(dir, "zz_generated_b.rs")))

	require.NoError(t, generateCell(t, dir, answerWith("zz_generated_a.rs")))
}

func TestAGeneratorThatDeclaresAManifestAndAnswersNoneFailsNamingTheGenerator(t *testing.T) {
	dir := t.TempDir()
	writeKindFixture(t, dir, customKindYaml())

	answer := answerWith("zz_generated_a.rs")
	answer["manifest"] = true

	err := generateCell(t, dir, answer)

	require.Error(t, err)
	require.Contains(t, err.Error(), GeneratedCellManifestFile)
	require.Contains(t, err.Error(), "forge://example.com/org/gui-gen/engines/gui-gen")
}

func TestAGeneratorThatDeclaresAManifestAndAnswersOneIsAccepted(t *testing.T) {
	dir := t.TempDir()
	writeKindFixture(t, dir, customKindYaml())

	answer := answerWith("zz_generated_a.rs", "src/rest/"+GeneratedCellManifestFile)
	answer["manifest"] = true

	require.NoError(t, generateCell(t, dir, answer))
	require.FileExists(t, filepath.Join(dir, "src", "rest", GeneratedCellManifestFile))
}

func TestAGeneratorThatDeclaresNoManifestNeedsNone(t *testing.T) {
	dir := t.TempDir()
	writeKindFixture(t, dir, customKindYaml())

	require.NoError(t, generateCell(t, dir, answerWith("zz_generated_a.rs")))
}
