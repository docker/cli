package config

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/swarm"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

const configDataFile = "config-create-with-name.golden"

func TestConfigCreateErrors(t *testing.T) {
	testCases := []struct {
		args             []string
		configCreateFunc func(context.Context, swarm.ConfigSpec) (swarm.ConfigCreateResponse, error)
		expectedError    string
	}{
		{
			args:          []string{"too_few"},
			expectedError: "requires 2 arguments",
		},
		{
			args:          []string{"too", "many", "arguments"},
			expectedError: "requires 2 arguments",
		},
		{
			args: []string{"name", filepath.Join("testdata", configDataFile)},
			configCreateFunc: func(_ context.Context, configSpec swarm.ConfigSpec) (swarm.ConfigCreateResponse, error) {
				return swarm.ConfigCreateResponse{}, errors.New("error creating config")
			},
			expectedError: "error creating config",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.expectedError, func(t *testing.T) {
			cmd := newConfigCreateCommand(
				test.NewFakeCli(&fakeClient{
					configCreateFunc: tc.configCreateFunc,
				}),
			)
			cmd.SetArgs(tc.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestConfigCreateWithName(t *testing.T) {
	const name = "config-with-name"
	var actual []byte
	cli := test.NewFakeCli(&fakeClient{
		configCreateFunc: func(_ context.Context, spec swarm.ConfigSpec) (swarm.ConfigCreateResponse, error) {
			if spec.Name != name {
				return swarm.ConfigCreateResponse{}, fmt.Errorf("expected name %q, got %q", name, spec.Name)
			}

			actual = spec.Data

			return swarm.ConfigCreateResponse{
				ID: "ID-" + spec.Name,
			}, nil
		},
	})

	cmd := newConfigCreateCommand(cli)
	cmd.SetArgs([]string{name, filepath.Join("testdata", configDataFile)})
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, string(actual), configDataFile)
	assert.Check(t, is.Equal("ID-"+name, strings.TrimSpace(cli.OutBuffer().String())))
}

func TestConfigCreateWithLabels(t *testing.T) {
	expectedLabels := map[string]string{
		"lbl1": "Label-foo",
		"lbl2": "Label-bar",
	}
	const name = "config-with-labels"

	data, err := os.ReadFile(filepath.Join("testdata", configDataFile))
	assert.NilError(t, err)

	expected := swarm.ConfigSpec{
		Annotations: swarm.Annotations{
			Name:   name,
			Labels: expectedLabels,
		},
		Data: data,
	}

	cli := test.NewFakeCli(&fakeClient{
		configCreateFunc: func(_ context.Context, spec swarm.ConfigSpec) (swarm.ConfigCreateResponse, error) {
			if !reflect.DeepEqual(spec, expected) {
				return swarm.ConfigCreateResponse{}, fmt.Errorf("expected %+v, got %+v", expected, spec)
			}

			return swarm.ConfigCreateResponse{
				ID: "ID-" + spec.Name,
			}, nil
		},
	})

	cmd := newConfigCreateCommand(cli)
	cmd.SetArgs([]string{name, filepath.Join("testdata", configDataFile)})
	cmd.Flags().Set("label", "lbl1=Label-foo")
	cmd.Flags().Set("label", "lbl2=Label-bar")
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("ID-"+name, strings.TrimSpace(cli.OutBuffer().String())))
}

func TestConfigCreateWithTemplatingDriver(t *testing.T) {
	expectedDriver := &swarm.Driver{
		Name: "template-driver",
	}
	const name = "config-with-template-driver"

	cli := test.NewFakeCli(&fakeClient{
		configCreateFunc: func(_ context.Context, spec swarm.ConfigSpec) (swarm.ConfigCreateResponse, error) {
			if spec.Name != name {
				return swarm.ConfigCreateResponse{}, fmt.Errorf("expected name %q, got %q", name, spec.Name)
			}

			if spec.Templating.Name != expectedDriver.Name {
				return swarm.ConfigCreateResponse{}, fmt.Errorf("expected driver %v, got %v", expectedDriver, spec.Labels)
			}

			return swarm.ConfigCreateResponse{
				ID: "ID-" + spec.Name,
			}, nil
		},
	})

	cmd := newConfigCreateCommand(cli)
	cmd.SetArgs([]string{name, filepath.Join("testdata", configDataFile)})
	cmd.Flags().Set("template-driver", expectedDriver.Name)
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("ID-"+name, strings.TrimSpace(cli.OutBuffer().String())))
}
