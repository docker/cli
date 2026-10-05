package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/docker/cli/cli/command"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestConfigureCloudResolverIO(t *testing.T) {
	stdin, err := os.CreateTemp(t.TempDir(), "stdin")
	assert.NilError(t, err)
	t.Cleanup(func() { _ = stdin.Close() })
	stderr, err := os.CreateTemp(t.TempDir(), "stderr")
	assert.NilError(t, err)
	t.Cleanup(func() { _ = stderr.Close() })

	for _, tc := range []struct {
		name           string
		wrapStdin      bool
		wrapStderr     bool
		stdinTerminal  bool
		stderrTerminal bool
	}{
		{name: "terminal files", stdinTerminal: true, stderrTerminal: true},
		{name: "wrapped stdin", wrapStdin: true, stdinTerminal: true, stderrTerminal: true},
		{name: "wrapped stderr", wrapStderr: true, stdinTerminal: true, stderrTerminal: true},
		{name: "wrapped terminals", wrapStdin: true, wrapStderr: true, stdinTerminal: true, stderrTerminal: true},
		{name: "redirected stdin", stderrTerminal: true},
		{name: "redirected stderr", stdinTerminal: true},
		{name: "redirected stdin with wrapped stderr", wrapStderr: true, stderrTerminal: true},
		{name: "redirected stderr with wrapped stdin", wrapStdin: true, stdinTerminal: true},
		{name: "nonterminal wrappers", wrapStdin: true, wrapStderr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var in io.ReadCloser = stdin
			var errOut io.Writer = stderr
			if tc.wrapStdin {
				in = io.NopCloser(stdin)
			}
			if tc.wrapStderr {
				errOut = struct{ io.Writer }{stderr}
			}
			dockerCli, err := command.NewDockerCli(command.WithInputStream(in), command.WithErrorStream(errOut))
			assert.NilError(t, err)
			// Simulate terminal detection without requiring a console in the test runner.
			dockerCli.In().SetIsTerminal(tc.stdinTerminal)
			dockerCli.Err().SetIsTerminal(tc.stderrTerminal)

			var stdout bytes.Buffer
			cmd := &exec.Cmd{Stdout: &stdout}
			configureCloudResolverIO(cmd, dockerCli)
			assert.Equal(t, cmd.Stdout, io.Writer(&stdout))
			if !tc.stdinTerminal || !tc.stderrTerminal || (runtime.GOOS != "windows" && (tc.wrapStdin || tc.wrapStderr)) {
				assert.Assert(t, is.Nil(cmd.Stdin))
				assert.Equal(t, cmd.Stderr, io.Writer(dockerCli.Err()))
				return
			}

			wantStdin, wantStderr := stdin, stderr
			if tc.wrapStdin {
				wantStdin = os.Stdin
			}
			if tc.wrapStderr {
				wantStderr = os.Stderr
			}
			assert.Equal(t, cmd.Stdin, io.Reader(wantStdin))
			assert.Equal(t, cmd.Stderr, io.Writer(wantStderr))
		})
	}
}
