package system

import (
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestPrunePromptPre131DoesNotIncludeBuildCache(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{version: "1.30"})
	cmd := newPruneCommand(cli)
	cmd.SetArgs([]string{})
	assert.NilError(t, cmd.Execute())
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

	assert.NilError(t, cmd.Execute())
	expected := `WARNING! This will remove:
  - all stopped containers
  - all networks not used by at least one container
  - all dangling images
  - all dangling build cache

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
