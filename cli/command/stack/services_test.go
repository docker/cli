package stack

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/internal/test"
	// Import builders to get the builder function as package function
	. "github.com/docker/cli/cli/internal/test/builders"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/pkg/testutil"
	"github.com/docker/docker/pkg/testutil/golden"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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
		}, &bytes.Buffer{})
		cli.SetConfigfile(&configfile.ConfigFile{})
		cmd := newServicesCommand(cli)
		cmd.SetArgs(tc.args)
		for key, value := range tc.flags {
			cmd.Flags().Set(key, value)
		}
		cmd.SetOutput(ioutil.Discard)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestStackServicesEmptyServiceList(t *testing.T) {
	out := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	fakeCli := test.NewFakeCli(&fakeClient{
		serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{}, nil
		},
	}, out)
	fakeCli.SetErr(stderr)
	cmd := newServicesCommand(fakeCli)
	cmd.SetArgs([]string{"foo"})
	assert.NoError(t, cmd.Execute())
	assert.Equal(t, "", out.String())
	assert.Equal(t, "Nothing found in stack: foo\n", stderr.String())
}

func TestStackServicesWithQuietOption(t *testing.T) {
	buf := new(bytes.Buffer)
	cli := test.NewFakeCli(&fakeClient{
		serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{*Service(ServiceID("id-foo"))}, nil
		},
	}, buf)
	cli.SetConfigfile(&configfile.ConfigFile{})
	cmd := newServicesCommand(cli)
	cmd.Flags().Set("quiet", "true")
	cmd.SetArgs([]string{"foo"})
	assert.NoError(t, cmd.Execute())
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "stack-services-with-quiet-option.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}

func TestStackServicesWithFormat(t *testing.T) {
	buf := new(bytes.Buffer)
	cli := test.NewFakeCli(&fakeClient{
		serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{
				*Service(ServiceName("service-name-foo")),
			}, nil
		},
	}, buf)
	cli.SetConfigfile(&configfile.ConfigFile{})
	cmd := newServicesCommand(cli)
	cmd.SetArgs([]string{"foo"})
	cmd.Flags().Set("format", "{{ .Name }}")
	assert.NoError(t, cmd.Execute())
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "stack-services-with-format.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}

func TestStackServicesWithConfigFormat(t *testing.T) {
	buf := new(bytes.Buffer)
	cli := test.NewFakeCli(&fakeClient{
		serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{
				*Service(ServiceName("service-name-foo")),
			}, nil
		},
	}, buf)
	cli.SetConfigfile(&configfile.ConfigFile{
		ServicesFormat: "{{ .Name }}",
	})
	cmd := newServicesCommand(cli)
	cmd.SetArgs([]string{"foo"})
	assert.NoError(t, cmd.Execute())
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "stack-services-with-config-format.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}

func TestStackServicesWithoutFormat(t *testing.T) {
	buf := new(bytes.Buffer)
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
	}, buf)
	cli.SetConfigfile(&configfile.ConfigFile{})
	cmd := newServicesCommand(cli)
	cmd.SetArgs([]string{"foo"})
	assert.NoError(t, cmd.Execute())
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "stack-services-without-format.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}
