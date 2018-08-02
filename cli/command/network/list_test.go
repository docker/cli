package network

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	. "github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/golden"
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

func TestNetworkList(t *testing.T) {
	testCases := []struct {
		doc             string
		networkListFunc func(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error)
		flags           map[string]string
		golden          string
	}{
		{
			doc: "network list with flags",
			flags: map[string]string{
				"filter": "image.name=ubuntu",
			},
			golden: "network-list.golden",
			networkListFunc: func(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error) {
				expectedOpts := types.NetworkListOptions{
					Filters: filters.NewArgs(filters.Arg("image.name", "ubuntu")),
				}
				assert.Check(t, is.DeepEqual(expectedOpts, options, cmp.AllowUnexported(filters.Args{})))

				return []types.NetworkResource{*NetworkResource(NetworkResourceID("123454321"),
					NetworkResourceName("network_1"),
					NetworkResourceDriver("09.7.01"),
					NetworkResourceScope("global"))}, nil
			},
		},
		{
			doc: "network list sort order",
			flags: map[string]string{
				"format": "{{ .Name }}",
			},
			golden: "network-list-sort.golden",
			networkListFunc: func(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error) {
				return []types.NetworkResource{
					*NetworkResource(NetworkResourceName("network-2-foo")),
					*NetworkResource(NetworkResourceName("network-1-foo")),
					*NetworkResource(NetworkResourceName("network-10-foo"))}, nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{networkListFunc: tc.networkListFunc})
			cmd := newListCommand(cli)
			for key, value := range tc.flags {
				cmd.Flags().Set(key, value)
			}
			assert.NilError(t, cmd.Execute())
			golden.Assert(t, cli.OutBuffer().String(), tc.golden)
		})
	}
}
