package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/commands"
	"github.com/docker/cli/cli/command/commands/lazychecks"
	"github.com/docker/cli/cli/debug"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestClientDebugEnabled(t *testing.T) {
	defer debug.Disable()

	tcmd := newDockerCommand(&command.DockerCli{})
	tcmd.SetFlag("debug", "true")
	cmd, _, err := tcmd.HandleGlobalFlags()
	assert.NilError(t, err)
	assert.NilError(t, tcmd.Initialize())
	err = cmd.PersistentPreRunE(cmd, []string{})
	assert.NilError(t, err)
	assert.Check(t, is.Equal("1", os.Getenv("DEBUG")))
	assert.Check(t, is.Equal(logrus.DebugLevel, logrus.GetLevel()))
}

var discard = ioutil.NopCloser(bytes.NewBuffer(nil))

func runCliCommand(t *testing.T, r io.ReadCloser, w io.Writer, args ...string) error {
	t.Helper()
	if r == nil {
		r = discard
	}
	if w == nil {
		w = ioutil.Discard
	}
	cli, err := command.NewDockerCli(command.WithInputStream(r), command.WithCombinedStreams(w))
	assert.NilError(t, err)
	tcmd := newDockerCommand(cli)
	tcmd.SetArgs(args)
	cmd, _, err := tcmd.HandleGlobalFlags()
	assert.NilError(t, err)
	assert.NilError(t, tcmd.Initialize())
	return cmd.Execute()
}

func TestExitStatusForInvalidSubcommandWithHelpFlag(t *testing.T) {
	err := runCliCommand(t, nil, nil, "help", "invalid")
	assert.Error(t, err, "unknown help topic: invalid")
}

func TestExitStatusForInvalidSubcommand(t *testing.T) {
	err := runCliCommand(t, nil, nil, "invalid")
	assert.Check(t, is.ErrorContains(err, "docker: 'invalid' is not a docker command."))
}

func TestVersion(t *testing.T) {
	var b bytes.Buffer
	err := runCliCommand(t, nil, &b, "--version")
	assert.NilError(t, err)
	assert.Check(t, is.Contains(b.String(), "Docker version"))
}

type kv struct {
	key   string
	value string
}

type annotatedFlag struct {
	flag        string
	set         bool
	annotations []kv
}

func makeAnnotatedFlag(flag string, set bool, annotations ...kv) annotatedFlag {
	return annotatedFlag{flag: flag, set: set, annotations: annotations}
}

func commandWithFlags(flags ...annotatedFlag) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	for _, f := range flags {
		cmd.Flags().String(f.flag, "", "")
		for _, a := range f.annotations {
			cmd.Flags().SetAnnotation(f.flag, a.key, []string{a.value})
		}
		if f.set {
			cmd.Flags().Set(f.flag, "value")
		}
	}
	return cmd
}

func commandWithParentAndFlags(parent *cobra.Command, flags ...annotatedFlag) *cobra.Command {
	child := commandWithFlags(flags...)
	parent.AddCommand(child)
	return child
}

func commandWithAnnotations(annotations ...kv) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "test",
		Annotations: make(map[string]string),
	}
	for _, kv := range annotations {
		cmd.Annotations[kv.key] = kv.value
	}
	return cmd
}

func commandWithParent(parent *cobra.Command, annotations ...kv) *cobra.Command {
	cmd := commandWithAnnotations(annotations...)
	parent.AddCommand(cmd)
	return cmd
}

type lazyCheckFlag struct {
	flag  string
	err   error
	isSet bool
}

func commandWithLazyCheckFlag(f lazyCheckFlag) *cobra.Command {
	cmd := &cobra.Command{
		Use: "test",
	}
	cmd.Flags().String(f.flag, "", "")
	lazychecks.AddLazyFlagCheck(cmd.Flags(), f.flag, func(_ command.ClientInfo, _ command.ServerInfo, _ string) error {
		return f.err
	})
	if f.isSet {
		cmd.Flags().Set(f.flag, "value")
	}
	return cmd
}

func TestIsLocalOnly(t *testing.T) {
	cases := []struct {
		name     string
		cmd      *cobra.Command
		expected bool
	}{
		{
			name:     "not-local",
			cmd:      commandWithParent(&cobra.Command{}),
			expected: false,
		},
		{
			name:     "local",
			cmd:      commandWithParent(&cobra.Command{}, kv{key: "local-only"}),
			expected: true,
		},
		{
			name:     "local-parent",
			cmd:      commandWithParent(commandWithAnnotations(kv{key: "local-only"})),
			expected: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, isLocalOnly(c.cmd))
		})
	}
}

func TestCheckCommandRecursively(t *testing.T) {
	check := commandAnnotationCheck("error", func(v string) error {
		return errors.New(v)
	})
	cases := []struct {
		name          string
		cmd           *cobra.Command
		expectedError string
	}{
		{
			name:          "no-error",
			cmd:           commandWithParent(&cobra.Command{}),
			expectedError: "",
		},
		{
			name:          "child-error",
			cmd:           commandWithParent(&cobra.Command{}, kv{key: "error", value: "boom"}),
			expectedError: "boom",
		},
		{
			name:          "parrent-error",
			cmd:           commandWithParent(commandWithAnnotations(kv{key: "error", value: "boom"})),
			expectedError: "boom",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := checkCommandRecursively(c.cmd, []commandCheck{check})
			if c.expectedError == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, c.expectedError)
			}
		})
	}
}

func TestCheckFlags(t *testing.T) {
	check := flagAnnotationCheck("error", func(f *pflag.Flag) error {
		return fmt.Errorf("flag %s has error %s", f.Name, getFlagAnnotation(f, "error"))
	})
	cases := []struct {
		name          string
		cmd           *cobra.Command
		expectedError string
	}{
		{
			name:          "no-error",
			cmd:           commandWithFlags(),
			expectedError: "",
		},
		{
			name:          "1-error",
			cmd:           commandWithFlags(makeAnnotatedFlag("flag1", true, kv{key: "error", value: "boom"})),
			expectedError: "flag flag1 has error boom",
		},
		{
			name: "2-errors",
			cmd: commandWithFlags(makeAnnotatedFlag("flag1", true, kv{key: "error", value: "boom"}),
				makeAnnotatedFlag("flag2", true, kv{key: "error", value: "boom"})),
			expectedError: "flag flag1 has error boom\nflag flag2 has error boom",
		},
		{
			name:          "0-error",
			cmd:           commandWithFlags(makeAnnotatedFlag("flag1", false, kv{key: "error", value: "boom"})),
			expectedError: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := checkFlags(c.cmd, []flagCheck{check})
			if c.expectedError == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, c.expectedError)
			}
		})
	}
}

type testVersionDetails struct {
	clientVersion      string
	osType             string
	clientExperimental bool
	serverExperimental bool
}

func (d *testVersionDetails) Client() client.APIClient {
	client, _ := client.NewClientWithOpts(client.WithVersion(d.clientVersion))
	return client
}

func (d *testVersionDetails) ClientInfo() command.ClientInfo {
	return command.ClientInfo{
		HasExperimental: d.clientExperimental,
		DefaultVersion:  d.clientVersion,
	}
}

func (d *testVersionDetails) ServerInfo() command.ServerInfo {
	return command.ServerInfo{
		OSType:          d.osType,
		HasExperimental: d.serverExperimental,
	}
}

type noVersionDetails struct {
	t               *testing.T
	experimentalCLI bool
}

func (d *noVersionDetails) Client() client.APIClient {
	d.t.Fatal("should not have called Client")
	return nil
}
func (d *noVersionDetails) ClientInfo() command.ClientInfo {
	return command.ClientInfo{
		HasExperimental: d.experimentalCLI,
	}
}
func (d *noVersionDetails) ServerInfo() command.ServerInfo {
	d.t.Fatal("should not have called Client")
	return command.ServerInfo{}
}

func subCommand(t *testing.T, cmd *cobra.Command, path ...string) *cobra.Command {
	t.Helper()
	if len(path) == 0 {
		return cmd
	}
	for _, c := range cmd.Commands() {
		if c.Name() == path[0] {
			return subCommand(t, c, path[1:]...)
		}
	}
	t.Fatal("cant find requested command")
	return nil
}

func testDockerCmd() *cobra.Command {
	dockerCli, _ := command.NewDockerCli()
	cmd := &cobra.Command{
		Use: "docker",
	}
	commands.AddCommands(cmd, dockerCli)
	return cmd
}

func TestIsSupportedWithLocalOnly(t *testing.T) {
	cases := []struct {
		name                       string
		cmd                        *cobra.Command
		expectedErrExperimental    string
		expectedErrNotExperimental string
	}{
		{
			name: "context-ls",
			cmd:  subCommand(t, testDockerCmd(), "context", "ls"),
		},
		{
			name:                       "experimental-cmd",
			cmd:                        commandWithParent(subCommand(t, testDockerCmd(), "context"), kv{key: "experimentalCLI"}),
			expectedErrNotExperimental: "docker context test is only supported on a Docker cli with experimental cli features enabled",
		},
		{
			name:                       "experimental-flag",
			cmd:                        commandWithParentAndFlags(subCommand(t, testDockerCmd(), "context"), makeAnnotatedFlag("test-flag", true, kv{key: "experimentalCLI"})),
			expectedErrNotExperimental: `"--test-flag" is only supported on a Docker cli with experimental cli features enabled`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			details := &noVersionDetails{
				experimentalCLI: true,
				t:               t,
			}
			err := isSupported(c.cmd, details)
			if c.expectedErrExperimental == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, c.expectedErrExperimental)
			}
			details = &noVersionDetails{
				t: t,
			}
			err = isSupported(c.cmd, details)
			if c.expectedErrNotExperimental == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, c.expectedErrNotExperimental)
			}
		})
	}
}

func withFlagValue(cmd *cobra.Command, flagName string, value string) *cobra.Command {
	cmd.Flags().Set(flagName, value)
	return cmd
}

func TestIsSupported(t *testing.T) {
	cases := []struct {
		name          string
		cmd           *cobra.Command
		details       versionDetails
		expectedError string
	}{
		{
			name: "cmd-version-fail",
			cmd:  subCommand(t, testDockerCmd(), "builder"),
			details: &testVersionDetails{
				clientVersion: "1.30",
			},
			expectedError: "docker builder requires API version 1.31, but the Docker daemon API version is 1.30",
		},
		{
			name: "cmd-version-success",
			cmd:  subCommand(t, testDockerCmd(), "builder"),
			details: &testVersionDetails{
				clientVersion: "1.31",
			},
			expectedError: "",
		},
		{
			name: "cmd-ostype-fail",
			cmd:  commandWithAnnotations(kv{key: "ostype", value: "test-os"}),
			details: &testVersionDetails{
				osType: "wrong-os",
			},
			expectedError: "test is only supported on a Docker daemon running on test-os, but the Docker daemon is running on wrong-os",
		},
		{
			name: "cmd-ostype-success",
			cmd:  commandWithAnnotations(kv{key: "ostype", value: "test-os"}),
			details: &testVersionDetails{
				osType: "test-os",
			},
			expectedError: "",
		},
		{
			name: "cmd-experimental-fail",
			cmd:  subCommand(t, testDockerCmd(), "checkpoint"),
			details: &testVersionDetails{
				osType:        "linux",
				clientVersion: "1.25",
			},
			expectedError: "docker checkpoint is only supported on a Docker daemon with experimental features enabled",
		},
		{
			name: "cmd-experimental-success",
			cmd:  subCommand(t, testDockerCmd(), "checkpoint"),
			details: &testVersionDetails{
				osType:             "linux",
				clientVersion:      "1.25",
				serverExperimental: true,
			},
			expectedError: "",
		},

		{
			name: "flag-version-fail",
			cmd:  withFlagValue(subCommand(t, testDockerCmd(), "config", "create"), "template-driver", "test"),
			details: &testVersionDetails{
				clientVersion: "1.36",
			},
			expectedError: `"--template-driver" requires API version 1.37, but the Docker daemon API version is 1.36`,
		},
		{
			name: "flag-version-success",
			cmd:  withFlagValue(subCommand(t, testDockerCmd(), "config", "create"), "template-driver", "test"),
			details: &testVersionDetails{
				clientVersion: "1.37",
			},
			expectedError: "",
		},
		{
			name: "flag-ostype-fail",
			cmd:  withFlagValue(subCommand(t, testDockerCmd(), "container", "create"), "cpu-count", "2"),
			details: &testVersionDetails{
				osType: "wrong-os",
			},
			expectedError: `"--cpu-count" is only supported on a Docker daemon running on windows, but the Docker daemon is running on wrong-os`,
		},
		{
			name: "flag-ostype-success",
			cmd:  withFlagValue(subCommand(t, testDockerCmd(), "container", "create"), "cpu-count", "2"),
			details: &testVersionDetails{
				osType: "windows",
			},
			expectedError: "",
		},
		{
			name: "flag-experimental-fail",
			cmd:  withFlagValue(subCommand(t, testDockerCmd(), "image", "build"), "squash", "true"),
			details: &testVersionDetails{
				clientVersion: "1.25",
			},
			expectedError: `"--squash" is only supported on a Docker daemon with experimental features enabled`,
		},
		{
			name: "flag-experimental-success",
			cmd:  withFlagValue(subCommand(t, testDockerCmd(), "image", "build"), "squash", "true"),
			details: &testVersionDetails{
				clientVersion:      "1.25",
				serverExperimental: true,
			},
			expectedError: "",
		},
		{
			name: "lazy-check-unset",
			cmd:  commandWithLazyCheckFlag(lazyCheckFlag{flag: "flag1", isSet: false, err: errors.New("boom")}),
			details: &testVersionDetails{
				clientVersion:      "1.25",
				serverExperimental: true,
			},
			expectedError: "",
		},
		{
			name: "lazy-check-set",
			cmd:  commandWithLazyCheckFlag(lazyCheckFlag{flag: "flag1", isSet: true, err: errors.New("boom")}),
			details: &testVersionDetails{
				clientVersion:      "1.25",
				serverExperimental: true,
			},
			expectedError: "boom",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := isSupported(c.cmd, c.details)
			if c.expectedError == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, c.expectedError)
			}
		})
	}
}
