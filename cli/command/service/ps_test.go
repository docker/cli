package service

import (
	"context"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/opts"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestCreateFilter(t *testing.T) {
	apiClient := &fakeClient{
		serviceListFunc: func(ctx context.Context, options client.ServiceListOptions) (client.ServiceListResult, error) {
			return client.ServiceListResult{
				Items: []swarm.Service{
					{ID: "idmatch"},
					{ID: "idprefixmatch"},
					newService("cccccccc", "namematch"),
					newService("01010101", "notfoundprefix"),
				},
			}, nil
		},
	}

	filter := opts.NewFilterOpt()
	assert.NilError(t, filter.Set("node=somenode"))
	options := psOptions{
		services: []string{"idmatch", "idprefix", "namematch", "notfound"},
		filter:   filter,
	}

	actual, notfound, err := createFilter(context.Background(), apiClient, options)
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(notfound, []string{"no such service: notfound"}))

	expected := make(client.Filters).Add("service", "idmatch").Add("service", "idprefixmatch").Add("service", "cccccccc").Add("node", "somenode")
	assert.DeepEqual(t, expected, actual)
}

func TestCreateFilterWithAmbiguousIDPrefixError(t *testing.T) {
	apiClient := &fakeClient{
		serviceListFunc: func(ctx context.Context, options client.ServiceListOptions) (client.ServiceListResult, error) {
			return client.ServiceListResult{
				Items: []swarm.Service{
					{ID: "aaaone"},
					{ID: "aaatwo"},
				},
			}, nil
		},
	}
	_, _, err := createFilter(context.Background(), apiClient, psOptions{
		services: []string{"aaa"},
		filter:   opts.NewFilterOpt(),
	})
	assert.Error(t, err, "multiple services found with provided prefix: aaa")
}

func TestCreateFilterNoneFound(t *testing.T) {
	apiClient := &fakeClient{}
	options := psOptions{
		services: []string{"foo", "notfound"},
		filter:   opts.NewFilterOpt(),
	}
	_, _, err := createFilter(context.Background(), apiClient, options)
	assert.Error(t, err, "no such service: foo\nno such service: notfound")
}

func TestRunPSWarnsOnNotFound(t *testing.T) {
	apiClient := &fakeClient{
		serviceListFunc: func(ctx context.Context, options client.ServiceListOptions) (client.ServiceListResult, error) {
			return client.ServiceListResult{
				Items: []swarm.Service{{ID: "foo"}},
			}, nil
		},
	}

	cli := test.NewFakeCli(apiClient)
	options := psOptions{
		services: []string{"foo", "bar"},
		filter:   opts.NewFilterOpt(),
		format:   "{{.ID}}",
	}

	ctx := context.Background()
	err := runPS(ctx, cli, options)
	assert.Error(t, err, "no such service: bar")
}

func TestRunPSQuiet(t *testing.T) {
	apiClient := &fakeClient{
		serviceListFunc: func(ctx context.Context, options client.ServiceListOptions) (client.ServiceListResult, error) {
			return client.ServiceListResult{
				Items: []swarm.Service{{ID: "foo"}},
			}, nil
		},
		taskListFunc: func(ctx context.Context, options client.TaskListOptions) (client.TaskListResult, error) {
			return client.TaskListResult{
				Items: []swarm.Task{{ID: "sxabyp0obqokwekpun4rjo0b3"}},
			}, nil
		},
	}

	cli := test.NewFakeCli(apiClient)
	ctx := context.Background()
	err := runPS(ctx, cli, psOptions{services: []string{"foo"}, quiet: true, filter: opts.NewFilterOpt()})
	assert.NilError(t, err)
	assert.Check(t, is.Equal("sxabyp0obqokwekpun4rjo0b3\n", cli.OutBuffer().String()))
}

func TestUpdateNodeFilter(t *testing.T) {
	selfNodeID := "foofoo"
	filter := make(client.Filters).Add("node", "one", "two", "self")

	apiClient := &fakeClient{
		infoFunc: func(_ context.Context) (client.SystemInfoResult, error) {
			return client.SystemInfoResult{
				Info: system.Info{
					Swarm: swarm.Info{NodeID: selfNodeID},
				},
			}, nil
		},
	}

	err := updateNodeFilter(context.Background(), apiClient, filter)
	assert.NilError(t, err)

	expected := make(client.Filters).Add("node", "one", "two", selfNodeID)
	assert.DeepEqual(t, expected, filter)
}
