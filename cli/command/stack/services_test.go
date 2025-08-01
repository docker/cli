package stack

import (
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/moby/moby/api/types/swarm"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestStackServicesErrors(t *testing.T) {
	testCases := []struct {
		args            []string
		flags           map[string]string
		serviceListFunc func(options swarm.ServiceListOptions) ([]swarm.Service, error)
		nodeListFunc    func(options swarm.NodeListOptions) ([]swarm.Node, error)
		taskListFunc    func(options swarm.TaskListOptions) ([]swarm.Task, error)
		expectedError   string
	}{
		{
			args: []string{"foo"},
			serviceListFunc: func(options swarm.ServiceListOptions) ([]swarm.Service, error) {
				return nil, errors.New("error getting services")
			},
			expectedError: "error getting services",
		},
		{
			args: []string{"foo"},
			serviceListFunc: func(options swarm.ServiceListOptions) ([]swarm.Service, error) {
				return []swarm.Service{*builders.Service(builders.GlobalService())}, nil
			},
			nodeListFunc: func(options swarm.NodeListOptions) ([]swarm.Node, error) {
				return nil, errors.New("error getting nodes")
			},
			taskListFunc: func(options swarm.TaskListOptions) ([]swarm.Task, error) {
				return []swarm.Task{*builders.Task()}, nil
			},
			expectedError: "error getting nodes",
		},
		{
			args: []string{"foo"},
			serviceListFunc: func(options swarm.ServiceListOptions) ([]swarm.Service, error) {
				return []swarm.Service{*builders.Service(builders.GlobalService())}, nil
			},
			taskListFunc: func(options swarm.TaskListOptions) ([]swarm.Task, error) {
				return nil, errors.New("error getting tasks")
			},
			expectedError: "error getting tasks",
		},
		{
			args: []string{"foo"},
			flags: map[string]string{
				"format": "{{invalid format}}",
			},
			serviceListFunc: func(options swarm.ServiceListOptions) ([]swarm.Service, error) {
				return []swarm.Service{*builders.Service()}, nil
			},
			expectedError: "template parsing error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.expectedError, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				serviceListFunc: tc.serviceListFunc,
				nodeListFunc:    tc.nodeListFunc,
				taskListFunc:    tc.taskListFunc,
			})
			cmd := newServicesCommand(cli)
			cmd.SetArgs(tc.args)
			for key, value := range tc.flags {
				assert.Check(t, cmd.Flags().Set(key, value))
			}
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestRunServicesWithEmptyName(t *testing.T) {
	cmd := newServicesCommand(test.NewFakeCli(&fakeClient{}))
	cmd.SetArgs([]string{"'   '"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	assert.ErrorContains(t, cmd.Execute(), `invalid stack name: "'   '"`)
}

func TestStackServicesEmptyServiceList(t *testing.T) {
	fakeCli := test.NewFakeCli(&fakeClient{
		serviceListFunc: func(options swarm.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{}, nil
		},
	})
	cmd := newServicesCommand(fakeCli)
	cmd.SetArgs([]string{"foo"})
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("", fakeCli.OutBuffer().String()))
	assert.Check(t, is.Equal("Nothing found in stack: foo\n", fakeCli.ErrBuffer().String()))
}

func TestStackServicesWithQuietOption(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		serviceListFunc: func(options swarm.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{*builders.Service(builders.ServiceID("id-foo"))}, nil
		},
	})
	cmd := newServicesCommand(cli)
	assert.Check(t, cmd.Flags().Set("quiet", "true"))
	cmd.SetArgs([]string{"foo"})
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "stack-services-with-quiet-option.golden")
}

func TestStackServicesWithFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		serviceListFunc: func(options swarm.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{
				*builders.Service(builders.ServiceName("service-name-foo")),
			}, nil
		},
	})
	cmd := newServicesCommand(cli)
	cmd.SetArgs([]string{"foo"})
	assert.Check(t, cmd.Flags().Set("format", "{{ .Name }}"))
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "stack-services-with-format.golden")
}

func TestStackServicesWithConfigFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		serviceListFunc: func(options swarm.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{
				*builders.Service(builders.ServiceName("service-name-foo")),
			}, nil
		},
	})
	cli.SetConfigFile(&configfile.ConfigFile{
		ServicesFormat: "{{ .Name }}",
	})
	cmd := newServicesCommand(cli)
	cmd.SetArgs([]string{"foo"})
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "stack-services-with-config-format.golden")
}

func TestStackServicesWithoutFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		serviceListFunc: func(options swarm.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{*builders.Service(
				builders.ServiceName("name-foo"),
				builders.ServiceID("id-foo"),
				builders.ReplicatedService(2),
				builders.ServiceImage("busybox:latest"),
				builders.ServicePort(swarm.PortConfig{
					PublishMode:   swarm.PortConfigPublishModeIngress,
					PublishedPort: 0,
					TargetPort:    3232,
					Protocol:      swarm.PortConfigProtocolTCP,
				}),
			)}, nil
		},
	})
	cmd := newServicesCommand(cli)
	cmd.SetArgs([]string{"foo"})
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "stack-services-without-format.golden")
}
