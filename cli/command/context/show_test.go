package context

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestShow(t *testing.T) {
	cli := makeFakeCli(t)
	createTestContext(t, cli, "current")
	cli.SetCurrentContext("current")

	cli.OutBuffer().Reset()
	assert.NilError(t, runShow(cli))
	golden.Assert(t, cli.OutBuffer().String(), "show.golden")

}
