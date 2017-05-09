package container

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestNewAttachCommandErrors(t *testing.T) {
	testCases := []struct {
		name                 string
		args                 []string
		expectedError        string
		containerInspectFunc func(img string) (types.ContainerJSON, error)
	}{
		{
			name:          "client-error",
			args:          []string{"5cb5bb5e4a3b"},
			expectedError: "something went wrong",
			containerInspectFunc: func(containerID string) (types.ContainerJSON, error) {
				return types.ContainerJSON{}, errors.Errorf("something went wrong")
			},
		},
		{
			name:          "client-stopped",
			args:          []string{"5cb5bb5e4a3b"},
			expectedError: "You cannot attach to a stopped container",
			containerInspectFunc: func(containerID string) (types.ContainerJSON, error) {
				c := types.ContainerJSON{}
				c.ContainerJSONBase = &types.ContainerJSONBase{}
				c.ContainerJSONBase.State = &types.ContainerState{Running: false}
				return c, nil
			},
		},
		{
			name:          "client-paused",
			args:          []string{"5cb5bb5e4a3b"},
			expectedError: "You cannot attach to a paused container",
			containerInspectFunc: func(containerID string) (types.ContainerJSON, error) {
				c := types.ContainerJSON{}
				c.ContainerJSONBase = &types.ContainerJSONBase{}
				c.ContainerJSONBase.State = &types.ContainerState{
					Running: true,
					Paused:  true,
				}
				return c, nil
			},
		},
		{
			name:          "client-restarting",
			args:          []string{"5cb5bb5e4a3b"},
			expectedError: "You cannot attach to a restarting container",
			containerInspectFunc: func(containerID string) (types.ContainerJSON, error) {
				c := types.ContainerJSON{}
				c.ContainerJSONBase = &types.ContainerJSONBase{}
				c.ContainerJSONBase.State = &types.ContainerState{
					Running:    true,
					Paused:     false,
					Restarting: true,
				}
				return c, nil
			},
		},
	}
	for _, tc := range testCases {
		cmd := NewAttachCommand(test.NewFakeCli(&fakeClient{inspectFunc: tc.containerInspectFunc}))
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestGetExitStatus(t *testing.T) {
	containerID := "the exec id"
	expecatedErr := errors.New("unexpected error")

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
			inspectFunc: func(id string) (types.ContainerJSON, error) {
				assert.Equal(t, containerID, id)
				return types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						State: &types.ContainerState{ExitCode: testcase.exitCode},
					},
				}, testcase.inspectError
			},
		}
		err := getExitStatus(context.Background(), client, containerID)
		assert.Equal(t, testcase.expectedError, err)
	}
}
