package context

import (
	"testing"

	"gotest.tools/v3/golden"
)

func TestShow(t *testing.T) {
	cli := makeFakeCli(t)
	createTestContext(t, cli, "current", nil)
	cli.SetCurrentContext("current")

	cli.OutBuffer().Reset()
	runShow(cli)
	golden.Assert(t, cli.OutBuffer().String(), "show.golden")
}
