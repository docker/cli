package container

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/testutil"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

type arguments struct {
	options execOptions
	execCmd []string
}

func TestParseExec(t *testing.T) {
	valids := map[*arguments]*types.ExecConfig{
		{
			execCmd: []string{"command"},
		}: {
			Cmd:          []string{"command"},
			AttachStdout: true,
			AttachStderr: true,
		},
		{
			execCmd: []string{"command1", "command2"},
		}: {
			Cmd:          []string{"command1", "command2"},
			AttachStdout: true,
			AttachStderr: true,
		},
		{
			options: execOptions{
				interactive: true,
				tty:         true,
				user:        "uid",
			},
			execCmd: []string{"command"},
		}: {
			User:         "uid",
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Tty:          true,
			Cmd:          []string{"command"},
		},
		{
			options: execOptions{
				detach: true,
			},
			execCmd: []string{"command"},
		}: {
			AttachStdin:  false,
			AttachStdout: false,
			AttachStderr: false,
			Detach:       true,
			Cmd:          []string{"command"},
		},
		{
			options: execOptions{
				tty:         true,
				interactive: true,
				detach:      true,
			},
			execCmd: []string{"command"},
		}: {
			AttachStdin:  false,
			AttachStdout: false,
			AttachStderr: false,
			Detach:       true,
			Tty:          true,
			Cmd:          []string{"command"},
		},
	}

	for valid, expectedExecConfig := range valids {
		execConfig, err := parseExec(&valid.options, valid.execCmd)
		require.NoError(t, err)
		if !compareExecConfig(expectedExecConfig, execConfig) {
			t.Fatalf("Expected [%v] for %v, got [%v]", expectedExecConfig, valid, execConfig)
		}
	}
}

func compareExecConfig(config1 *types.ExecConfig, config2 *types.ExecConfig) bool {
	if config1.AttachStderr != config2.AttachStderr {
		return false
	}
	if config1.AttachStdin != config2.AttachStdin {
		return false
	}
	if config1.AttachStdout != config2.AttachStdout {
		return false
	}
	if config1.Detach != config2.Detach {
		return false
	}
	if config1.Privileged != config2.Privileged {
		return false
	}
	if config1.Tty != config2.Tty {
		return false
	}
	if config1.User != config2.User {
		return false
	}
	if len(config1.Cmd) != len(config2.Cmd) {
		return false
	}
	for index, value := range config1.Cmd {
		if value != config2.Cmd[index] {
			return false
		}
	}
	return true
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
