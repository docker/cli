package stack

import (
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestListErrors(t *testing.T) {
	testCases := []struct {
		args            []string
		flags           map[string]string
		serviceListFunc func(options types.ServiceListOptions) ([]swarm.Service, error)
		expectedError   string
	}{
		{
			args:          []string{"foo"},
			expectedError: "accepts no argument",
		},
		{
			args: []string{},
			flags: map[string]string{
				"format": "{{invalid format}}",
			},
			expectedError: "template parsing error",
		},
		{
			args: []string{},
			serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
				return []swarm.Service{}, errors.Errorf("error getting services")
			},
			expectedError: "error getting services",
		},
		{
			args: []string{},
			serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
				return []swarm.Service{*builders.Service()}, nil
			},
			expectedError: "cannot get label",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.expectedError, func(t *testing.T) {
			cmd := newListCommand(test.NewFakeCli(&fakeClient{
				serviceListFunc: tc.serviceListFunc,
			}))
			cmd.SetArgs(tc.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			for key, value := range tc.flags {
				assert.Check(t, cmd.Flags().Set(key, value))
			}
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestStackList(t *testing.T) {
	testCases := []struct {
		doc          string
		serviceNames []string
		flags        map[string]string
		golden       string
	}{
		{
			doc:          "WithFormat",
			serviceNames: []string{"service-name-foo"},
			flags: map[string]string{
				"format": "{{ .Name }}",
			},
			golden: "stack-list-with-format.golden",
		},
		{
			doc:          "WithoutFormat",
			serviceNames: []string{"service-name-foo"},
			golden:       "stack-list-without-format.golden",
		},
		{
			doc: "Sort",
			serviceNames: []string{
				"service-name-foo",
				"service-name-bar",
			},
			golden: "stack-list-sort.golden",
		},
		{
			doc: "SortNatural",
			serviceNames: []string{
				"service-name-1-foo",
				"service-name-10-foo",
				"service-name-2-foo",
			},
			golden: "stack-list-sort-natural.golden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			var services []swarm.Service
			for _, name := range tc.serviceNames {
				services = append(services,
					*builders.Service(
						builders.ServiceLabels(map[string]string{
							"com.docker.stack.namespace": name,
						}),
					),
				)
			}
			cli := test.NewFakeCli(&fakeClient{
				serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
					return services, nil
				},
			})
			cmd := newListCommand(cli)
			for key, value := range tc.flags {
				assert.Check(t, cmd.Flags().Set(key, value))
			}
			assert.NilError(t, cmd.Execute())
			golden.Assert(t, cli.OutBuffer().String(), tc.golden)
		})
	}
}
