package forgepath

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The answer becomes an argument to a go command, so it has to agree with
// the workspace that command will actually use. Deciding from go.work on
// disk while the command runs with GOWORK=off produced "go run <module>"
// with no version in a directory with no go.mod, and the failure named the
// module rather than the workspace that was switched off. forge clone into
// a fresh directory hit it every time.
func TestGOWORKOffMeansThereIsNoWorkspace(t *testing.T) {
	root := t.TempDir()
	member := filepath.Join(root, "member")
	require.NoError(t, os.MkdirAll(member, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.work"),
		[]byte("go 1.26\n\nuse ./member\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(member, "go.mod"),
		[]byte("module example.com/member\n\ngo 1.26\n"), 0o600))

	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(member))

	t.Cleanup(func() { _ = os.Chdir(cwd) })

	t.Setenv("GOWORK", "")
	require.True(t, IsWorkspaceModule("example.com/member"),
		"the workspace carries it and go will read the workspace")

	t.Setenv("GOWORK", "off")
	require.False(t, IsWorkspaceModule("example.com/member"),
		"go will not read the workspace, so neither may this")
}

func TestAModuleTheWorkspaceDoesNotCarry(t *testing.T) {
	root := t.TempDir()
	member := filepath.Join(root, "member")
	require.NoError(t, os.MkdirAll(member, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.work"),
		[]byte("go 1.26\n\nuse ./member\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(member, "go.mod"),
		[]byte("module example.com/member\n\ngo 1.26\n"), 0o600))

	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(member))

	t.Cleanup(func() { _ = os.Chdir(cwd) })
	t.Setenv("GOWORK", "")

	require.False(t, IsWorkspaceModule("example.com/other"))
	// A package inside the member is inside the workspace.
	require.True(t, IsWorkspaceModule("example.com/member/cmd/thing"))
}

func TestNoGoWorkAtAll(t *testing.T) {
	dir := t.TempDir()

	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))

	t.Cleanup(func() { _ = os.Chdir(cwd) })
	t.Setenv("GOWORK", "")

	require.False(t, IsWorkspaceModule("example.com/member"))
}
