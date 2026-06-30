package network

import (
	"context"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
)

func TestNetworkEditNoChanges(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cmd := newEditCommand(cli)
	cmd.SetArgs([]string{"mynet"})
	err := cmd.Execute()
	assert.ErrorContains(t, err, "no changes requested")
}

func TestNetworkEditActiveEndpoints(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		networkInspectFunc: func(_ context.Context, _ string, _ client.NetworkInspectOptions) (client.NetworkInspectResult, error) {
			return client.NetworkInspectResult{
				Network: network.Inspect{
					Network: network.Network{
						Name: "mynet",
						ID:   "abc123",
					},
					Containers: map[string]network.EndpointResource{
						"ep1": {Name: "mycontainer"},
					},
				},
			}, nil
		},
	})
	cmd := newEditCommand(cli)
	cmd.SetArgs([]string{"--label-add", "env=prod", "mynet"})
	err := cmd.Execute()
	assert.ErrorContains(t, err, "active endpoints")
	assert.ErrorContains(t, err, "mycontainer")
}

func TestNetworkEditLabelAdd(t *testing.T) {
	var createCalled bool
	var removeCalled bool

	fakeNetwork := network.Inspect{
		Network: network.Network{
			Name:    "mynet",
			ID:      "abc123",
			Driver:  "bridge",
			Options: map[string]string{},
			Labels:  map[string]string{"existing": "value"},
		},
	}

	cli := test.NewFakeCli(&fakeClient{
		networkInspectFunc: func(_ context.Context, _ string, _ client.NetworkInspectOptions) (client.NetworkInspectResult, error) {
			return client.NetworkInspectResult{Network: fakeNetwork}, nil
		},
		networkRemoveFunc: func(_ context.Context, _ string) error {
			removeCalled = true
			return nil
		},
		networkCreateFunc: func(_ context.Context, name string, opts client.NetworkCreateOptions) (client.NetworkCreateResult, error) {
			createCalled = true
			assert.Equal(t, "mynet", name)
			assert.Equal(t, "prod", opts.Labels["env"])
			assert.Equal(t, "value", opts.Labels["existing"])
			return client.NetworkCreateResult{ID: "newid123"}, nil
		},
	})

	cmd := newEditCommand(cli)
	cmd.SetArgs([]string{"--label-add", "env=prod", "mynet"})
	assert.NilError(t, cmd.Execute())
	assert.Assert(t, removeCalled)
	assert.Assert(t, createCalled)
}

func TestNetworkEditLabelRemove(t *testing.T) {
	fakeNetwork := network.Inspect{
		Network: network.Network{
			Name:    "mynet",
			ID:      "abc123",
			Driver:  "bridge",
			Options: map[string]string{},
			Labels:  map[string]string{"env": "prod", "keep": "me"},
		},
	}

	cli := test.NewFakeCli(&fakeClient{
		networkInspectFunc: func(_ context.Context, _ string, _ client.NetworkInspectOptions) (client.NetworkInspectResult, error) {
			return client.NetworkInspectResult{Network: fakeNetwork}, nil
		},
		networkRemoveFunc: func(_ context.Context, _ string) error { return nil },
		networkCreateFunc: func(_ context.Context, _ string, opts client.NetworkCreateOptions) (client.NetworkCreateResult, error) {
			_, hasEnv := opts.Labels["env"]
			assert.Assert(t, !hasEnv, "expected 'env' label to be removed")
			assert.Equal(t, "me", opts.Labels["keep"])
			return client.NetworkCreateResult{ID: "newid456"}, nil
		},
	})

	cmd := newEditCommand(cli)
	cmd.SetArgs([]string{"--label-rm", "env", "mynet"})
	assert.NilError(t, cmd.Execute())
}

func TestNetworkEditOutputsNewID(t *testing.T) {
	fakeNetwork := network.Inspect{
		Network: network.Network{
			Name:    "mynet",
			ID:      "abc123",
			Driver:  "bridge",
			Options: map[string]string{},
		},
	}

	cli := test.NewFakeCli(&fakeClient{
		networkInspectFunc: func(_ context.Context, _ string, _ client.NetworkInspectOptions) (client.NetworkInspectResult, error) {
			return client.NetworkInspectResult{Network: fakeNetwork}, nil
		},
		networkRemoveFunc: func(_ context.Context, _ string) error { return nil },
		networkCreateFunc: func(_ context.Context, _ string, _ client.NetworkCreateOptions) (client.NetworkCreateResult, error) {
			return client.NetworkCreateResult{ID: "newid789"}, nil
		},
	})

	cmd := newEditCommand(cli)
	cmd.SetArgs([]string{"--label-add", "foo=bar", "mynet"})
	assert.NilError(t, cmd.Execute())
	assert.Equal(t, "newid789\n", cli.OutBuffer().String())
}
