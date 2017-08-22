package config

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const configDataFile = "config-create-with-name.golden"

func TestConfigCreateErrors(t *testing.T) {
	testCases := []struct {
		args             []string
		configCreateFunc func(swarm.ConfigSpec) (types.ConfigCreateResponse, error)
		expectedError    string
	}{
		{
			args:          []string{"too_few"},
			expectedError: "requires exactly 2 arguments",
		},
		{args: []string{"too", "many", "arguments"},
			expectedError: "requires exactly 2 arguments",
		},
		{
			args: []string{"name", filepath.Join("testdata", configDataFile)},
			configCreateFunc: func(configSpec swarm.ConfigSpec) (types.ConfigCreateResponse, error) {
				return types.ConfigCreateResponse{}, errors.Errorf("error creating config")
			},
			expectedError: "error creating config",
		},
	}
	for _, tc := range testCases {
		cmd := newConfigCreateCommand(
			test.NewFakeCli(&fakeClient{
				configCreateFunc: tc.configCreateFunc,
			}),
		)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestConfigCreateWithName(t *testing.T) {
	name := "foo"
	var actual []byte
	cli := test.NewFakeCli(&fakeClient{
		configCreateFunc: func(spec swarm.ConfigSpec) (types.ConfigCreateResponse, error) {
			if spec.Name != name {
				return types.ConfigCreateResponse{}, errors.Errorf("expected name %q, got %q", name, spec.Name)
			}

			actual = spec.Data

			return types.ConfigCreateResponse{
				ID: "ID-" + spec.Name,
			}, nil
		},
	})

	cmd := newConfigCreateCommand(cli)
	cmd.SetArgs([]string{name, filepath.Join("testdata", configDataFile)})
	assert.NoError(t, cmd.Execute())
	golden.Assert(t, string(actual), configDataFile)
	assert.Equal(t, "ID-"+name, strings.TrimSpace(cli.OutBuffer().String()))
}

func TestConfigCreateWithLabels(t *testing.T) {
	expectedLabels := map[string]string{
		"lbl1": "Label-foo",
		"lbl2": "Label-bar",
	}
	name := "foo"

	cli := test.NewFakeCli(&fakeClient{
		configCreateFunc: func(spec swarm.ConfigSpec) (types.ConfigCreateResponse, error) {
			if spec.Name != name {
				return types.ConfigCreateResponse{}, errors.Errorf("expected name %q, got %q", name, spec.Name)
			}

			if !reflect.DeepEqual(spec.Labels, expectedLabels) {
				return types.ConfigCreateResponse{}, errors.Errorf("expected labels %v, got %v", expectedLabels, spec.Labels)
			}

			return types.ConfigCreateResponse{
				ID: "ID-" + spec.Name,
			}, nil
		},
	})

	cmd := newConfigCreateCommand(cli)
	cmd.SetArgs([]string{name, filepath.Join("testdata", configDataFile)})
	cmd.Flags().Set("label", "lbl1=Label-foo")
	cmd.Flags().Set("label", "lbl2=Label-bar")
	assert.NoError(t, cmd.Execute())
	assert.Equal(t, "ID-"+name, strings.TrimSpace(cli.OutBuffer().String()))
}
