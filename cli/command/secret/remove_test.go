package secret

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
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
	assert.NoError(t, cmd.Execute())
	assert.Equal(t, names, strings.Split(strings.TrimSpace(cli.OutBuffer().String()), "\n"))
	assert.Equal(t, names, removedSecrets)
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
	assert.EqualError(t, cmd.Execute(), "error removing secret: foo")
	assert.Equal(t, names, removedSecrets)
}
