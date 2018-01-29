package swarm

import (
	"testing"

	"github.com/docker/cli/internal/test/network"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

type notFound struct {
	error
}

func (n notFound) NotFound() bool {
	return true
}

func TestValidateExternalNetworks(t *testing.T) {
	var testcases = []struct {
		inspectResponse types.NetworkResource
		inspectError    error
		expectedMsg     string
		network         string
	}{
		{
			inspectError: notFound{},
			expectedMsg:  "could not be found. You need to create a swarm-scoped network",
		},
		{
			inspectError: errors.New("Unexpected"),
			expectedMsg:  "Unexpected",
		},
		{
			inspectError: errors.New("host net does not exist on swarm classic"),
			network:      "host",
		},
		{
			network:     "user",
			expectedMsg: "is not in the right scope",
		},
		{
			network:         "user",
			inspectResponse: types.NetworkResource{Scope: "swarm"},
		},
	}

	for _, testcase := range testcases {
		fakeClient := &network.FakeClient{
			NetworkInspectFunc: func(_ context.Context, _ string, _ types.NetworkInspectOptions) (types.NetworkResource, error) {
				return testcase.inspectResponse, testcase.inspectError
			},
		}
		networks := []string{testcase.network}
		err := validateExternalNetworks(context.Background(), fakeClient, networks)
		if testcase.expectedMsg == "" {
			assert.NoError(t, err)
		} else {
			testutil.ErrorContains(t, err, testcase.expectedMsg)
		}
	}
}
