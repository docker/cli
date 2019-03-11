package stack

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	composetypes "github.com/docker/stacks/pkg/compose/types"
	stacktypes "github.com/docker/stacks/pkg/types"
	// Import builders to get the builder function as package function
	. "github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/pkg/errors"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/golden"
)

func TestStackServicesErrors(t *testing.T) {
	testCases := []struct {
		args            []string
		flags           map[string]string
		serviceListFunc func(options types.ServiceListOptions) ([]swarm.Service, error)
		nodeListFunc    func(options types.NodeListOptions) ([]swarm.Node, error)
		taskListFunc    func(options types.TaskListOptions) ([]swarm.Task, error)
		expectedError   string
	}{
		{
			args: []string{"foo"},
			serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
				return nil, errors.Errorf("error getting services")
			},
			expectedError: "error getting services",
		},
		{
			args: []string{"foo"},
			serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
				return []swarm.Service{*Service()}, nil
			},
			nodeListFunc: func(options types.NodeListOptions) ([]swarm.Node, error) {
				return nil, errors.Errorf("error getting nodes")
			},
			expectedError: "error getting nodes",
		},
		{
			args: []string{"foo"},
			serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
				return []swarm.Service{*Service()}, nil
			},
			taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
				return nil, errors.Errorf("error getting tasks")
			},
			expectedError: "error getting tasks",
		},
		{
			args: []string{"foo"},
			flags: map[string]string{
				"format": "{{invalid format}}",
			},
			serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
				return []swarm.Service{*Service()}, nil
			},
			expectedError: "Template parsing error",
		},
	}

	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			serviceListFunc: tc.serviceListFunc,
			nodeListFunc:    tc.nodeListFunc,
			taskListFunc:    tc.taskListFunc,
		})
		cmd := newServicesCommand(cli, &orchestrator)
		cmd.SetArgs(tc.args)
		for key, value := range tc.flags {
			cmd.Flags().Set(key, value)
		}
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestRunServicesWithEmptyName(t *testing.T) {
	cmd := newServicesCommand(test.NewFakeCli(&fakeClient{}), &orchestrator)
	cmd.SetArgs([]string{"'   '"})
	cmd.SetOutput(ioutil.Discard)

	assert.ErrorContains(t, cmd.Execute(), `invalid stack name: "'   '"`)
}

func TestStackServicesEmptyServiceList(t *testing.T) {
	fakeCli := test.NewFakeCli(&fakeClient{
		serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{}, nil
		},
	})
	cmd := newServicesCommand(fakeCli, &orchestrator)
	cmd.SetArgs([]string{"foo"})
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("", fakeCli.OutBuffer().String()))
	assert.Check(t, is.Equal("Nothing found in stack: foo\n", fakeCli.ErrBuffer().String()))
}

func TestStackServicesWithQuietOption(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{*Service(ServiceID("id-foo"))}, nil
		},
	})
	cmd := newServicesCommand(cli, &orchestrator)
	cmd.Flags().Set("quiet", "true")
	cmd.SetArgs([]string{"foo"})
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "stack-services-with-quiet-option.golden")
}

func TestStackServicesWithFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{
				*Service(ServiceName("service-name-foo")),
			}, nil
		},
	})
	cmd := newServicesCommand(cli, &orchestrator)
	cmd.SetArgs([]string{"foo"})
	cmd.Flags().Set("format", "{{ .Name }}")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "stack-services-with-format.golden")
}

func TestStackServicesWithConfigFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{
				*Service(ServiceName("service-name-foo")),
			}, nil
		},
	})
	cli.SetConfigFile(&configfile.ConfigFile{
		ServicesFormat: "{{ .Name }}",
	})
	cmd := newServicesCommand(cli, &orchestrator)
	cmd.SetArgs([]string{"foo"})
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "stack-services-with-config-format.golden")
}

func TestStackServicesWithoutFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{*Service(
				ServiceName("name-foo"),
				ServiceID("id-foo"),
				ReplicatedService(2),
				ServiceImage("busybox:latest"),
				ServicePort(swarm.PortConfig{
					PublishMode:   swarm.PortConfigPublishModeIngress,
					PublishedPort: 0,
					TargetPort:    3232,
					Protocol:      swarm.PortConfigProtocolTCP,
				}),
			)}, nil
		},
	})
	cmd := newServicesCommand(cli, &orchestrator)
	cmd.SetArgs([]string{"foo"})
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "stack-services-without-format.golden")
}

func TestStackServicesServerSideListFailure(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		version: clientSideStackVersion,
		stackListFunc: func(options stacktypes.StackListOptions) ([]stacktypes.Stack, error) {
			return nil, fmt.Errorf("failed to list stacks")
		},
	})
	cmd := newServicesCommand(cli, &orchestrator)
	cmd.SetArgs([]string{"stackname"})
	cmd.SetOutput(ioutil.Discard)
	assert.ErrorContains(t, cmd.Execute(), `failed to list stacks`)
}

func TestStackServicesServerSideMissingResources(t *testing.T) {
	stacks := []stacktypes.Stack{
		{
			Orchestrator: "swarm",
			Spec: stacktypes.StackSpec{
				Metadata: stacktypes.Metadata{
					Name: "stackname",
				},
				Collection: "collection",
			},
		},
	}
	cli := test.NewFakeCli(&fakeClient{
		version: clientSideStackVersion,
		stackListFunc: func(options stacktypes.StackListOptions) ([]stacktypes.Stack, error) {
			return stacks, nil
		},
	})
	cmd := newServicesCommand(cli, &orchestrator)
	cmd.SetArgs([]string{"stackname"})
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Contains(cli.ErrBuffer().String(), `no services found in stack`))
}

func TestStackServicesServerSideMissingStatus(t *testing.T) {
	stacks := []stacktypes.Stack{
		{
			Orchestrator: "swarm",
			Spec: stacktypes.StackSpec{
				Metadata: stacktypes.Metadata{
					Name: "stackname",
				},
				Collection: "collection",
				Services: composetypes.Services{
					composetypes.ServiceConfig{
						Name: "svc1",
					},
				},
			},
			StackResources: stacktypes.StackResources{
				Services: map[string]stacktypes.StackResource{
					"svc1": {
						Orchestrator: "swarm",
						Kind:         "service",
						ID:           "id",
					},
				},
			},
		},
	}
	cli := test.NewFakeCli(&fakeClient{
		version: clientSideStackVersion,
		stackListFunc: func(options stacktypes.StackListOptions) ([]stacktypes.Stack, error) {
			return stacks, nil
		},
	})
	cmd := newServicesCommand(cli, &orchestrator)
	cmd.SetOutput(ioutil.Discard)
	cmd.SetArgs([]string{"stackname"})
	assert.ErrorContains(t, cmd.Execute(), `unable to find stack status for service`)
}

func TestStackServicesServerSideSuccess(t *testing.T) {
	stacks := []stacktypes.Stack{
		{
			Orchestrator: "swarm",
			Spec: stacktypes.StackSpec{
				Metadata: stacktypes.Metadata{
					Name: "stackname",
				},
				Collection: "collection",
				Services: composetypes.Services{
					composetypes.ServiceConfig{
						Name: "svc1",
					},
				},
			},
			StackResources: stacktypes.StackResources{
				Services: map[string]stacktypes.StackResource{
					"svc1": {
						Orchestrator: "swarm",
						Kind:         "service",
						ID:           "id",
					},
				},
			},
			Status: stacktypes.StackStatus{
				ServicesStatus: map[string]stacktypes.ServiceStatus{
					"svc1": {
						DesiredTasks: 1,
						RunningTasks: 1,
					},
				},
			},
		},
	}
	cli := test.NewFakeCli(&fakeClient{
		version: clientSideStackVersion,
		stackListFunc: func(options stacktypes.StackListOptions) ([]stacktypes.Stack, error) {
			return stacks, nil
		},
	})
	cmd := newServicesCommand(cli, &orchestrator)
	cmd.SetOutput(ioutil.Discard)
	cmd.SetArgs([]string{"stackname"})
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "stack-services-server-side.golden")
}

func TestStackServicesServerSideFilters(t *testing.T) {
	stacks := []stacktypes.Stack{
		{
			Orchestrator: "swarm",
			Spec: stacktypes.StackSpec{
				Metadata: stacktypes.Metadata{
					Name: "stackname",
				},
				Collection: "collection",
				Services: composetypes.Services{
					composetypes.ServiceConfig{
						Name: "svc1",
					},
					composetypes.ServiceConfig{
						Name: "svc2",
					},
					composetypes.ServiceConfig{
						Name: "svc3",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
					composetypes.ServiceConfig{
						Name: "svc4",
						Deploy: composetypes.DeployConfig{
							Mode: "global",
						},
					},
					composetypes.ServiceConfig{
						Name: "svc5",
					},
				},
			},
			StackResources: stacktypes.StackResources{
				Services: map[string]stacktypes.StackResource{
					"svc1": {
						Orchestrator: "swarm",
						Kind:         "service",
						ID:           "id1",
					},
					"svc2": {
						Orchestrator: "swarm",
						Kind:         "service",
						ID:           "id2",
					},
					"svc3": {
						Orchestrator: "swarm",
						Kind:         "service",
						ID:           "id3",
					},
					"svc4": {
						Orchestrator: "swarm",
						Kind:         "service",
						ID:           "id4",
					},
					"svc5": {
						Orchestrator: "swarm",
						Kind:         "service",
						ID:           "id5",
					},
				},
			},
			Status: stacktypes.StackStatus{
				ServicesStatus: map[string]stacktypes.ServiceStatus{
					"svc1": {
						DesiredTasks: 1,
						RunningTasks: 1,
					},
					"svc2": {
						DesiredTasks: 2,
						RunningTasks: 2,
					},
					"svc3": {
						DesiredTasks: 3,
						RunningTasks: 3,
					},
					"svc4": {
						DesiredTasks: 4,
						RunningTasks: 4,
					},
					"svc5": {
						DesiredTasks: 5,
						RunningTasks: 5,
					},
				},
			},
		},
	}
	cli := test.NewFakeCli(&fakeClient{
		version: clientSideStackVersion,
		stackListFunc: func(options stacktypes.StackListOptions) ([]stacktypes.Stack, error) {
			return stacks, nil
		},
	})
	cmd := newServicesCommand(cli, &orchestrator)
	cmd.SetOutput(ioutil.Discard)
	cmd.SetArgs([]string{"stackname"})
	cmd.Flags().Set("filter", "name=svc1")
	cmd.Flags().Set("filter", "id=id2")
	cmd.Flags().Set("filter", "label=foo=bar")
	cmd.Flags().Set("filter", "mode=global")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "stack-services-server-side-filtered.golden")
}
