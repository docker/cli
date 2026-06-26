package stack

import (
	"context"
	"errors"
	"testing"

	"github.com/docker/cli/internal/test/network"
	networktypes "github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
)

type notFound struct {
	error
}

func (notFound) NotFound() {}

func TestValidateExternalNetworks(t *testing.T) {
	testcases := []struct {
		inspectResponse client.NetworkInspectResult
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
			network: "user",
			inspectResponse: client.NetworkInspectResult{
				Network: networktypes.Inspect{
					Network: networktypes.Network{
						Scope: "swarm",
					},
				},
			},
		},
	}

	for _, testcase := range testcases {
		fakeAPIClient := &network.FakeClient{
			NetworkInspectFunc: func(_ context.Context, _ string, _ client.NetworkInspectOptions) (client.NetworkInspectResult, error) {
				return testcase.inspectResponse, testcase.inspectError
			},
		}
		networks := []string{testcase.network}
		err := validateExternalNetworks(context.Background(), fakeAPIClient, networks)
		if testcase.expectedMsg == "" {
			assert.NilError(t, err)
		} else {
			assert.ErrorContains(t, err, testcase.expectedMsg)
		}
	}
}
