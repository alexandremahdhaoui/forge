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

package forgepath

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func chdirIntoWorkspaceListing(t *testing.T, memberModules ...string) string {
	t.Helper()

	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)

	goWork := "go 1.26\n\nuse (\n"
	for i, module := range memberModules {
		dir := filepath.Join(root, "member"+string(rune('a'+i)))
		writeFakeForgeCheckout(t, dir, module)
		goWork += "\t./" + filepath.Base(dir) + "\n"
	}
	goWork += ")\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.work"), []byte(goWork), 0o600))

	caller := filepath.Join(root, "membera")
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(caller))
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	t.Setenv("GOWORK", "")
	t.Setenv("FORGE_RUN_LOCAL_ENABLED", "")
	t.Setenv("FORGE_RUN_LOCAL_BASEDIR", "")

	return filepath.Join(root, "memberb")
}

func writeFakeForgeCheckout(t *testing.T, dir, module string) {
	t.Helper()

	for _, command := range []string{"forge", "go-build"} {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd", command), 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "cmd", command, "main.go"),
			[]byte("package main\n\nfunc main() {}\n"), 0o600))
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module "+module+"\n\ngo 1.26\n"), 0o600))
}

func fakeForgeCheckout(t *testing.T) string {
	t.Helper()

	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	writeFakeForgeCheckout(t, dir, forgeModule)

	return dir
}

func stampSourceDir(t *testing.T, dir string) {
	t.Helper()

	previous := SourceDir
	SourceDir = dir
	t.Cleanup(func() { SourceDir = previous })
}

const describeVersion = "v0.50.10-16-gda6c582"

func TestADescribeVersionWithAStampedSourceDirBuildsTheEngineFromThatDirAndKeepsTheCallersWorkingDirectory(t *testing.T) {
	chdirIntoWorkspaceListing(t, "example.com/caller", "example.com/other")
	forgeDir := fakeForgeCheckout(t)
	stampSourceDir(t, forgeDir)
	caller, err := os.Getwd()
	require.NoError(t, err)

	command, args, err := EngineCommand("go-build", describeVersion)
	require.NoError(t, err)

	require.Equal(t, filepath.Join(forgeDir, "build", "local-engines", "go-build"), command)
	require.Nil(t, args)
	require.FileExists(t, command)
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.Equal(t, caller, cwd)
}

func TestADescribeVersionWithNoStampAndTheBaseDirVariableSetUsesTheVariable(t *testing.T) {
	chdirIntoWorkspaceListing(t, "example.com/caller", "example.com/other")
	forgeDir := fakeForgeCheckout(t)
	stampSourceDir(t, "")
	t.Setenv("FORGE_RUN_LOCAL_BASEDIR", forgeDir)

	command, args, err := EngineCommand("go-build", describeVersion)
	require.NoError(t, err)

	require.Equal(t, filepath.Join(forgeDir, "build", "local-engines", "go-build"), command)
	require.Nil(t, args)
	require.FileExists(t, command)
}

func TestADescribeVersionWithNoStampAndNoVariableIsRefusedNamingTheVersionAndBothSources(t *testing.T) {
	chdirIntoWorkspaceListing(t, "example.com/caller", "example.com/other")
	stampSourceDir(t, "")

	_, _, err := EngineCommand("go-build", describeVersion)
	require.Error(t, err)
	require.Contains(t, err.Error(), describeVersion)
	require.Contains(t, err.Error(), "not a release tag")
	require.Contains(t, err.Error(), "no stamped source directory")
	require.Contains(t, err.Error(), "FORGE_RUN_LOCAL_BASEDIR is unset")
	require.Contains(t, err.Error(), "go-build")
}

func TestAStampedDirThatIsNotAForgeCheckoutIsRefusedNamingTheDir(t *testing.T) {
	otherDir := chdirIntoWorkspaceListing(t, "example.com/caller", "example.com/other")
	stampSourceDir(t, otherDir)

	_, _, err := EngineCommand("go-build", describeVersion)
	require.Error(t, err)
	require.Contains(t, err.Error(), otherDir)
	require.Contains(t, err.Error(), "stamped source directory")
}

func TestATagKeepsTheAtTagFormOutsideAWorkspace(t *testing.T) {
	chdirIntoWorkspaceListing(t, "example.com/caller", "example.com/other")
	stampSourceDir(t, "")

	command, args, err := EngineCommand("go-build", "v0.50.0")
	require.NoError(t, err)

	require.Equal(t, "go", command)
	require.Equal(t, []string{"run", forgeModule + "/cmd/go-build@v0.50.0"}, args)
}

func TestADirtyTagIsTrimmedAsBefore(t *testing.T) {
	chdirIntoWorkspaceListing(t, "example.com/caller", "example.com/other")
	stampSourceDir(t, "")

	_, args, err := EngineCommand("go-build", "v0.50.0-dirty")
	require.NoError(t, err)
	require.Equal(t, []string{"run", forgeModule + "/cmd/go-build@v0.50.0"}, args)

	_, args, err = EngineCommand("go-build", "v0.50.0+dirty")
	require.NoError(t, err)
	require.Equal(t, []string{"run", forgeModule + "/cmd/go-build@v0.50.0"}, args)
}

func TestDevIsNotATag(t *testing.T) {
	require.False(t, IsReleaseTag("dev"))
}

func TestOnlyASemverTagWithoutADescribeSuffixIsAReleaseTag(t *testing.T) {
	for version, want := range map[string]bool{
		"dev":                                    false,
		"v0.50.10-16-gda6c582":                   false,
		"v0.50.11-0.20260917090238-38080aa3993a": false,
		"v0.50.0":                                true,
		"v1.0.0-rc1":                             true,
		"v0.50.0+dirty":                          true,
	} {
		require.Equal(t, want, IsReleaseTag(version), version)
	}
}

func TestAnEmptyForgeVersionOutsideAWorkspaceIsRefusedByName(t *testing.T) {
	chdirIntoWorkspaceListing(t, "example.com/caller", "example.com/other")
	stampSourceDir(t, "")

	_, _, err := EngineCommand("go-build", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "forge version cannot be empty")
}

func TestAnEngineBuiltFromSourceCarriesTheStampedSourceDirItsDetectorIsBuiltFrom(t *testing.T) {
	chdirIntoWorkspaceListing(t, "example.com/caller", "example.com/other")
	forgeDir := fakeForgeCheckout(t)
	require.NoError(t, os.MkdirAll(filepath.Join(forgeDir, "internal", "forgepath"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(forgeDir, "internal", "forgepath", "forgepath.go"),
		[]byte("package forgepath\n\nvar SourceDir string\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(forgeDir, "cmd", "go-build", "main.go"),
		[]byte("package main\n\nimport (\n\t\"fmt\"\n\n\t\""+forgeModule+"/internal/forgepath\"\n)\n\nfunc main() { fmt.Print(forgepath.SourceDir) }\n"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(forgeDir, "cmd", "go-dependency-detector"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(forgeDir, "cmd", "go-dependency-detector", "main.go"),
		[]byte("package main\n\nfunc main() {}\n"), 0o600))

	engine, err := BuildEngineFromSource(filepath.Join(forgeDir, "cmd", "go-build"), "go-build")
	require.NoError(t, err)

	stamped, err := exec.Command(engine).Output()
	require.NoError(t, err)
	require.Equal(t, forgeDir, string(stamped))

	stampSourceDir(t, string(stamped))

	detector, args, err := EngineCommand("go-dependency-detector", "dev")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(forgeDir, "build", "local-engines", "go-dependency-detector"), detector)
	require.Nil(t, args)
	require.FileExists(t, detector)
}

func TestABuiltinRunsUnversionedFromTheWorkspaceCheckoutWhenGoWorkCarriesForge(t *testing.T) {
	chdirIntoWorkspaceListing(t, "example.com/caller", forgeModule)

	command, args, err := EngineCommand("go-build", "v1.2.3")
	require.NoError(t, err)

	require.Equal(t, "go", command)
	require.Equal(t, []string{"run", forgeModule + "/cmd/go-build"}, args)
}

func TestABuiltinKeepsThePinnedVersionWhenGoWorkDoesNotCarryForge(t *testing.T) {
	chdirIntoWorkspaceListing(t, "example.com/caller", "example.com/other")

	command, args, err := EngineCommand("go-build", "v1.2.3")
	require.NoError(t, err)

	require.Equal(t, "go", command)
	require.Equal(t, []string{"run", forgeModule + "/cmd/go-build@v1.2.3"}, args)
}

func TestTheLocalFlagForcesABuildFromTheNamedCheckoutEvenInsideAWorkspace(t *testing.T) {
	forgeDir := chdirIntoWorkspaceListing(t, "example.com/caller", forgeModule)
	t.Setenv("FORGE_RUN_LOCAL_ENABLED", "true")
	t.Setenv("FORGE_RUN_LOCAL_BASEDIR", forgeDir)

	command, args, err := EngineCommand("go-build", "v1.2.3")
	require.NoError(t, err)

	require.Equal(t, filepath.Join(forgeDir, "build", "local-engines", "go-build"), command)
	require.Nil(t, args)
	require.FileExists(t, command)
}

func TestAnEmptyBuiltinNameIsRefused(t *testing.T) {
	chdirIntoWorkspaceListing(t, "example.com/caller", forgeModule)

	_, _, err := EngineCommand("", "v1.2.3")
	require.Error(t, err)
}

func TestASiblingModuleNeverMatchesAWorkspaceMemberByPrefix(t *testing.T) {
	chdirIntoWorkspaceListing(t, "example.com/caller", forgeModule)

	require.True(t, IsWorkspaceModule(forgeModule))
	require.True(t, IsWorkspaceModule(forgeModule+"/cmd/go-build"))
	require.False(t, IsWorkspaceModule(forgeModule+"-ci"))
	require.False(t, IsWorkspaceModule(forgeModule+"-ci/cmd/forge-ci"))
}
