package secret

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/docker/cli/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/pkg/testutil"
	"github.com/docker/docker/pkg/testutil/golden"
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
			expectedError: "requires exactly 2 argument(s)",
		},
		{args: []string{"too", "many", "arguments"},
			expectedError: "requires exactly 2 argument(s)",
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
	buf := new(bytes.Buffer)
	var actual []byte
	cli := test.NewFakeCliWithOutput(&fakeClient{
		secretCreateFunc: func(spec swarm.SecretSpec) (types.SecretCreateResponse, error) {
			if spec.Name != name {
				return types.SecretCreateResponse{}, errors.Errorf("expected name %q, got %q", name, spec.Name)
			}

			actual = spec.Data

			return types.SecretCreateResponse{
				ID: "ID-" + spec.Name,
			}, nil
		},
	}, buf)

	cmd := newSecretCreateCommand(cli)
	cmd.SetArgs([]string{name, filepath.Join("testdata", secretDataFile)})
	assert.NoError(t, cmd.Execute())
	expected := golden.Get(t, actual, secretDataFile)
	assert.Equal(t, expected, actual)
	assert.Equal(t, "ID-"+name, strings.TrimSpace(buf.String()))
}

func TestSecretCreateWithLabels(t *testing.T) {
	expectedLabels := map[string]string{
		"lbl1": "Label-foo",
		"lbl2": "Label-bar",
	}
	name := "foo"

	buf := new(bytes.Buffer)
	cli := test.NewFakeCliWithOutput(&fakeClient{
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
	}, buf)

	cmd := newSecretCreateCommand(cli)
	cmd.SetArgs([]string{name, filepath.Join("testdata", secretDataFile)})
	cmd.Flags().Set("label", "lbl1=Label-foo")
	cmd.Flags().Set("label", "lbl2=Label-bar")
	assert.NoError(t, cmd.Execute())
	assert.Equal(t, "ID-"+name, strings.TrimSpace(buf.String()))
}
