package container

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/testutil"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestParseExec(t *testing.T) {
	testcases := []struct {
		options    execOptions
		execCmd    []string
		configFile configfile.ConfigFile
		expected   types.ExecConfig
	}{
		{
			execCmd: []string{"command"},
			expected: types.ExecConfig{
				Cmd:          []string{"command"},
				AttachStdout: true,
				AttachStderr: true,
			},
		},
		{
			execCmd: []string{"command1", "command2"},
			expected: types.ExecConfig{
				Cmd:          []string{"command1", "command2"},
				AttachStdout: true,
				AttachStderr: true,
			},
		},
		{
			options: execOptions{
				interactive: true,
				tty:         true,
				user:        "uid",
			},
			execCmd: []string{"command"},
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
			options: execOptions{
				detach: true,
			},
			execCmd: []string{"command"},
			expected: types.ExecConfig{
				Detach: true,
				Cmd:    []string{"command"},
			},
		},
		{
			options: execOptions{
				tty:         true,
				interactive: true,
				detach:      true,
			},
			execCmd: []string{"command"},
			expected: types.ExecConfig{
				Detach: true,
				Tty:    true,
				Cmd:    []string{"command"},
			},
		},
		{
			execCmd:    []string{"command"},
			options:    execOptions{detach: true},
			configFile: configfile.ConfigFile{DetachKeys: "de"},
			expected: types.ExecConfig{
				Cmd:        []string{"command"},
				DetachKeys: "de",
				Detach:     true,
			},
		},
		{
			execCmd:    []string{"command"},
			options:    execOptions{detach: true, detachKeys: "ab"},
			configFile: configfile.ConfigFile{DetachKeys: "de"},
			expected: types.ExecConfig{
				Cmd:        []string{"command"},
				DetachKeys: "ab",
				Detach:     true,
			},
		},
	}

	for _, testcase := range testcases {
		execConfig := parseExec(&testcase.options, &testcase.configFile, testcase.execCmd)
		assert.Equal(t, testcase.expected, *execConfig)
	}
}

func TestRunExec(t *testing.T) {
	client := &fakeClient{}
	cli := test.NewFakeCli(client)
	options := &execOptions{detach: true}

	err := runExec(cli, options, "cid", []string{"bash"})
	require.NoError(t, err)
}

func TestGetExecExitStatus(t *testing.T) {
	execID := "the exec id"
	expecatedErr := errors.New("unexpected error")

	testcases := []struct {
		inspectError  error
		exitCode      int
		expectedError error
	}{
		{
			inspectError:  nil,
			expectedError: nil,
		},
		{
			inspectError:  expecatedErr,
			expectedError: expecatedErr,
		},
		{
			exitCode:      15,
			expectedError: cli.StatusError{StatusCode: 15},
		},
	}

	for _, testcase := range testcases {
		client := &fakeClient{
			execInspectFunc: func(ctx context.Context, id string) (types.ContainerExecInspect, error) {
				assert.Equal(t, execID, id)
				return types.ContainerExecInspect{ExitCode: testcase.exitCode}, testcase.inspectError
			},
		}
		err := getExecExitStatus(context.Background(), client, execID)
		assert.Equal(t, testcase.expectedError, err)
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
		cli := test.NewFakeCli(&fakeClient{containerInspectFunc: tc.containerInspectFunc})
		cmd := NewExecCommand(cli)
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}
