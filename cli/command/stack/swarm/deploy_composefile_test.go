package swarm

import (
	"context"
	"errors"
	"testing"

	"github.com/docker/cli/v28/internal/test/network"
	networktypes "github.com/docker/docker/api/types/network"
	"gotest.tools/v3/assert"
)

type notFound struct {
	error
}

func (notFound) NotFound() {}

func TestValidateExternalNetworks(t *testing.T) {
	testcases := []struct {
		inspectResponse networktypes.Inspect
		inspectError    error
		expectedMsg     string
		network         string
	}{
		{
			inspectError: notFound{},
			expectedMsg:  "could not be found. You need to create a swarm-scoped network",
		},
		{
			inspectError: errors.New("unexpected"),
			expectedMsg:  "unexpected",
		},
		// FIXME(vdemeester) that doesn't work under windows, the check needs to be smarter
		/*
			{
				inspectError: errors.New("host net does not exist on swarm classic"),
				network:      "host",
			},
		*/
		{
			network:     "user",
			expectedMsg: "is not in the right scope",
		},
		{
			network:         "user",
			inspectResponse: networktypes.Inspect{Scope: "swarm"},
		},
	}

	for _, testcase := range testcases {
		client := &network.FakeClient{
			NetworkInspectFunc: func(_ context.Context, _ string, _ networktypes.InspectOptions) (networktypes.Inspect, error) {
				return testcase.inspectResponse, testcase.inspectError
			},
		}
		networks := []string{testcase.network}
		err := validateExternalNetworks(context.Background(), client, networks)
		if testcase.expectedMsg == "" {
			assert.NilError(t, err)
		} else {
			assert.ErrorContains(t, err, testcase.expectedMsg)
		}
	}
}
