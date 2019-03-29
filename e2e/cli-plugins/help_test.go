package cliplugins

import (
	"bufio"
	"regexp"
	"strings"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/icmd"
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
	scanner := bufio.NewScanner(strings.NewReader(res.Stdout()))

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
	helloworldre := regexp.MustCompile(`^  helloworld\*\s+A basic Hello World plugin for tests \(Docker Inc\., testing\)$`)
	badmetare := regexp.MustCompile(`^  badmeta\s+invalid metadata: invalid character 'i' looking for beginning of object key string$`)
	var helloworldcount, badmetacount int
	for _, expected := range []*regexp.Regexp{
		regexp.MustCompile(`^A self-sufficient runtime for containers$`),
		regexp.MustCompile(`^Management Commands:$`),
		regexp.MustCompile(`^  container\s+Manage containers$`),
		helloworldre,
		regexp.MustCompile(`^  image\s+Manage images$`),
		regexp.MustCompile(`^Commands:$`),
		regexp.MustCompile(`^  create\s+Create a new container$`),
		regexp.MustCompile(`^Invalid Plugins:$`),
		badmetare,
		nil, // scan to end of input rather than stopping at badmetare
	} {
		var found bool
		for scanner.Scan() {
			text := scanner.Text()
			if helloworldre.MatchString(text) {
				helloworldcount++
			}
			if badmetare.MatchString(text) {
				badmetacount++
			}

			if expected != nil && expected.MatchString(text) {
				found = true
				break
			}
		}
		assert.Assert(t, expected == nil || found, "Did not find match for %q in `docker help` output", expected)
	}
	// We successfully scanned all the input
	assert.Assert(t, !scanner.Scan())
	assert.NilError(t, scanner.Err())
	// Plugins should only be listed once.
	assert.Assert(t, is.Equal(helloworldcount, 1))
	assert.Assert(t, is.Equal(badmetacount, 1))

	// Running with `--help` should produce the same.
	res2 := icmd.RunCmd(run("--help"))
	res2.Assert(t, icmd.Expected{
		ExitCode: 0,
	})
	assert.Assert(t, is.Equal(res2.Stdout(), res.Stdout()))
	assert.Assert(t, is.Equal(res2.Stderr(), ""))

	// Running just `docker` (without `help` nor `--help`) should produce the same thing, except on Stderr.
	res2 = icmd.RunCmd(run())
	res2.Assert(t, icmd.Expected{
		ExitCode: 0,
	})
	assert.Assert(t, is.Equal(res2.Stdout(), ""))
	assert.Assert(t, is.Equal(res2.Stderr(), res.Stdout()))
}
