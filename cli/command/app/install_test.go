package app

import (
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/opts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gotest.tools/v3/assert"
)

func TestNewInstallCommandInvalidArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "empty args - no url",
			args:     []string{},
			expected: "requires at least 1 argument",
		},
		{
			name:     "some args - no url",
			args:     []string{"-q", "--build-arg", "var=val", "--env", "var=val"},
			expected: "requires at least 1 argument",
		},
		{
			name:     "unsupported flag",
			args:     []string{"--random-option", "url"},
			expected: "unknown flag: --random-option",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{})
			cmd := NewInstallCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			assert.ErrorContains(t, err, tc.expected)
		})
	}
}

func TestNewInstallCommandAddInstallFlags(t *testing.T) {
	assertChange := func(cmd *cobra.Command, n string, v any) {
		f := cmd.Flag(n)
		assert.Equal(t, f.Changed, true)
		if f.Value.Type() == "list" {
			s := f.Value.(*opts.ListOpts).GetAll()
			assert.DeepEqual(t, s, v)
		} else {
			assert.Equal(t, f.Value.String(), v)
		}
	}
	assertNoChange := func(cmd *cobra.Command, n string, v any) {
		f := cmd.Flag(n)
		assert.Equal(t, f.Changed, false)
		if f.Value.Type() == "list" {
			s := f.Value.(*opts.ListOpts).GetAll()
			assert.DeepEqual(t, s, v)
		} else {
			assert.Equal(t, f.Value.String(), v)
		}
	}
	assertNoChangeNotEmpty := func(cmd *cobra.Command, n string) {
		f := cmd.Flag(n)
		assert.Equal(t, f.Changed, false)
		assert.Assert(t, f.Value.String() != "")
	}

	tests := []struct {
		name string
		args []string
		runE func(*cobra.Command, []string) error
	}{
		{
			name: "all install options",
			args: []string{"--cidfile", "cid.txt", "--destination", "/a/b/c", "--detach", "--egress", "/d/e/f", "--iidfile", "id.txt", "--launch", "url"},
			runE: func(cmd *cobra.Command, args []string) error {
				assert.Assert(t, len(args) == 1)
				assert.Equal(t, args[0], "url")

				assertChange(cmd, "cidfile", "cid.txt")
				assertChange(cmd, "destination", "/a/b/c")
				assertChange(cmd, "detach", "true")
				assertChange(cmd, "egress", "/d/e/f")
				assertChange(cmd, "iidfile", "id.txt")
				assertChange(cmd, "launch", "true")
				return nil
			},
		},
		{
			name: "default install options",
			args: []string{"url"},
			runE: func(cmd *cobra.Command, args []string) error {
				assert.Assert(t, len(args) == 1)
				assert.Equal(t, args[0], "url")

				assertNoChangeNotEmpty(cmd, "cidfile")
				assertNoChange(cmd, "destination", defaultAppBase())
				assertNoChange(cmd, "detach", "false")
				assertNoChange(cmd, "egress", "/egress")
				assertNoChangeNotEmpty(cmd, "iidfile")
				assertNoChange(cmd, "launch", "false")
				return nil
			},
		},
		{
			name: "common build options",
			args: []string{"--build-arg", "var=val", "--file", "docker.file", "--platform", "target_os/arch", "--pull=false", "--tag", "name:tag", "url"},
			runE: func(cmd *cobra.Command, args []string) error {
				assert.Equal(t, len(args), 1)
				assert.Equal(t, args[0], "url")

				// assertChange(cmd, "build-arg", []string{"var=val"})
				assertChange(cmd, "file", "docker.file")
				assertChange(cmd, "platform", "target_os/arch")
				assertChange(cmd, "pull", "false")
				assertChange(cmd, "tag", []string{"name:tag"})
				return nil
			},
		},
		{
			name: "supported run options",
			args: []string{"--entrypoint", "/entry.sh", "--env", "var=val", "--env-file", ".env", "--privileged", "--volume", "$HOME:/home", "--workdir", "/tmp", "url"},
			runE: func(cmd *cobra.Command, args []string) error {
				assert.Equal(t, len(args), 1)
				assert.Equal(t, args[0], "url")

				assertChange(cmd, "entrypoint", "/entry.sh")
				assertChange(cmd, "env", []string{"var=val"})
				assertChange(cmd, "env-file", []string{".env"})
				assertChange(cmd, "privileged", "true")
				assertChange(cmd, "volume", []string{"$HOME:/home"})
				assertChange(cmd, "workdir", "/tmp")
				return nil
			},
		},
		{
			name: "container run command args",
			args: []string{"url", "cmd", "--arg1", "--arg2"},
			runE: func(cmd *cobra.Command, args []string) error {
				assert.Equal(t, len(args), 4)
				assert.Equal(t, args[0], "url")

				assert.Equal(t, args[1], "cmd")
				assert.Equal(t, args[2], "--arg1")
				assert.Equal(t, args[3], "--arg2")
				return nil
			},
		},
		{
			name: "supported cp options",
			args: []string{"--archive", "--follow-link", "url"},
			runE: func(cmd *cobra.Command, args []string) error {
				assert.Equal(t, len(args), 1)
				assert.Equal(t, args[0], "url")

				assertChange(cmd, "archive", "true")
				assertChange(cmd, "follow-link", "true")
				return nil
			},
		},
		{
			name: "launch args after option terminator --",
			args: []string{"url", "--", "launch-sub-cmd", "--arg1", "--arg2"},
			runE: func(cmd *cobra.Command, args []string) error {
				assert.Equal(t, len(args), 5)
				assert.Equal(t, args[0], "url")

				assert.Equal(t, args[1], "--")

				assert.Equal(t, args[2], "launch-sub-cmd")
				assert.Equal(t, args[3], "--arg1")
				assert.Equal(t, args[4], "--arg2")
				return nil
			},
		},
		{
			name: "split run/launch args by option terminator --",
			args: []string{"--entrypoint", "/entry.sh", "url", "cmd", "arg1", "--", "launch-sub-cmd", "--arg2"},
			runE: func(cmd *cobra.Command, args []string) error {
				assert.Equal(t, len(args), 6)
				assert.Equal(t, args[0], "url")

				options := AppOptions{}
				options.setArgs(args)
				assertChange(cmd, "entrypoint", "/entry.sh")
				assert.Equal(t, options.buildContext(), "url")
				assert.DeepEqual(t, options.runArgs(), []string{"cmd", "arg1"})
				assert.DeepEqual(t, options.launchArgs(), []string{"launch-sub-cmd", "--arg2"})
				return nil
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{})
			cmd := NewInstallCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetArgs(tc.args)
			cmd.RunE = tc.runE
			cmd.Execute()
		})
	}
}

func TestAddInstallFlagsAppOptions(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		check func(*AppOptions)
	}{
		{
			name: "all install options",
			args: []string{"--destination", "/a/b/c", "--egress", "/d/e/f", "--iidfile", "id.txt", "--cidfile", "cid.txt", "--detach", "--launch", "-"},
			check: func(o *AppOptions) {
				assert.Equal(t, o.destination, "/a/b/c")
				assert.Equal(t, o.egress, "/d/e/f")
				assert.Equal(t, o.imageIDFile, "id.txt")
				assert.Equal(t, o.containerIDFile, "cid.txt")
				assert.Equal(t, o.detach, true)
				assert.Equal(t, o.launch, true)
			},
		},
		{
			name: "default install options",
			args: []string{"-"},
			check: func(o *AppOptions) {
				assert.Equal(t, o.destination, defaultAppBase())
				assert.Equal(t, o.egress, "/egress")
				assert.Assert(t, o.imageIDFile != "")
				assert.Assert(t, o.containerIDFile != "")
				assert.Equal(t, o.detach, false)
				assert.Equal(t, o.launch, false)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			flags := &pflag.FlagSet{}

			options := addInstallFlags(flags, defaultAppBase(), false)

			flags.Parse(tc.args)

			assert.Assert(t, options != nil)
			assert.Assert(t, options.buildOpts != nil)
			assert.Assert(t, options.runOpts != nil)
			assert.Assert(t, options.containerOpts != nil)
			assert.Assert(t, options.copyOpts != nil)
			assert.Assert(t, options._appBase == defaultAppBase())

			tc.check(options)
		})
	}
}
