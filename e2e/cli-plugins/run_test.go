package cliplugins

import (
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/golden"
	"gotest.tools/icmd"
)

const shortHFlagDeprecated = "Flag shorthand -h has been deprecated, please use --help\n"

// TestRunNonexisting ensures correct behaviour when running a nonexistent plugin.
func TestRunNonexisting(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("nonexistent"))
	res.Assert(t, icmd.Expected{
		ExitCode: 1,
		Out:      icmd.None,
	})
	golden.Assert(t, res.Stderr(), "docker-nonexistent-err.golden")
}

// TestHelpNonexisting ensures correct behaviour when invoking help on a nonexistent plugin.
func TestHelpNonexisting(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("help", "nonexistent"))
	res.Assert(t, icmd.Expected{
		ExitCode: 1,
		Out:      icmd.None,
	})
	golden.Assert(t, res.Stderr(), "docker-help-nonexistent-err.golden")
}

// TestNonexistingHelp ensures correct behaviour when invoking a
// nonexistent plugin with `--help`.
func TestNonexistingHelp(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("nonexistent", "--help"))
	res.Assert(t, icmd.Expected{
		ExitCode: 0,
		// This should actually be the whole docker help
		// output, so spot check instead having of a golden
		// with everything in, which will change too frequently.
		Out: "Usage:	docker [OPTIONS] COMMAND\n\nA self-sufficient runtime for containers",
		Err: icmd.None,
	})
	// Short -h should be the same, modulo the deprecation message
	exp := shortHFlagDeprecated + res.Stdout()
	res = icmd.RunCmd(run("nonexistent", "-h"))
	res.Assert(t, icmd.Expected{
		ExitCode: 0,
		// This should be identical to the --help case above
		Out: exp,
		Err: icmd.None,
	})
}

// TestRunBad ensures correct behaviour when running an existent but invalid plugin
func TestRunBad(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("badmeta"))
	res.Assert(t, icmd.Expected{
		ExitCode: 1,
		Out:      icmd.None,
	})
	golden.Assert(t, res.Stderr(), "docker-badmeta-err.golden")
}

// TestHelpBad ensures correct behaviour when invoking help on a existent but invalid plugin.
func TestHelpBad(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("help", "badmeta"))
	res.Assert(t, icmd.Expected{
		ExitCode: 1,
		Out:      icmd.None,
	})
	golden.Assert(t, res.Stderr(), "docker-help-badmeta-err.golden")
}

// TestBadHelp ensures correct behaviour when invoking an
// existent but invalid plugin with `--help`.
func TestBadHelp(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("badmeta", "--help"))
	res.Assert(t, icmd.Expected{
		ExitCode: 0,
		// This should be literally the whole docker help
		// output, so spot check instead of a golden with
		// everything in which will change all the time.
		Out: "Usage:	docker [OPTIONS] COMMAND\n\nA self-sufficient runtime for containers",
		Err: icmd.None,
	})
	// Short -h should be the same, modulo the deprecation message
	exp := shortHFlagDeprecated + res.Stdout()
	res = icmd.RunCmd(run("badmeta", "-h"))
	res.Assert(t, icmd.Expected{
		ExitCode: 0,
		// This should be identical to the --help case above
		Out: exp,
		Err: icmd.None,
	})
}

// TestRunGood ensures correct behaviour when running a valid plugin
func TestRunGood(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("helloworld"))
	res.Assert(t, icmd.Expected{
		ExitCode: 0,
		Out:      "Hello World!",
		Err:      icmd.None,
	})
}

// TestHelpGood ensures correct behaviour when invoking help on a
// valid plugin. A global argument is included to ensure it does not
// interfere.
func TestHelpGood(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("-l", "info", "help", "helloworld"))
	res.Assert(t, icmd.Expected{
		ExitCode: 0,
		Err:      icmd.None,
	})
	golden.Assert(t, res.Stdout(), "docker-help-helloworld.golden")
}

// TestGoodHelp ensures correct behaviour when calling a valid plugin
// with `--help`. A global argument is used to ensure it does not
// interfere.
func TestGoodHelp(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("-l", "info", "helloworld", "--help"))
	res.Assert(t, icmd.Expected{
		ExitCode: 0,
		Err:      icmd.None,
	})
	// This is the same golden file as `TestHelpGood`, above.
	golden.Assert(t, res.Stdout(), "docker-help-helloworld.golden")
	// Short -h should be the same, modulo the deprecation message
	exp := shortHFlagDeprecated + res.Stdout()
	res = icmd.RunCmd(run("-l", "info", "helloworld", "-h"))
	res.Assert(t, icmd.Expected{
		ExitCode: 0,
		// This should be identical to the --help case above
		Out: exp,
		Err: icmd.None,
	})
}

// TestRunGoodSubcommand ensures correct behaviour when running a valid plugin with a subcommand
func TestRunGoodSubcommand(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("helloworld", "goodbye"))
	res.Assert(t, icmd.Expected{
		ExitCode: 0,
		Out:      "Goodbye World!",
		Err:      icmd.None,
	})
}

// TestHelpGoodSubcommand ensures correct behaviour when invoking help on a
// valid plugin subcommand. A global argument is included to ensure it does not
// interfere.
func TestHelpGoodSubcommand(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("-l", "info", "help", "helloworld", "goodbye"))
	res.Assert(t, icmd.Expected{
		ExitCode: 0,
		Err:      icmd.None,
	})
	golden.Assert(t, res.Stdout(), "docker-help-helloworld-goodbye.golden")
}

// TestGoodSubcommandHelp ensures correct behaviour when calling a valid plugin
// with a subcommand and `--help`. A global argument is used to ensure it does not
// interfere.
func TestGoodSubcommandHelp(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("-l", "info", "helloworld", "goodbye", "--help"))
	res.Assert(t, icmd.Expected{
		ExitCode: 0,
		Err:      icmd.None,
	})
	// This is the same golden file as `TestHelpGoodSubcommand`, above.
	golden.Assert(t, res.Stdout(), "docker-help-helloworld-goodbye.golden")
}

// TestCliInitialized tests the code paths which ensure that the Cli
// object is initialized even if the plugin uses PersistentRunE
func TestCliInitialized(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("helloworld", "--pre-run", "apiversion"))
	res.Assert(t, icmd.Success)
	assert.Assert(t, res.Stdout() != "")
	assert.Assert(t, is.Equal(res.Stderr(), "Plugin PersistentPreRunE called"))
}

// TestPluginErrorCode tests when the plugin return with a given exit status.
// We want to verify that the exit status does not get output to stdout and also that we return the exit code.
func TestPluginErrorCode(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()
	res := icmd.RunCmd(run("helloworld", "exitstatus2"))
	res.Assert(t, icmd.Expected{
		ExitCode: 2,
		Out:      icmd.None,
		Err:      "Exiting with error status 2",
	})
}
