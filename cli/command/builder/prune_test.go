package builder

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/client"
)

func TestBuilderPromptTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cli := test.NewFakeCli(&fakeClient{
		builderPruneFunc: func(ctx context.Context, opts client.BuildCachePruneOptions) (client.BuildCachePruneResult, error) {
			return client.BuildCachePruneResult{}, errors.New("fakeClient builderPruneFunc should not be called")
		},
	})
	cmd := newPruneCommand(cli)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	test.TerminatePrompt(ctx, t, cmd, cli)
}
