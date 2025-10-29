package container

import (
	"strings"
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/icmd"
)

func TestContainerRename(t *testing.T) {
	oldName := "old_name_" + t.Name()
	res := icmd.RunCommand("docker", "run", "-d", "--name", oldName, fixtures.AlpineImage, "sleep", "60")
	res.Assert(t, icmd.Success)
	cID := strings.TrimSpace(res.Stdout())
	t.Cleanup(func() {
		icmd.RunCommand("docker", "container", "rm", "-f", cID).Assert(t, icmd.Success)
	})

	newName := "new_name_" + t.Name()
	res = icmd.RunCommand("docker", "container", "rename", oldName, newName)
	res.Assert(t, icmd.Success)

	res = icmd.RunCommand("docker", "container", "inspect", "--format", "{{.Name}}", cID)
	res.Assert(t, icmd.Success)
	assert.Equal(t, "/"+newName, strings.TrimSpace(res.Stdout()))
}

func TestContainerRenameEmptyOldName(t *testing.T) {
	res := icmd.RunCommand("docker", "container", "rename", "", "newName")
	res.Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "invalid container name or ID: value is empty",
	})
}

func TestContainerRenameEmptyNewName(t *testing.T) {
	oldName := "old_name_" + t.Name()
	res := icmd.RunCommand("docker", "run", "-d", "--name", oldName, fixtures.AlpineImage, "sleep", "60")
	res.Assert(t, icmd.Success)
	cID := strings.TrimSpace(res.Stdout())
	t.Cleanup(func() {
		icmd.RunCommand("docker", "container", "rm", "-f", cID).Assert(t, icmd.Success)
	})

	res = icmd.RunCommand("docker", "container", "rename", oldName, "")
	res.Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "new name cannot be blank",
	})
}
