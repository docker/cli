package secret

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/pkg/errors"
)

const secretDataFile = "secret-create-with-name.golden"

func TestSecretCreateErrors(t *testing.T) {
	testCases := []struct {
		args             []string
		secretCreateFunc func(swarm.SecretSpec) (types.SecretCreateResponse, error)
		expectedError    string
	}{
		{args: []string{"too", "many", "arguments"},
			expectedError: "requires at least 1 and at most 2 arguments",
		},
		{args: []string{"create", "--driver", "driver", "-"},
			expectedError: "secret data must be empty",
		},
		{
			args: []string{"name", filepath.Join("testdata", secretDataFile)},
			secretCreateFunc: func(secretSpec swarm.SecretSpec) (types.SecretCreateResponse, error) {
				return types.SecretCreateResponse{}, errors.Errorf("error creating secret")
			},
			expectedError: "error creating secret",
		},
	}
	for _, tc := range testCases {
		cmd := newSecretCreateCommand(
			test.NewFakeCli(&fakeClient{
				secretCreateFunc: tc.secretCreateFunc,
			}),
		)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestSecretCreateWithName(t *testing.T) {
	name := "foo"
	data, err := ioutil.ReadFile(filepath.Join("testdata", secretDataFile))
	assert.NilError(t, err)

	expected := swarm.SecretSpec{
		Annotations: swarm.Annotations{
			Name:   name,
			Labels: make(map[string]string),
		},
		Data: data,
	}

	cli := test.NewFakeCli(&fakeClient{
		secretCreateFunc: func(spec swarm.SecretSpec) (types.SecretCreateResponse, error) {
			if !reflect.DeepEqual(spec, expected) {
				return types.SecretCreateResponse{}, errors.Errorf("expected %+v, got %+v", expected, spec)
			}
			return types.SecretCreateResponse{
				ID: "ID-" + spec.Name,
			}, nil
		},
	})

	cmd := newSecretCreateCommand(cli)
	cmd.SetArgs([]string{name, filepath.Join("testdata", secretDataFile)})
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("ID-"+name, strings.TrimSpace(cli.OutBuffer().String())))
}

func TestSecretCreateWithDriver(t *testing.T) {
	expectedDriver := &swarm.Driver{
		Name: "secret-driver",
	}
	name := "foo"

	cli := test.NewFakeCli(&fakeClient{
		secretCreateFunc: func(spec swarm.SecretSpec) (types.SecretCreateResponse, error) {
			if spec.Name != name {
				return types.SecretCreateResponse{}, errors.Errorf("expected name %q, got %q", name, spec.Name)
			}

			if spec.Driver.Name != expectedDriver.Name {
				return types.SecretCreateResponse{}, errors.Errorf("expected driver %v, got %v", expectedDriver, spec.Labels)
			}

			return types.SecretCreateResponse{
				ID: "ID-" + spec.Name,
			}, nil
		},
	})

	cmd := newSecretCreateCommand(cli)
	cmd.SetArgs([]string{name})
	cmd.Flags().Set("driver", expectedDriver.Name)
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("ID-"+name, strings.TrimSpace(cli.OutBuffer().String())))
}

func TestSecretCreateWithTemplatingDriver(t *testing.T) {
	expectedDriver := &swarm.Driver{
		Name: "template-driver",
	}
	name := "foo"

	cli := test.NewFakeCli(&fakeClient{
		secretCreateFunc: func(spec swarm.SecretSpec) (types.SecretCreateResponse, error) {
			if spec.Name != name {
				return types.SecretCreateResponse{}, errors.Errorf("expected name %q, got %q", name, spec.Name)
			}

			if spec.Templating.Name != expectedDriver.Name {
				return types.SecretCreateResponse{}, errors.Errorf("expected driver %v, got %v", expectedDriver, spec.Labels)
			}

			return types.SecretCreateResponse{
				ID: "ID-" + spec.Name,
			}, nil
		},
	})

	cmd := newSecretCreateCommand(cli)
	cmd.SetArgs([]string{name})
	cmd.Flags().Set("template-driver", expectedDriver.Name)
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("ID-"+name, strings.TrimSpace(cli.OutBuffer().String())))
}

func TestSecretCreateWithLabels(t *testing.T) {
	expectedLabels := map[string]string{
		"lbl1": "Label-foo",
		"lbl2": "Label-bar",
	}
	name := "foo"

	cli := test.NewFakeCli(&fakeClient{
		secretCreateFunc: func(spec swarm.SecretSpec) (types.SecretCreateResponse, error) {
			if spec.Name != name {
				return types.SecretCreateResponse{}, errors.Errorf("expected name %q, got %q", name, spec.Name)
			}

			if !reflect.DeepEqual(spec.Labels, expectedLabels) {
				return types.SecretCreateResponse{}, errors.Errorf("expected labels %v, got %v", expectedLabels, spec.Labels)
			}

			return types.SecretCreateResponse{
				ID: "ID-" + spec.Name,
			}, nil
		},
	})

	cmd := newSecretCreateCommand(cli)
	cmd.SetArgs([]string{name, filepath.Join("testdata", secretDataFile)})
	cmd.Flags().Set("label", "lbl1=Label-foo")
	cmd.Flags().Set("label", "lbl2=Label-bar")
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("ID-"+name, strings.TrimSpace(cli.OutBuffer().String())))
}
