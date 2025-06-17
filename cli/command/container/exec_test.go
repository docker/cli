package container

import (
	"context"
	"errors"
	"io"
	"os"
	"strconv"
	"testing"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/container"
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
		expected   container.ExecOptions
	}{
		{
			expected: container.ExecOptions{
				Cmd:          []string{"command"},
				AttachStdout: true,
				AttachStderr: true,
			},
			options: withDefaultOpts(ExecOptions{}),
		},
		{
			expected: container.ExecOptions{
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
			expected: container.ExecOptions{
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
			expected: container.ExecOptions{
				Cmd: []string{"command"},
			},
		},
		{
			options: withDefaultOpts(ExecOptions{
				TTY:         true,
				Interactive: true,
				Detach:      true,
			}),
			expected: container.ExecOptions{
				Tty: true,
				Cmd: []string{"command"},
			},
		},
		{
			options:    withDefaultOpts(ExecOptions{Detach: true}),
			configFile: configfile.ConfigFile{DetachKeys: "de"},
			expected: container.ExecOptions{
				Cmd:        []string{"command"},
				DetachKeys: "de",
			},
		},
		{
			options: withDefaultOpts(ExecOptions{
				Detach:     true,
				DetachKeys: "ab",
			}),
			configFile: configfile.ConfigFile{DetachKeys: "de"},
			expected: container.ExecOptions{
				Cmd:        []string{"command"},
				DetachKeys: "ab",
			},
		},
		{
			expected: container.ExecOptions{
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
			expected: container.ExecOptions{
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

	for i, testcase := range testcases {
		t.Run("test "+strconv.Itoa(i+1), func(t *testing.T) {
			execConfig, err := parseExec(testcase.options, &testcase.configFile)
			assert.NilError(t, err)
			assert.Check(t, is.DeepEqual(testcase.expected, *execConfig))
		})
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
	testcases := []struct {
		doc           string
		options       ExecOptions
		client        *fakeClient
		expectedError string
		expectedOut   string
		expectedErr   string
	}{
		{
			doc: "successful detach",
			options: withDefaultOpts(ExecOptions{
				Detach: true,
			}),
			client: &fakeClient{execCreateFunc: execCreateWithID},
		},
		{
			doc:     "inspect error",
			options: NewExecOptions(),
			client: &fakeClient{
				inspectFunc: func(string) (container.InspectResponse, error) {
					return container.InspectResponse{}, errors.New("failed inspect")
				},
			},
			expectedError: "failed inspect",
		},
		{
			doc:           "missing exec ID",
			options:       NewExecOptions(),
			expectedError: "exec ID empty",
			client:        &fakeClient{},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.doc, func(t *testing.T) {
			fakeCLI := test.NewFakeCli(testcase.client)

			err := RunExec(context.TODO(), fakeCLI, "thecontainer", testcase.options)
			if testcase.expectedError != "" {
				assert.ErrorContains(t, err, testcase.expectedError)
			} else if !assert.Check(t, err) {
				return
			}
			assert.Check(t, is.Equal(testcase.expectedOut, fakeCLI.OutBuffer().String()))
			assert.Check(t, is.Equal(testcase.expectedErr, fakeCLI.ErrBuffer().String()))
		})
	}
}

func execCreateWithID(_ string, _ container.ExecOptions) (container.ExecCreateResponse, error) {
	return container.ExecCreateResponse{ID: "execid"}, nil
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
			execInspectFunc: func(id string) (container.ExecInspect, error) {
				assert.Check(t, is.Equal(execID, id))
				return container.ExecInspect{ExitCode: testcase.exitCode}, testcase.inspectError
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
		containerInspectFunc func(img string) (container.InspectResponse, error)
	}{
		{
			name:          "client-error",
			args:          []string{"5cb5bb5e4a3b", "-t", "-i", "bash"},
			expectedError: "something went wrong",
			containerInspectFunc: func(containerID string) (container.InspectResponse, error) {
				return container.InspectResponse{}, errors.New("something went wrong")
			},
		},
	}
	for _, tc := range testCases {
		fakeCLI := test.NewFakeCli(&fakeClient{inspectFunc: tc.containerInspectFunc})
		cmd := NewExecCommand(fakeCLI)
		cmd.SetOut(io.Discard)
		cmd.SetArgs(tc.args)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}
