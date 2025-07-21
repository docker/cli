package secret

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
)

const secretDataFile = "secret-create-with-name.golden"

func TestSecretCreateErrors(t *testing.T) {
	testCases := []struct {
		args             []string
		secretCreateFunc func(context.Context, swarm.SecretSpec) (swarm.SecretCreateResponse, error)
		expectedError    string
	}{
		{
			args:          []string{"too", "many", "arguments"},
			expectedError: "requires at least 1 and at most 2 arguments",
		},
		{
			args:          []string{"create", "--driver", "driver", "-"},
			expectedError: "secret data must be empty",
		},
		{
			args: []string{"name", filepath.Join("testdata", secretDataFile)},
			secretCreateFunc: func(_ context.Context, secretSpec swarm.SecretSpec) (swarm.SecretCreateResponse, error) {
				return swarm.SecretCreateResponse{}, errors.New("error creating secret")
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
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestSecretCreateWithName(t *testing.T) {
	const name = "secret-with-name"
	data, err := os.ReadFile(filepath.Join("testdata", secretDataFile))
	assert.NilError(t, err)

	expected := swarm.SecretSpec{
		Annotations: swarm.Annotations{
			Name:   name,
			Labels: make(map[string]string),
		},
		Data: data,
	}

	cli := test.NewFakeCli(&fakeClient{
		secretCreateFunc: func(_ context.Context, spec swarm.SecretSpec) (swarm.SecretCreateResponse, error) {
			if !reflect.DeepEqual(spec, expected) {
				return swarm.SecretCreateResponse{}, fmt.Errorf("expected %+v, got %+v", expected, spec)
			}
			return swarm.SecretCreateResponse{
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
	const name = "secret-with-driver"

	cli := test.NewFakeCli(&fakeClient{
		secretCreateFunc: func(_ context.Context, spec swarm.SecretSpec) (swarm.SecretCreateResponse, error) {
			if spec.Name != name {
				return swarm.SecretCreateResponse{}, fmt.Errorf("expected name %q, got %q", name, spec.Name)
			}

			if spec.Driver.Name != expectedDriver.Name {
				return swarm.SecretCreateResponse{}, fmt.Errorf("expected driver %v, got %v", expectedDriver, spec.Labels)
			}

			return swarm.SecretCreateResponse{
				ID: "ID-" + spec.Name,
			}, nil
		},
	})

	cmd := newSecretCreateCommand(cli)
	cmd.SetArgs([]string{name})
	assert.Check(t, cmd.Flags().Set("driver", expectedDriver.Name))
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("ID-"+name, strings.TrimSpace(cli.OutBuffer().String())))
}

func TestSecretCreateWithTemplatingDriver(t *testing.T) {
	expectedDriver := &swarm.Driver{
		Name: "template-driver",
	}
	const name = "secret-with-template-driver"

	cli := test.NewFakeCli(&fakeClient{
		secretCreateFunc: func(_ context.Context, spec swarm.SecretSpec) (swarm.SecretCreateResponse, error) {
			if spec.Name != name {
				return swarm.SecretCreateResponse{}, fmt.Errorf("expected name %q, got %q", name, spec.Name)
			}

			if spec.Templating.Name != expectedDriver.Name {
				return swarm.SecretCreateResponse{}, fmt.Errorf("expected driver %v, got %v", expectedDriver, spec.Labels)
			}

			return swarm.SecretCreateResponse{
				ID: "ID-" + spec.Name,
			}, nil
		},
	})

	cmd := newSecretCreateCommand(cli)
	cmd.SetArgs([]string{name, filepath.Join("testdata", secretDataFile)})
	assert.Check(t, cmd.Flags().Set("template-driver", expectedDriver.Name))
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("ID-"+name, strings.TrimSpace(cli.OutBuffer().String())))
}

func TestSecretCreateWithLabels(t *testing.T) {
	expectedLabels := map[string]string{
		"lbl1": "Label-foo",
		"lbl2": "Label-bar",
	}
	const name = "secret-with-labels"

	cli := test.NewFakeCli(&fakeClient{
		secretCreateFunc: func(_ context.Context, spec swarm.SecretSpec) (swarm.SecretCreateResponse, error) {
			if spec.Name != name {
				return swarm.SecretCreateResponse{}, fmt.Errorf("expected name %q, got %q", name, spec.Name)
			}

			if !reflect.DeepEqual(spec.Labels, expectedLabels) {
				return swarm.SecretCreateResponse{}, fmt.Errorf("expected labels %v, got %v", expectedLabels, spec.Labels)
			}

			return swarm.SecretCreateResponse{
				ID: "ID-" + spec.Name,
			}, nil
		},
	})

	cmd := newSecretCreateCommand(cli)
	cmd.SetArgs([]string{name, filepath.Join("testdata", secretDataFile)})
	assert.Check(t, cmd.Flags().Set("label", "lbl1=Label-foo"))
	assert.Check(t, cmd.Flags().Set("label", "lbl2=Label-bar"))
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("ID-"+name, strings.TrimSpace(cli.OutBuffer().String())))
}
