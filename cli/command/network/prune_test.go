package network

import (
	"context"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
)

func TestNetworkPrunePromptTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cli := test.NewFakeCli(&fakeClient{
		networkPruneFunc: func(ctx context.Context, pruneFilters filters.Args) (types.NetworksPruneReport, error) {
			return types.NetworksPruneReport{}, errors.New("fakeClient networkPruneFunc should not be called")
		},
	})
	cmd := NewPruneCommand(cli)
	test.TerminatePrompt(ctx, t, cmd, cli, func(t *testing.T, err error) {
		t.Helper()
		assert.ErrorIs(t, err, command.ErrPromptTerminated)
	})
}
