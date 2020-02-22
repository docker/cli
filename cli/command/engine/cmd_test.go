package engine

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestNewEngineCommand(t *testing.T) {
	cmd := NewEngineCommand(testCli)

	subcommands := cmd.Commands()
	assert.Assert(t, len(subcommands) == 3)
}
