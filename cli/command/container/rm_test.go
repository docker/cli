package container

import (
	"context"
	"fmt"
	"io"
	"sort"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/errdefs"
	"gotest.tools/v3/assert"
)

func TestRemoveForce(t *testing.T) {
	var removed []string

	cli := test.NewFakeCli(&fakeClient{
		containerRemoveFunc: func(ctx context.Context, container string, options types.ContainerRemoveOptions) error {
			removed = append(removed, container)
			if container == "nosuchcontainer" {
				return errdefs.NotFound(fmt.Errorf("Error: No such container: " + container))
			}
			return nil
		},
		Version: "1.36",
	})
	cmd := NewRmCommand(cli)
	cmd.SetOut(io.Discard)

	t.Run("without force", func(t *testing.T) {
		cmd.SetArgs([]string{"nosuchcontainer", "mycontainer"})
		removed = []string{}
		assert.ErrorContains(t, cmd.Execute(), "No such container")
		sort.Strings(removed)
		assert.DeepEqual(t, removed, []string{"mycontainer", "nosuchcontainer"})
	})
	t.Run("with force", func(t *testing.T) {
		cmd.SetArgs([]string{"--force", "nosuchcontainer", "mycontainer"})
		removed = []string{}
		assert.NilError(t, cmd.Execute())
		sort.Strings(removed)
		assert.DeepEqual(t, removed, []string{"mycontainer", "nosuchcontainer"})
	})
}
