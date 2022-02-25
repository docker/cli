package container

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/fs"
)

func withDefaultOpts(options ExecOptions) ExecOptions {
	options.Env = opts.NewListOpts(opts.ValidateEnv)
	options.EnvFile = opts.NewListOpts(nil)
	if len(options.Command) == 0 {
		options.Command = []string{"command"}
	}
	return options
}

func TestParseExec(t *testing.T) {
	content := `ONE=1
TWO=2
	`

	tmpFile := fs.NewFile(t, t.Name(), fs.WithContent(content))
	defer tmpFile.Remove()

	testcases := []struct {
		options    ExecOptions
		configFile configfile.ConfigFile
		expected   types.ExecConfig
	}{
		{
			expected: types.ExecConfig{
				Cmd:          []string{"command"},
				AttachStdout: true,
				AttachStderr: true,
			},
			options: withDefaultOpts(ExecOptions{}),
		},
		{
			expected: types.ExecConfig{
				Cmd:          []string{"command1", "command2"},
				AttachStdout: true,
				AttachStderr: true,
			},
			options: withDefaultOpts(ExecOptions{
				Command: []string{"command1", "command2"},
			}),
		},
		{
			options: withDefaultOpts(ExecOptions{
				Interactive: true,
				TTY:         true,
				User:        "uid",
			}),
			expected: types.ExecConfig{
				User:         "uid",
				AttachStdin:  true,
				AttachStdout: true,
				AttachStderr: true,
				Tty:          true,
				Cmd:          []string{"command"},
			},
		},
		{
			options: withDefaultOpts(ExecOptions{Detach: true}),
			expected: types.ExecConfig{
				Detach: true,
				Cmd:    []string{"command"},
			},
		},
		{
			options: withDefaultOpts(ExecOptions{
				TTY:         true,
				Interactive: true,
				Detach:      true,
			}),
			expected: types.ExecConfig{
				Detach: true,
				Tty:    true,
				Cmd:    []string{"command"},
			},
		},
		{
			options:    withDefaultOpts(ExecOptions{Detach: true}),
			configFile: configfile.ConfigFile{DetachKeys: "de"},
			expected: types.ExecConfig{
				Cmd:        []string{"command"},
				DetachKeys: "de",
				Detach:     true,
			},
		},
		{
			options: withDefaultOpts(ExecOptions{
				Detach:     true,
				DetachKeys: "ab",
			}),
			configFile: configfile.ConfigFile{DetachKeys: "de"},
			expected: types.ExecConfig{
				Cmd:        []string{"command"},
				DetachKeys: "ab",
				Detach:     true,
			},
		},
		{
			expected: types.ExecConfig{
				Cmd:          []string{"command"},
				AttachStdout: true,
				AttachStderr: true,
				Env:          []string{"ONE=1", "TWO=2"},
			},
			options: func() ExecOptions {
				o := withDefaultOpts(ExecOptions{})
				o.EnvFile.Set(tmpFile.Path())
				return o
			}(),
		},
		{
			expected: types.ExecConfig{
				Cmd:          []string{"command"},
				AttachStdout: true,
				AttachStderr: true,
				Env:          []string{"ONE=1", "TWO=2", "ONE=override"},
			},
			options: func() ExecOptions {
				o := withDefaultOpts(ExecOptions{})
				o.EnvFile.Set(tmpFile.Path())
				o.Env.Set("ONE=override")
				return o
			}(),
		},
	}

	for _, testcase := range testcases {
		execConfig, err := parseExec(testcase.options, &testcase.configFile)
		assert.NilError(t, err)
		assert.Check(t, is.DeepEqual(testcase.expected, *execConfig))
	}
}

func TestParseExecNoSuchFile(t *testing.T) {
	execOpts := withDefaultOpts(ExecOptions{})
	execOpts.EnvFile.Set("no-such-env-file")
	execConfig, err := parseExec(execOpts, &configfile.ConfigFile{})
	assert.ErrorContains(t, err, "no-such-env-file")
	assert.Check(t, os.IsNotExist(err))
	assert.Check(t, execConfig == nil)
}

func TestRunExec(t *testing.T) {
	var testcases = []struct {
		doc           string
		options       ExecOptions
		client        fakeClient
		expectedError string
		expectedOut   string
		expectedErr   string
	}{
		{
			doc: "successful detach",
			options: withDefaultOpts(ExecOptions{
				Container: "thecontainer",
				Detach:    true,
			}),
			client: fakeClient{execCreateFunc: execCreateWithID},
		},
		{
			doc:     "inspect error",
			options: NewExecOptions(),
			client: fakeClient{
				inspectFunc: func(string) (types.ContainerJSON, error) {
					return types.ContainerJSON{}, errors.New("failed inspect")
				},
			},
			expectedError: "failed inspect",
		},
		{
			doc:           "missing exec ID",
			options:       NewExecOptions(),
			expectedError: "exec ID empty",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.doc, func(t *testing.T) {
			cli := test.NewFakeCli(&testcase.client)

			err := RunExec(cli, testcase.options)
			if testcase.expectedError != "" {
				assert.ErrorContains(t, err, testcase.expectedError)
			} else {
				if !assert.Check(t, err) {
					return
				}
			}
			assert.Check(t, is.Equal(testcase.expectedOut, cli.OutBuffer().String()))
			assert.Check(t, is.Equal(testcase.expectedErr, cli.ErrBuffer().String()))
		})
	}
}

func execCreateWithID(_ string, _ types.ExecConfig) (types.IDResponse, error) {
	return types.IDResponse{ID: "execid"}, nil
}

func TestGetExecExitStatus(t *testing.T) {
	execID := "the exec id"
	expectedErr := errors.New("unexpected error")

	testcases := []struct {
		inspectError  error
		exitCode      int
		expectedError error
	}{
		{
			inspectError: nil,
			exitCode:     0,
		},
		{
			inspectError:  expectedErr,
			expectedError: expectedErr,
		},
		{
			exitCode:      15,
			expectedError: cli.StatusError{StatusCode: 15},
		},
	}

	for _, testcase := range testcases {
		client := &fakeClient{
			execInspectFunc: func(id string) (types.ContainerExecInspect, error) {
				assert.Check(t, is.Equal(execID, id))
				return types.ContainerExecInspect{ExitCode: testcase.exitCode}, testcase.inspectError
			},
		}
		err := getExecExitStatus(context.Background(), client, execID)
		assert.Check(t, is.Equal(testcase.expectedError, err))
	}
}

func TestNewExecCommandErrors(t *testing.T) {
	testCases := []struct {
		name                 string
		args                 []string
		expectedError        string
		containerInspectFunc func(img string) (types.ContainerJSON, error)
	}{
		{
			name:          "client-error",
			args:          []string{"5cb5bb5e4a3b", "-t", "-i", "bash"},
			expectedError: "something went wrong",
			containerInspectFunc: func(containerID string) (types.ContainerJSON, error) {
				return types.ContainerJSON{}, errors.Errorf("something went wrong")
			},
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{inspectFunc: tc.containerInspectFunc})
		cmd := NewExecCommand(cli)
		cmd.SetOut(io.Discard)
		cmd.SetArgs(tc.args)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}
