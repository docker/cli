package builder

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/build"
)

func TestBuilderPromptTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	fakeCli := test.NewFakeCli(&fakeClient{
		builderPruneFunc: func(ctx context.Context, opts build.CachePruneOptions) (*build.CachePruneReport, error) {
			return nil, errors.New("fakeClient builderPruneFunc should not be called")
		},
	})
	cmd := NewPruneCommand(fakeCli)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	test.TerminatePrompt(ctx, t, cmd, fakeCli)
}
