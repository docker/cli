package service

import (
	"testing"

	"bytes"

	"github.com/docker/cli/cli/internal/test"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

type fakeClient struct {
	client.Client
	serviceListFunc func(context.Context, types.ServiceListOptions) ([]swarm.Service, error)
}

func (f *fakeClient) ServiceList(ctx context.Context, options types.ServiceListOptions) ([]swarm.Service, error) {
	if f.serviceListFunc != nil {
		return f.serviceListFunc(ctx, options)
	}
	return nil, nil
}

func (f *fakeClient) TaskList(ctx context.Context, options types.TaskListOptions) ([]swarm.Task, error) {
	return nil, nil
}

func newService(id string, name string) swarm.Service {
	return swarm.Service{
		ID:   id,
		Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: name}},
	}
}

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
	require.NoError(t, filter.Set("node=somenode"))
	options := psOptions{
		services: []string{"idmatch", "idprefix", "namematch", "notfound"},
		filter:   filter,
	}

	actual, notfound, err := createFilter(context.Background(), client, options)
	require.NoError(t, err)
	assert.Equal(t, notfound, []string{"no such service: notfound"})

	expected := filters.NewArgs()
	expected.Add("service", "idmatch")
	expected.Add("service", "idprefixmatch")
	expected.Add("service", "cccccccc")
	expected.Add("node", "somenode")
	assert.Equal(t, expected, actual)
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
	assert.EqualError(t, err, "multiple services found with provided prefix: aaa")
}

func TestCreateFilterNoneFound(t *testing.T) {
	client := &fakeClient{}
	options := psOptions{
		services: []string{"foo", "notfound"},
		filter:   opts.NewFilterOpt(),
	}
	_, _, err := createFilter(context.Background(), client, options)
	assert.EqualError(t, err, "no such service: foo\nno such service: notfound")
}

func TestRunPSWarnsOnNotFound(t *testing.T) {
	client := &fakeClient{
		serviceListFunc: func(ctx context.Context, options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{
				{ID: "foo"},
			}, nil
		},
	}

	out := new(bytes.Buffer)
	cli := test.NewFakeCliWithOutput(client, out)
	options := psOptions{
		services: []string{"foo", "bar"},
		filter:   opts.NewFilterOpt(),
		format:   "{{.ID}}",
	}
	err := runPS(cli, options)
	assert.EqualError(t, err, "no such service: bar")
}
