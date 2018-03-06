package secret

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/pkg/errors"
)

func TestSecretRemoveErrors(t *testing.T) {
	testCases := []struct {
		args             []string
		secretRemoveFunc func(string) error
		expectedError    string
	}{
		{
			args:          []string{},
			expectedError: "requires at least 1 argument.",
		},
		{
			args: []string{"foo"},
			secretRemoveFunc: func(name string) error {
				return errors.Errorf("error removing secret")
			},
			expectedError: "error removing secret",
		},
	}
	for _, tc := range testCases {
		cmd := newSecretRemoveCommand(
			test.NewFakeCli(&fakeClient{
				secretRemoveFunc: tc.secretRemoveFunc,
			}),
		)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestSecretRemoveWithName(t *testing.T) {
	names := []string{"foo", "bar"}
	var removedSecrets []string
	cli := test.NewFakeCli(&fakeClient{
		secretRemoveFunc: func(name string) error {
			removedSecrets = append(removedSecrets, name)
			return nil
		},
	})
	cmd := newSecretRemoveCommand(cli)
	cmd.SetArgs(names)
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.DeepEqual(names, strings.Split(strings.TrimSpace(cli.OutBuffer().String()), "\n")))
	assert.Check(t, is.DeepEqual(names, removedSecrets))
}

func TestSecretRemoveContinueAfterError(t *testing.T) {
	names := []string{"foo", "bar"}
	var removedSecrets []string

	cli := test.NewFakeCli(&fakeClient{
		secretRemoveFunc: func(name string) error {
			removedSecrets = append(removedSecrets, name)
			if name == "foo" {
				return errors.Errorf("error removing secret: %s", name)
			}
			return nil
		},
	})

	cmd := newSecretRemoveCommand(cli)
	cmd.SetOutput(ioutil.Discard)
	cmd.SetArgs(names)
	assert.Error(t, cmd.Execute(), "error removing secret: foo")
	assert.Check(t, is.DeepEqual(names, removedSecrets))
}
