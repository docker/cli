package system

import (
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestPrunePromptPre131DoesNotIncludeBuildCache(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{version: "1.30"})
	cmd := newPruneCommand(cli)
	assert.NilError(t, cmd.Execute())
	expected := `WARNING! This will remove:
        - all stopped containers
        - all networks not used by at least one container
        - all dangling images
Are you sure you want to continue? [y/N] `
	assert.Check(t, is.Equal(expected, cli.OutBuffer().String()))

}
