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
	run, cleanup := prepare(t)
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
	//
	// Regexps are needed because the width depends on `unix.TIOCGWINSZ` or similar.
	for _, expected := range []*regexp.Regexp{
		regexp.MustCompile(`^A self-sufficient runtime for containers$`),
		regexp.MustCompile(`^Management Commands:$`),
		regexp.MustCompile(`^  container\s+Manage containers$`),
		regexp.MustCompile(`^Commands:$`),
		regexp.MustCompile(`^  create\s+Create a new container$`),
		regexp.MustCompile(`^  helloworld\s+\(Docker Inc\.\)\s+A basic Hello World plugin for tests$`),
		regexp.MustCompile(`^  ps\s+List containers$`),
	} {
		var found bool
		for scanner.Scan() {
			if expected.MatchString(scanner.Text()) {
				found = true
				break
			}
		}
		assert.Assert(t, found, "Did not find match for %q in `docker help` output", expected)
	}
}
