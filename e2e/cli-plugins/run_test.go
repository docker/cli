package cliplugins

import (
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/golden"
	"gotest.tools/icmd"
)

// TestRunNonexisting ensures correct behaviour when running a nonexistent plugin.
func TestRunNonexisting(t *testing.T) {
	run, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("nonexistent"))
	res.Assert(t, icmd.Expected{
		ExitCode: 1,
	})
	assert.Assert(t, is.Equal(res.Stdout(), ""))
	golden.Assert(t, res.Stderr(), "docker-nonexistent-err.golden")
}

// TestRunBad ensures correct behaviour when running an existent but invalid plugin
func TestRunBad(t *testing.T) {
	run, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("badmeta"))
	res.Assert(t, icmd.Expected{
		ExitCode: 1,
	})
	assert.Assert(t, is.Equal(res.Stdout(), ""))
	golden.Assert(t, res.Stderr(), "docker-badmeta-err.golden")
}

// TestRunGood ensures correct behaviour when running a valid plugin
func TestRunGood(t *testing.T) {
	run, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("helloworld"))
	res.Assert(t, icmd.Expected{
		ExitCode: 0,
		Out:      "Hello World!",
	})
}

// TestRunGoodSubcommand ensures correct behaviour when running a valid plugin with a subcommand
func TestRunGoodSubcommand(t *testing.T) {
	run, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("helloworld", "goodbye"))
	res.Assert(t, icmd.Expected{
		ExitCode: 0,
		Out:      "Goodbye World!",
	})
}
