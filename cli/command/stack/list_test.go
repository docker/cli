package stack

import (
	"bytes"
	"io/ioutil"
	"testing"

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
			flags: map[string]string{
				"format": "{{invalid format}}",
			},
			expectedError: "Template parsing error",
		},
		{
			serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
				return []swarm.Service{}, errors.Errorf("error getting services")
			},
			expectedError: "error getting services",
		},
		{
			serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
				return []swarm.Service{*Service()}, nil
			},
			expectedError: "cannot get label",
		},
	}

	for _, tc := range testCases {
		cmd := newListCommand(test.NewFakeCli(&fakeClient{
			serviceListFunc: tc.serviceListFunc,
		}))
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		for key, value := range tc.flags {
			cmd.Flags().Set(key, value)
		}
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestListWithFormat(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := newListCommand(test.NewFakeCliWithOutput(&fakeClient{
		serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{
				*Service(
					ServiceLabels(map[string]string{
						"com.docker.stack.namespace": "service-name-foo",
					}),
				)}, nil
		},
	}, buf))
	cmd.Flags().Set("format", "{{ .Name }}")
	assert.NoError(t, cmd.Execute())
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "stack-list-with-format.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}

func TestListWithoutFormat(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := newListCommand(test.NewFakeCliWithOutput(&fakeClient{
		serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{
				*Service(
					ServiceLabels(map[string]string{
						"com.docker.stack.namespace": "service-name-foo",
					}),
				)}, nil
		},
	}, buf))
	assert.NoError(t, cmd.Execute())
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "stack-list-without-format.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}

func TestListOrder(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := newListCommand(test.NewFakeCliWithOutput(&fakeClient{
		serviceListFunc: func(options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{
				*Service(
					ServiceLabels(map[string]string{
						"com.docker.stack.namespace": "service-name-foo",
					}),
				),
				*Service(
					ServiceLabels(map[string]string{
						"com.docker.stack.namespace": "service-name-bar",
					}),
				),
			}, nil
		},
	}, buf))
	assert.NoError(t, cmd.Execute())
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "stack-list-sort.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}
