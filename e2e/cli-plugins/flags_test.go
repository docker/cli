package cliplugins

import (
	"testing"

	"gotest.tools/icmd"
)

// TestRunGoodArgument ensures correct behaviour when running a valid plugin with an `--argument`.
func TestRunGoodArgument(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("helloworld", "--who", "Cleveland"))
	res.Assert(t, icmd.Expected{
		ExitCode: 0,
		Out:      "Hello Cleveland!",
	})
}
