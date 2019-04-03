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
			expectedErr: icmd.None,
		},
		{
			name:        "short-with-val",
			args:        []string{"--context", "Christmas"},
			expectedOut: "Merry Christmas!",
			expectedErr: icmd.None,
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

// TestCliPluginsVersion checks that `-v` and friends DTRT
func TestCliPluginsVersion(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	for _, tc := range []struct {
		name           string
		args           []string
		expCode        int
		expOut, expErr string
	}{
		{
			name:    "global-version",
			args:    []string{"version"},
			expCode: 0,
			expOut:  "Client:\n Version:",
			expErr:  icmd.None,
		},
		{
			name:    "global-version-flag",
			args:    []string{"--version"},
			expCode: 0,
			expOut:  "Docker version",
			expErr:  icmd.None,
		},
		{
			name:    "global-short-version-flag",
			args:    []string{"-v"},
			expCode: 0,
			expOut:  "Docker version",
			expErr:  icmd.None,
		},
		{
			name:    "global-with-unknown-arg",
			args:    []string{"version", "foo"},
			expCode: 1,
			expOut:  icmd.None,
			expErr:  `"docker version" accepts no arguments.`,
		},
		{
			name:    "global-with-plugin-arg",
			args:    []string{"version", "helloworld"},
			expCode: 1,
			expOut:  icmd.None,
			expErr:  `"docker version" accepts no arguments.`,
		},
		{
			name:    "global-version-flag-with-unknown-arg",
			args:    []string{"--version", "foo"},
			expCode: 0,
			expOut:  "Docker version",
			expErr:  icmd.None,
		},
		{
			name:    "global-short-version-flag-with-unknown-arg",
			args:    []string{"-v", "foo"},
			expCode: 0,
			expOut:  "Docker version",
			expErr:  icmd.None,
		},
		{
			name:    "global-version-flag-with-plugin",
			args:    []string{"--version", "helloworld"},
			expCode: 125,
			expOut:  icmd.None,
			expErr:  "unknown flag: --version",
		},
		{
			name:    "global-short-version-flag-with-plugin",
			args:    []string{"-v", "helloworld"},
			expCode: 125,
			expOut:  icmd.None,
			expErr:  "unknown shorthand flag: 'v' in -v",
		},
		{
			name:    "plugin-with-version",
			args:    []string{"helloworld", "version"},
			expCode: 0,
			expOut:  "Hello World!",
			expErr:  icmd.None,
		},
		{
			name:    "plugin-with-version-flag",
			args:    []string{"helloworld", "--version"},
			expCode: 125,
			expOut:  icmd.None,
			expErr:  "unknown flag: --version",
		},
		{
			name:    "plugin-with-short-version-flag",
			args:    []string{"helloworld", "-v"},
			expCode: 125,
			expOut:  icmd.None,
			expErr:  "unknown shorthand flag: 'v' in -v",
		},
		{
			name:    "",
			args:    []string{},
			expCode: 0,
			expOut:  "",
			expErr:  "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := icmd.RunCmd(run(tc.args...))
			res.Assert(t, icmd.Expected{
				ExitCode: tc.expCode,
				Out:      tc.expOut,
				Err:      tc.expErr,
			})
		})
	}

}
