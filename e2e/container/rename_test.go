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
	result := icmd.RunCommand("docker", "run", "-d", "--name", oldName, fixtures.AlpineImage, "sleep", "60")
	result.Assert(t, icmd.Success)
	containerID := strings.TrimSpace(result.Stdout())

	newName := "new_name_" + t.Name()
	renameResult := icmd.RunCommand("docker", "container", "rename", oldName, newName)
	renameResult.Assert(t, icmd.Success)

	inspectResult := icmd.RunCommand("docker", "inspect", "--format", "{{.Name}}", containerID)
	inspectResult.Assert(t, icmd.Success)
	assert.Equal(t, "/"+newName, strings.TrimSpace(inspectResult.Stdout()))
}
