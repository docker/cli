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

// TestHelpNonexisting ensures correct behaviour when invoking help on a nonexistent plugin.
func TestHelpNonexisting(t *testing.T) {
	run, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("help", "nonexistent"))
	res.Assert(t, icmd.Expected{
		ExitCode: 1,
	})
	assert.Assert(t, is.Equal(res.Stdout(), ""))
	golden.Assert(t, res.Stderr(), "docker-help-nonexistent-err.golden")
}

// TestNonexistingHelp ensures correct behaviour when invoking a
// nonexistent plugin with `--help`.
func TestNonexistingHelp(t *testing.T) {
	run, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("nonexistent", "--help"))
	res.Assert(t, icmd.Expected{
		ExitCode: 0,
		// This should actually be the whole docker help
		// output, so spot check instead having of a golden
		// with everything in, which will change too frequently.
		Out: "Usage:	docker [OPTIONS] COMMAND\n\nA self-sufficient runtime for containers",
	})
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

// TestHelpBad ensures correct behaviour when invoking help on a existent but invalid plugin.
func TestHelpBad(t *testing.T) {
	run, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("help", "badmeta"))
	res.Assert(t, icmd.Expected{
		ExitCode: 1,
	})
	assert.Assert(t, is.Equal(res.Stdout(), ""))
	golden.Assert(t, res.Stderr(), "docker-help-badmeta-err.golden")
}

// TestBadHelp ensures correct behaviour when invoking an
// existent but invalid plugin with `--help`.
func TestBadHelp(t *testing.T) {
	run, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("badmeta", "--help"))
	res.Assert(t, icmd.Expected{
		ExitCode: 0,
		// This should be literally the whole docker help
		// output, so spot check instead of a golden with
		// everything in which will change all the time.
		Out: "Usage:	docker [OPTIONS] COMMAND\n\nA self-sufficient runtime for containers",
	})
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

// TestHelpGood ensures correct behaviour when invoking help on a
// valid plugin. A global argument is included to ensure it does not
// interfere.
func TestHelpGood(t *testing.T) {
	run, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("-D", "help", "helloworld"))
	res.Assert(t, icmd.Success)
	golden.Assert(t, res.Stdout(), "docker-help-helloworld.golden")
	assert.Assert(t, is.Equal(res.Stderr(), ""))
}

// TestGoodHelp ensures correct behaviour when calling a valid plugin
// with `--help`. A global argument is used to ensure it does not
// interfere.
func TestGoodHelp(t *testing.T) {
	run, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("-D", "helloworld", "--help"))
	res.Assert(t, icmd.Success)
	// This is the same golden file as `TestHelpGood`, above.
	golden.Assert(t, res.Stdout(), "docker-help-helloworld.golden")
	assert.Assert(t, is.Equal(res.Stderr(), ""))
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

// TestHelpGoodSubcommand ensures correct behaviour when invoking help on a
// valid plugin subcommand. A global argument is included to ensure it does not
// interfere.
func TestHelpGoodSubcommand(t *testing.T) {
	run, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("-D", "help", "helloworld", "goodbye"))
	res.Assert(t, icmd.Success)
	golden.Assert(t, res.Stdout(), "docker-help-helloworld-goodbye.golden")
	assert.Assert(t, is.Equal(res.Stderr(), ""))
}

// TestGoodSubcommandHelp ensures correct behaviour when calling a valid plugin
// with a subcommand and `--help`. A global argument is used to ensure it does not
// interfere.
func TestGoodSubcommandHelp(t *testing.T) {
	run, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("-D", "helloworld", "goodbye", "--help"))
	res.Assert(t, icmd.Success)
	// This is the same golden file as `TestHelpGoodSubcommand`, above.
	golden.Assert(t, res.Stdout(), "docker-help-helloworld-goodbye.golden")
	assert.Assert(t, is.Equal(res.Stderr(), ""))
}
