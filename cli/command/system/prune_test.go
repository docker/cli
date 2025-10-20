package system

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	// Make sure pruners are registered for tests (they're included automatically when building).
	_ "github.com/docker/cli/cli/command/builder"
	_ "github.com/docker/cli/cli/command/container"
	_ "github.com/docker/cli/cli/command/image"
	_ "github.com/docker/cli/cli/command/network"
	_ "github.com/docker/cli/cli/command/volume"
)

func TestPrunePromptFilters(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{version: "1.51"})
	cli.SetConfigFile(&configfile.ConfigFile{
		PruneFilters: []string{"label!=never=remove-me", "label=remove=me"},
	})
	cmd := newPruneCommand(cli)
	cmd.SetArgs([]string{"--filter", "until=24h", "--filter", "label=hello-world", "--filter", "label!=foo=bar", "--filter", "label=bar=baz"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

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
		containerPruneFunc: func(context.Context, client.ContainerPruneOptions) (client.ContainerPruneResult, error) {
			return client.ContainerPruneResult{}, errors.New("fakeClient containerPruneFunc should not be called")
		},
		networkPruneFunc: func(context.Context, client.NetworkPruneOptions) (client.NetworkPruneResult, error) {
			return client.NetworkPruneResult{}, errors.New("fakeClient networkPruneFunc should not be called")
		},
	})

	cmd := newPruneCommand(cli)
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	test.TerminatePrompt(ctx, t, cmd, cli)
}
