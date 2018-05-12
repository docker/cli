package network

import (
	"context"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	. "github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/google/go-cmp/cmp"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/pkg/errors"
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
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNetworkListWithFlags(t *testing.T) {
	expectedOpts := types.NetworkListOptions{
		Filters: filters.NewArgs(filters.Arg("image.name", "ubuntu")),
	}

	cli := test.NewFakeCli(&fakeClient{
		networkListFunc: func(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error) {
			assert.Check(t, is.DeepEqual(expectedOpts, options, cmp.AllowUnexported(filters.Args{})))
			return []types.NetworkResource{*NetworkResource(NetworkResourceID("123454321"),
				NetworkResourceName("network_1"),
				NetworkResourceDriver("09.7.01"),
				NetworkResourceScope("global"))}, nil
		},
	})
	cmd := newListCommand(cli)

	cmd.Flags().Set("filter", "image.name=ubuntu")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, strings.TrimSpace(cli.OutBuffer().String()), "network-list.golden")
}
