package system

import (
	"context"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestPrunePromptPre131DoesNotIncludeBuildCache(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{version: "1.30"})
	cmd := newPruneCommand(cli)
	cmd.SetArgs([]string{})
	assert.ErrorContains(t, cmd.Execute(), "system prune has been cancelled")
	expected := `WARNING! This will remove:
  - all stopped containers
  - all networks not used by at least one container
  - all dangling images

Are you sure you want to continue? [y/N] `
	assert.Check(t, is.Equal(expected, cli.OutBuffer().String()))
}

func TestPrunePromptFilters(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{version: "1.31"})
	cli.SetConfigFile(&configfile.ConfigFile{
		PruneFilters: []string{"label!=never=remove-me", "label=remove=me"},
	})
	cmd := newPruneCommand(cli)
	cmd.SetArgs([]string{"--filter", "until=24h", "--filter", "label=hello-world", "--filter", "label!=foo=bar", "--filter", "label=bar=baz"})

	assert.ErrorContains(t, cmd.Execute(), "system prune has been cancelled")
	expected := `WARNING! This will remove:
  - all stopped containers
  - all networks not used by at least one container
  - all dangling images
  - unused build cache

  Items to be pruned will be filtered with:
  - label!=foo=bar
  - label!=never=remove-me
  - label=bar=baz
  - label=hello-world
  - label=remove=me
  - until=24h

Are you sure you want to continue? [y/N] `
	assert.Check(t, is.Equal(expected, cli.OutBuffer().String()))
}

func TestSystemPrunePromptTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cli := test.NewFakeCli(&fakeClient{
		containerPruneFunc: func(ctx context.Context, pruneFilters filters.Args) (types.ContainersPruneReport, error) {
			return types.ContainersPruneReport{}, errors.New("fakeClient containerPruneFunc should not be called")
		},
		networkPruneFunc: func(ctx context.Context, pruneFilters filters.Args) (types.NetworksPruneReport, error) {
			return types.NetworksPruneReport{}, errors.New("fakeClient networkPruneFunc should not be called")
		},
	})

	cmd := newPruneCommand(cli)
	test.TerminatePrompt(ctx, t, cmd, cli)
}
