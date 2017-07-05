package container

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/testutil"
	"github.com/pkg/errors"
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
		buf := new(bytes.Buffer)
		cmd := NewAttachCommand(test.NewFakeCliWithOutput(&fakeClient{containerInspectFunc: tc.containerInspectFunc}, buf))
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}
