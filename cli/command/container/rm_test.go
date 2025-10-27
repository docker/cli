package container

import (
	"context"
	"errors"
	"io"
	"sort"
	"sync"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
)

func TestRemoveForce(t *testing.T) {
	for _, tc := range []struct {
		name        string
		args        []string
		expectedErr string
	}{
		{name: "without force", args: []string{"nosuchcontainer", "mycontainer"}, expectedErr: "no such container"},
		{name: "with force", args: []string{"--force", "nosuchcontainer", "mycontainer"}, expectedErr: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var removed []string
			mutex := new(sync.Mutex)

			cli := test.NewFakeCli(&fakeClient{
				containerRemoveFunc: func(ctx context.Context, container string, options client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
					// containerRemoveFunc is called in parallel for each container
					// by the remove command so append must be synchronized.
					mutex.Lock()
					removed = append(removed, container)
					mutex.Unlock()

					if container == "nosuchcontainer" {
						return client.ContainerRemoveResult{}, notFound(errors.New("Error: no such container: " + container))
					}
					return client.ContainerRemoveResult{}, nil
				},
				Version: "1.36",
			})
			cmd := newRmCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if tc.expectedErr != "" {
				assert.ErrorContains(t, err, tc.expectedErr)
			} else {
				assert.NilError(t, err)
			}
			sort.Strings(removed)
			assert.DeepEqual(t, removed, []string{"mycontainer", "nosuchcontainer"})
		})
	}
}
