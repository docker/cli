package builder

import (
	"context"
	"errors"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"gotest.tools/v3/assert"
)

func TestBuilderPromptTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cli := test.NewFakeCli(&fakeClient{
		builderPruneFunc: func(ctx context.Context, opts types.BuildCachePruneOptions) (*types.BuildCachePruneReport, error) {
			return nil, errors.New("fakeClient builderPruneFunc should not be called")
		},
	})
	cmd := NewPruneCommand(cli)
	test.TerminatePrompt(ctx, t, cmd, cli, func(t *testing.T, err error) {
		t.Helper()
		assert.ErrorIs(t, err, command.ErrPromptTerminated)
	})
}
