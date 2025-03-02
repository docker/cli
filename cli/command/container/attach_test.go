package container

import (
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/container"
	"gotest.tools/v3/assert"
)

func TestNewAttachCommandErrors(t *testing.T) {
	testCases := []struct {
		name                 string
		args                 []string
		expectedError        string
		containerInspectFunc func(img string) (container.InspectResponse, error)
	}{
		{
			name:          "client-error",
			args:          []string{"5cb5bb5e4a3b"},
			expectedError: "something went wrong",
			containerInspectFunc: func(containerID string) (container.InspectResponse, error) {
				return container.InspectResponse{}, errors.New("something went wrong")
			},
		},
		{
			name:          "client-stopped",
			args:          []string{"5cb5bb5e4a3b"},
			expectedError: "You cannot attach to a stopped container",
			containerInspectFunc: func(containerID string) (container.InspectResponse, error) {
				return container.InspectResponse{
					ContainerJSONBase: &container.ContainerJSONBase{
						State: &container.State{
							Running: false,
						},
					},
				}, nil
			},
		},
		{
			name:          "client-paused",
			args:          []string{"5cb5bb5e4a3b"},
			expectedError: "You cannot attach to a paused container",
			containerInspectFunc: func(containerID string) (container.InspectResponse, error) {
				return container.InspectResponse{
					ContainerJSONBase: &container.ContainerJSONBase{
						State: &container.State{
							Running: true,
							Paused:  true,
						},
					},
				}, nil
			},
		},
		{
			name:          "client-restarting",
			args:          []string{"5cb5bb5e4a3b"},
			expectedError: "You cannot attach to a restarting container",
			containerInspectFunc: func(containerID string) (container.InspectResponse, error) {
				return container.InspectResponse{
					ContainerJSONBase: &container.ContainerJSONBase{
						State: &container.State{
							Running:    true,
							Paused:     false,
							Restarting: true,
						},
					},
				}, nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewAttachCommand(test.NewFakeCli(&fakeClient{inspectFunc: tc.containerInspectFunc}))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestGetExitStatus(t *testing.T) {
	expectedErr := errors.New("unexpected error")

	testcases := []struct {
		result        *container.WaitResponse
		err           error
		expectedError error
	}{
		{
			result: &container.WaitResponse{
				StatusCode: 0,
			},
		},
		{
			err:           expectedErr,
			expectedError: expectedErr,
		},
		{
			result: &container.WaitResponse{
				Error: &container.WaitExitError{Message: expectedErr.Error()},
			},
			expectedError: expectedErr,
		},
		{
			result: &container.WaitResponse{
				StatusCode: 15,
			},
			expectedError: cli.StatusError{StatusCode: 15},
		},
	}

	for _, testcase := range testcases {
		errC := make(chan error, 1)
		resultC := make(chan container.WaitResponse, 1)
		if testcase.err != nil {
			errC <- testcase.err
		}
		if testcase.result != nil {
			resultC <- *testcase.result
		}

		err := getExitStatus(errC, resultC)

		if testcase.expectedError == nil {
			assert.NilError(t, err)
		} else {
			assert.Error(t, err, testcase.expectedError.Error())
		}
	}
}
