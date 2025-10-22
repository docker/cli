package idresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/docker/cli/internal/test/builders"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestResolveError(t *testing.T) {
	apiClient := &fakeClient{
		nodeInspectFunc: func(nodeID string) (client.NodeInspectResult, error) {
			return client.NodeInspectResult{}, errors.New("error inspecting node")
		},
	}

	idResolver := New(apiClient, false)
	_, err := idResolver.Resolve(context.Background(), struct{}{}, "nodeID")

	assert.Error(t, err, "unsupported type")
}

func TestResolveWithNoResolveOption(t *testing.T) {
	resolved := false
	apiClient := &fakeClient{
		nodeInspectFunc: func(nodeID string) (client.NodeInspectResult, error) {
			resolved = true
			return client.NodeInspectResult{}, nil
		},
		serviceInspectFunc: func(serviceID string) (client.ServiceInspectResult, error) {
			resolved = true
			return client.ServiceInspectResult{}, nil
		},
	}

	idResolver := New(apiClient, true)
	id, err := idResolver.Resolve(context.Background(), swarm.Node{}, "nodeID")

	assert.NilError(t, err)
	assert.Check(t, is.Equal("nodeID", id))
	assert.Check(t, !resolved)
}

func TestResolveWithCache(t *testing.T) {
	inspectCounter := 0
	apiClient := &fakeClient{
		nodeInspectFunc: func(string) (client.NodeInspectResult, error) {
			inspectCounter++
			return client.NodeInspectResult{
				Node: *builders.Node(builders.NodeName("node-foo")),
			}, nil
		},
	}

	idResolver := New(apiClient, false)

	ctx := context.Background()
	for i := 0; i < 2; i++ {
		id, err := idResolver.Resolve(ctx, swarm.Node{}, "nodeID")
		assert.NilError(t, err)
		assert.Check(t, is.Equal("node-foo", id))
	}

	assert.Check(t, is.Equal(1, inspectCounter))
}

func TestResolveNode(t *testing.T) {
	testCases := []struct {
		nodeID          string
		nodeInspectFunc func(string) (client.NodeInspectResult, error)
		expectedID      string
	}{
		{
			nodeID: "nodeID",
			nodeInspectFunc: func(string) (client.NodeInspectResult, error) {
				return client.NodeInspectResult{}, errors.New("error inspecting node")
			},
			expectedID: "nodeID",
		},
		{
			nodeID: "nodeID",
			nodeInspectFunc: func(string) (client.NodeInspectResult, error) {
				return client.NodeInspectResult{
					Node: *builders.Node(builders.NodeName("node-foo")),
				}, nil
			},
			expectedID: "node-foo",
		},
		{
			nodeID: "nodeID",
			nodeInspectFunc: func(string) (client.NodeInspectResult, error) {
				return client.NodeInspectResult{
					Node: *builders.Node(builders.NodeName(""), builders.Hostname("node-hostname")),
				}, nil
			},
			expectedID: "node-hostname",
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		apiClient := &fakeClient{
			nodeInspectFunc: tc.nodeInspectFunc,
		}
		idResolver := New(apiClient, false)
		id, err := idResolver.Resolve(ctx, swarm.Node{}, tc.nodeID)

		assert.NilError(t, err)
		assert.Check(t, is.Equal(tc.expectedID, id))
	}
}

func TestResolveService(t *testing.T) {
	testCases := []struct {
		serviceID          string
		serviceInspectFunc func(string) (client.ServiceInspectResult, error)
		expectedID         string
	}{
		{
			serviceID: "serviceID",
			serviceInspectFunc: func(string) (client.ServiceInspectResult, error) {
				return client.ServiceInspectResult{}, errors.New("error inspecting service")
			},
			expectedID: "serviceID",
		},
		{
			serviceID: "serviceID",
			serviceInspectFunc: func(string) (client.ServiceInspectResult, error) {
				return client.ServiceInspectResult{
					Service: *builders.Service(builders.ServiceName("service-foo")),
				}, nil
			},
			expectedID: "service-foo",
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		apiClient := &fakeClient{
			serviceInspectFunc: tc.serviceInspectFunc,
		}
		idResolver := New(apiClient, false)
		id, err := idResolver.Resolve(ctx, swarm.Service{}, tc.serviceID)

		assert.NilError(t, err)
		assert.Check(t, is.Equal(tc.expectedID, id))
	}
}
