package engine

import (
	"testing"

	"gotest.tools/assert"
)

func TestNewEngineCommand(t *testing.T) {
	cmd := NewEngineCommand(testCli)

	subcommands := cmd.Commands()
	assert.Assert(t, len(subcommands) == 5)
}
