package service

import (
	"context"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/google/go-cmp/cmp"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestCreateFilter(t *testing.T) {
	client := &fakeClient{
		serviceListFunc: func(ctx context.Context, options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{
				{ID: "idmatch"},
				{ID: "idprefixmatch"},
				newService("cccccccc", "namematch"),
				newService("01010101", "notfoundprefix"),
			}, nil
		},
	}

	filter := opts.NewFilterOpt()
	assert.NilError(t, filter.Set("node=somenode"))
	options := psOptions{
		services: []string{"idmatch", "idprefix", "namematch", "notfound"},
		filter:   filter,
	}

	actual, notfound, err := createFilter(context.Background(), client, options)
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(notfound, []string{"no such service: notfound"}))

	expected := filters.NewArgs(
		filters.Arg("service", "idmatch"),
		filters.Arg("service", "idprefixmatch"),
		filters.Arg("service", "cccccccc"),
		filters.Arg("node", "somenode"),
	)
	assert.DeepEqual(t, expected, actual, cmpFilters)
}

func TestCreateFilterWithAmbiguousIDPrefixError(t *testing.T) {
	client := &fakeClient{
		serviceListFunc: func(ctx context.Context, options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{
				{ID: "aaaone"},
				{ID: "aaatwo"},
			}, nil
		},
	}
	options := psOptions{
		services: []string{"aaa"},
		filter:   opts.NewFilterOpt(),
	}
	_, _, err := createFilter(context.Background(), client, options)
	assert.Error(t, err, "multiple services found with provided prefix: aaa")
}

func TestCreateFilterNoneFound(t *testing.T) {
	client := &fakeClient{}
	options := psOptions{
		services: []string{"foo", "notfound"},
		filter:   opts.NewFilterOpt(),
	}
	_, _, err := createFilter(context.Background(), client, options)
	assert.Error(t, err, "no such service: foo\nno such service: notfound")
}

func TestRunPSWarnsOnNotFound(t *testing.T) {
	client := &fakeClient{
		serviceListFunc: func(ctx context.Context, options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{
				{ID: "foo"},
			}, nil
		},
	}

	cli := test.NewFakeCli(client)
	options := psOptions{
		services: []string{"foo", "bar"},
		filter:   opts.NewFilterOpt(),
		format:   "{{.ID}}",
	}
	err := runPS(cli, options)
	assert.Error(t, err, "no such service: bar")
}

func TestRunPSQuiet(t *testing.T) {
	client := &fakeClient{
		serviceListFunc: func(ctx context.Context, options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{{ID: "foo"}}, nil
		},
		taskListFunc: func(ctx context.Context, options types.TaskListOptions) ([]swarm.Task, error) {
			return []swarm.Task{{ID: "sxabyp0obqokwekpun4rjo0b3"}}, nil
		},
	}

	cli := test.NewFakeCli(client)
	err := runPS(cli, psOptions{services: []string{"foo"}, quiet: true, filter: opts.NewFilterOpt()})
	assert.NilError(t, err)
	assert.Check(t, is.Equal("sxabyp0obqokwekpun4rjo0b3\n", cli.OutBuffer().String()))
}

func TestUpdateNodeFilter(t *testing.T) {
	selfNodeID := "foofoo"
	filter := filters.NewArgs(
		filters.Arg("node", "one"),
		filters.Arg("node", "two"),
		filters.Arg("node", "self"),
	)

	client := &fakeClient{
		infoFunc: func(_ context.Context) (types.Info, error) {
			return types.Info{Swarm: swarm.Info{NodeID: selfNodeID}}, nil
		},
	}

	updateNodeFilter(context.Background(), client, filter)

	expected := filters.NewArgs(
		filters.Arg("node", "one"),
		filters.Arg("node", "two"),
		filters.Arg("node", selfNodeID),
	)
	assert.DeepEqual(t, expected, filter, cmpFilters)
}

var cmpFilters = cmp.AllowUnexported(filters.Args{})
