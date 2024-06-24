package container

import (
	"context"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/pkg/errors"
)

func TestContainerPrunePromptTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cli := test.NewFakeCli(&fakeClient{
		containerPruneFunc: func(ctx context.Context, pruneFilters filters.Args) (container.PruneReport, error) {
			return container.PruneReport{}, errors.New("fakeClient containerPruneFunc should not be called")
		},
	})
	cmd := NewPruneCommand(cli)
	test.TerminatePrompt(ctx, t, cmd, cli)
}
