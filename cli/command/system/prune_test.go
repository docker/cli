package system

import (
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/stretchr/testify/assert"
)

func TestPrunePromptPre131(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{version: "1.30"})
	cmd := newPruneCommand(cli)
	assert.NoError(t, cmd.Execute())
	assert.NotContains(t, cli.OutBuffer().String(), "all build cache")
}
