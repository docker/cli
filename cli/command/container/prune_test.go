package container

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
)

func TestContainerPrunePromptTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cli := test.NewFakeCli(&fakeClient{
		containerPruneFunc: func(ctx context.Context, opts client.ContainerPruneOptions) (client.ContainerPruneResult, error) {
			return client.ContainerPruneResult{}, errors.New("fakeClient containerPruneFunc should not be called")
		},
	})
	cmd := newPruneCommand(cli)
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	test.TerminatePrompt(ctx, t, cmd, cli)
}

func TestContainerPruneProgressBeforeReport(t *testing.T) {
	var cli *test.FakeCli
	cli = test.NewFakeCli(&fakeClient{
		containerPruneFunc: func(_ context.Context, opts client.ContainerPruneOptions) (client.ContainerPruneResult, error) {
			assert.Assert(t, opts.OnProgress != nil)
			assert.NilError(t, opts.OnProgress(client.PruneProgress{ID: "removed", Action: "deleted"}))
			assert.Equal(t, cli.OutBuffer().String(), "Deleted Containers:\nremoved\n")
			return client.ContainerPruneResult{Report: container.PruneReport{
				ContainersDeleted: []string{"removed"},
				SpaceReclaimed:    42,
			}}, nil
		},
	})
	cmd := newPruneCommand(cli)
	cmd.SetArgs([]string{"--force"})
	assert.NilError(t, cmd.Execute())
	assert.Equal(t, cli.OutBuffer().String(), "Deleted Containers:\nremoved\n\nTotal reclaimed space: 42B\n")
}
