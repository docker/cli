package network

import (
	"testing"

	"io/ioutil"

	"strings"

	"github.com/docker/cli/internal/test"
	. "github.com/docker/cli/internal/test/builders"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestNetworkListErrors(t *testing.T) {
	testCases := []struct {
		networkListFunc func(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error)
		expectedError   string
	}{
		{
			networkListFunc: func(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error) {
				return []types.NetworkResource{}, errors.Errorf("error creating network")
			},
			expectedError: "error creating network",
		},
	}

	for _, tc := range testCases {
		cmd := newListCommand(
			test.NewFakeCli(&fakeClient{
				networkListFunc: tc.networkListFunc,
			}),
		)
		cmd.SetOutput(ioutil.Discard)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)

	}
}

func TestNetworkListWithFlags(t *testing.T) {

	filterArgs := filters.NewArgs()
	filterArgs.Add("image.name", "ubuntu")

	expectedOpts := types.NetworkListOptions{
		Filters: filterArgs,
	}

	cli := test.NewFakeCli(&fakeClient{
		networkListFunc: func(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error) {
			assert.Equal(t, expectedOpts, options, "not expected options error")
			return []types.NetworkResource{*NetworkResource(NetworkResourceID("123454321"),
				NetworkResourceName("network_1"),
				NetworkResourceDriver("09.7.01"),
				NetworkResourceScope("global"))}, nil
		},
	})
	cmd := newListCommand(cli)

	cmd.Flags().Set("filter", "image.name=ubuntu")
	assert.NoError(t, cmd.Execute())
	golden.Assert(t, strings.TrimSpace(cli.OutBuffer().String()), "network-list.golden")
}
