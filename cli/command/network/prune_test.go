package network // import "docker.com/cli/v28/cli/command/network"

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/v28/internal/test"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
)

func TestNetworkPrunePromptTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cli := test.NewFakeCli(&fakeClient{
		networkPruneFunc: func(ctx context.Context, pruneFilters filters.Args) (network.PruneReport, error) {
			return network.PruneReport{}, errors.New("fakeClient networkPruneFunc should not be called")
		},
	})
	cmd := NewPruneCommand(cli)
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	test.TerminatePrompt(ctx, t, cmd, cli)
}
