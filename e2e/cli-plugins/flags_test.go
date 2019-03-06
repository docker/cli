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

// TestClashWithGlobalArgs ensures correct behaviour when a plugin
// has an argument with the same name as one of the globals.
func TestClashWithGlobalArgs(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	for _, tc := range []struct {
		name                     string
		args                     []string
		expectedOut, expectedErr string
	}{
		{
			name:        "short-without-val",
			args:        []string{"-D"},
			expectedOut: "Hello World!",
			expectedErr: "Plugin debug mode enabled",
		},
		{
			name:        "long-without-val",
			args:        []string{"--debug"},
			expectedOut: "Hello World!",
			expectedErr: "Plugin debug mode enabled",
		},
		{
			name:        "short-with-val",
			args:        []string{"-c", "Christmas"},
			expectedOut: "Merry Christmas!",
			expectedErr: "",
		},
		{
			name:        "short-with-val",
			args:        []string{"--context", "Christmas"},
			expectedOut: "Merry Christmas!",
			expectedErr: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"helloworld"}, tc.args...)
			res := icmd.RunCmd(run(args...))
			res.Assert(t, icmd.Expected{
				ExitCode: 0,
				Out:      tc.expectedOut,
				Err:      tc.expectedErr,
			})
		})
	}
}

// TestUnknownGlobal checks that unknown globals report errors
func TestUnknownGlobal(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	for name, args := range map[string][]string{
		"no-val":       {"--unknown", "helloworld"},
		"separate-val": {"--unknown", "foo", "helloworld"},
		"joined-val":   {"--unknown=foo", "helloworld"},
	} {
		t.Run(name, func(t *testing.T) {
			res := icmd.RunCmd(run(args...))
			res.Assert(t, icmd.Expected{
				ExitCode: 125,
				Out:      icmd.None,
				Err:      "unknown flag: --unknown",
			})
		})
	}
}
