package secret

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/docker/cli/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/pkg/testutil"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const secretDataFile = "secret-create-with-name.golden"

func TestSecretCreateErrors(t *testing.T) {
	testCases := []struct {
		args             []string
		secretCreateFunc func(swarm.SecretSpec) (types.SecretCreateResponse, error)
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
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestSecretCreateWithName(t *testing.T) {
	name := "foo"
	var actual []byte
	cli := test.NewFakeCli(&fakeClient{
		secretCreateFunc: func(spec swarm.SecretSpec) (types.SecretCreateResponse, error) {
			if spec.Name != name {
				return types.SecretCreateResponse{}, errors.Errorf("expected name %q, got %q", name, spec.Name)
			}

			actual = spec.Data

			return types.SecretCreateResponse{
				ID: "ID-" + spec.Name,
			}, nil
		},
	})

	cmd := newSecretCreateCommand(cli)
	cmd.SetArgs([]string{name, filepath.Join("testdata", secretDataFile)})
	assert.NoError(t, cmd.Execute())
	golden.Assert(t, string(actual), secretDataFile)
	assert.Equal(t, "ID-"+name, strings.TrimSpace(cli.OutBuffer().String()))
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
	assert.NoError(t, cmd.Execute())
	assert.Equal(t, "ID-"+name, strings.TrimSpace(cli.OutBuffer().String()))
}
