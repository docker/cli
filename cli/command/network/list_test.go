package network

import (
	"context"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestNetworkListErrors(t *testing.T) {
	testCases := []struct {
		networkListFunc func(ctx context.Context, options network.ListOptions) ([]network.Summary, error)
		expectedError   string
	}{
		{
			networkListFunc: func(ctx context.Context, options network.ListOptions) ([]network.Summary, error) {
				return []network.Summary{}, errors.Errorf("error creating network")
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
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNetworkList(t *testing.T) {
	testCases := []struct {
		doc             string
		networkListFunc func(ctx context.Context, options network.ListOptions) ([]network.Summary, error)
		flags           map[string]string
		golden          string
	}{
		{
			doc: "network list with flags",
			flags: map[string]string{
				"filter": "image.name=ubuntu",
			},
			golden: "network-list.golden",
			networkListFunc: func(ctx context.Context, options network.ListOptions) ([]network.Summary, error) {
				expectedOpts := network.ListOptions{
					Filters: filters.NewArgs(filters.Arg("image.name", "ubuntu")),
				}
				assert.Check(t, is.DeepEqual(expectedOpts, options, cmp.AllowUnexported(filters.Args{})))

				return []network.Summary{*builders.NetworkResource(builders.NetworkResourceID("123454321"),
					builders.NetworkResourceName("network_1"),
					builders.NetworkResourceDriver("09.7.01"),
					builders.NetworkResourceScope("global"))}, nil
			},
		},
		{
			doc: "network list sort order",
			flags: map[string]string{
				"format": "{{ .Name }}",
			},
			golden: "network-list-sort.golden",
			networkListFunc: func(ctx context.Context, options network.ListOptions) ([]network.Summary, error) {
				return []network.Summary{
					*builders.NetworkResource(builders.NetworkResourceName("network-2-foo")),
					*builders.NetworkResource(builders.NetworkResourceName("network-1-foo")),
					*builders.NetworkResource(builders.NetworkResourceName("network-10-foo")),
				}, nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{networkListFunc: tc.networkListFunc})
			cmd := newListCommand(cli)
			for key, value := range tc.flags {
				assert.Check(t, cmd.Flags().Set(key, value))
			}
			assert.NilError(t, cmd.Execute())
			golden.Assert(t, cli.OutBuffer().String(), tc.golden)
		})
	}
}
