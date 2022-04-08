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
	var (
		removed1 []string
		removed2 []string
	)

	cli := test.NewFakeCli(&fakeClient{
		containerRemoveFunc: func(ctx context.Context, container string, options types.ContainerRemoveOptions) error {
			removed1 = append(removed1, container)
			removed2 = append(removed2, container)
			if container == "nosuchcontainer" {
				return errdefs.NotFound(fmt.Errorf("Error: no such container: " + container))
			}
			return nil
		},
		Version: "1.36",
	})
	cmd := NewRmCommand(cli)
	cmd.SetOut(io.Discard)

	t.Run("without force", func(t *testing.T) {
		cmd.SetArgs([]string{"nosuchcontainer", "mycontainer"})
		removed1 = []string{}
		assert.ErrorContains(t, cmd.Execute(), "no such container")
		sort.Strings(removed1)
		assert.DeepEqual(t, removed1, []string{"mycontainer", "nosuchcontainer"})
	})
	t.Run("with force", func(t *testing.T) {
		cmd.SetArgs([]string{"--force", "nosuchcontainer", "mycontainer"})
		removed2 = []string{}
		assert.NilError(t, cmd.Execute())
		sort.Strings(removed2)
		assert.DeepEqual(t, removed2, []string{"mycontainer", "nosuchcontainer"})
	})
}
