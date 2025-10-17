package network

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/client"
)

func TestNetworkPrunePromptTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cli := test.NewFakeCli(&fakeClient{
		networkPruneFunc: func(ctx context.Context, opts client.NetworkPruneOptions) (client.NetworkPruneResult, error) {
			return client.NetworkPruneResult{}, errors.New("fakeClient networkPruneFunc should not be called")
		},
	})
	cmd := newPruneCommand(cli)
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	test.TerminatePrompt(ctx, t, cmd, cli)
}
