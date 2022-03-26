package cliplugins

import (
	"regexp"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/icmd"
)

// TestGlobalHelp ensures correct behaviour when running `docker help`
func TestGlobalHelp(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	res := icmd.RunCmd(run("help"))
	res.Assert(t, icmd.Expected{
		ExitCode: 0,
	})
	assert.Assert(t, is.Equal(res.Stderr(), ""))
	output := res.Stdout()

	// Instead of baking in the full current output of `docker
	// help`, which can be expected to change regularly, bake in
	// some checkpoints. Key things we are looking for:
	//
	//  - The top-level description
	//  - Each of the main headings
	//  - Some builtin commands under the main headings
	//  - The `helloworld` plugin in the appropriate place
	//  - The `badmeta` plugin under the "Invalid Plugins" heading.
	//
	// Regexps are needed because the width depends on `unix.TIOCGWINSZ` or similar.
	for _, s := range []string{
		`Management Commands:`,
		`\s+container\s+Manage containers`,
		`\s+helloworld\*\s+A basic Hello World plugin for tests \(Docker Inc\., testing\)`,
		`\s+image\s+Manage images`,
		`Commands:`,
		`\s+create\s+Create a new container`,
		`Invalid Plugins:`,
		`\s+badmeta\s+invalid metadata: invalid character 'i' looking for beginning of object key string`,
	} {
		expected := regexp.MustCompile(`(?m)^` + s + `$`)
		matches := expected.FindAllString(output, -1)
		assert.Equal(t, len(matches), 1, "Did not find expected number of matches for %q in `docker help` output", expected)
	}

	// Running with `--help` should produce the same.
	t.Run("help_flag", func(t *testing.T) {
		res2 := icmd.RunCmd(run("--help"))
		res2.Assert(t, icmd.Expected{
			ExitCode: 0,
		})
		assert.Assert(t, is.Equal(res2.Stdout(), output))
		assert.Assert(t, is.Equal(res2.Stderr(), ""))
	})

	// Running just `docker` (without `help` nor `--help`) should produce the same thing, except on Stderr.
	t.Run("bare", func(t *testing.T) {
		res2 := icmd.RunCmd(run())
		res2.Assert(t, icmd.Expected{
			ExitCode: 0,
		})
		assert.Assert(t, is.Equal(res2.Stdout(), ""))
		assert.Assert(t, is.Equal(res2.Stderr(), output))
	})

	t.Run("badopt", func(t *testing.T) {
		// Running `docker --badopt` should also produce the
		// same thing, give or take the leading error message
		// and a trailing carriage return (due to main() using
		// Println in the error case).
		res2 := icmd.RunCmd(run("--badopt"))
		res2.Assert(t, icmd.Expected{
			ExitCode: 125,
		})
		assert.Assert(t, is.Equal(res2.Stdout(), ""))
		assert.Assert(t, is.Contains(res2.Stderr(), "unknown flag: --badopt"))
		assert.Assert(t, is.Contains(res2.Stderr(), "See 'docker --help"))
	})
}
