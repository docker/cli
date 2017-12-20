package system

import (
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/stretchr/testify/assert"
)

func TestPrunePromptPre131DoesNotIncludeBuildCache(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{version: "1.30"})
	cmd := newPruneCommand(cli)
	assert.NoError(t, cmd.Execute())
	expected := `WARNING! This will remove:
        - all stopped containers
        - all networks not used by at least one container
        - all dangling images
Are you sure you want to continue? [y/N] `
	assert.Equal(t, expected, cli.OutBuffer().String())

}
