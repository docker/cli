package secret

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types/swarm"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestSecretInspectErrors(t *testing.T) {
	testCases := []struct {
		args              []string
		flags             map[string]string
		secretInspectFunc func(ctx context.Context, secretID string) (swarm.Secret, []byte, error)
		expectedError     string
	}{
		{
			expectedError: "requires at least 1 argument",
		},
		{
			args: []string{"foo"},
			secretInspectFunc: func(_ context.Context, secretID string) (swarm.Secret, []byte, error) {
				return swarm.Secret{}, nil, errors.Errorf("error while inspecting the secret")
			},
			expectedError: "error while inspecting the secret",
		},
		{
			args: []string{"foo"},
			flags: map[string]string{
				"format": "{{invalid format}}",
			},
			expectedError: "template parsing error",
		},
		{
			args: []string{"foo", "bar"},
			secretInspectFunc: func(_ context.Context, secretID string) (swarm.Secret, []byte, error) {
				if secretID == "foo" {
					return *builders.Secret(builders.SecretName("foo")), nil, nil
				}
				return swarm.Secret{}, nil, errors.Errorf("error while inspecting the secret")
			},
			expectedError: "error while inspecting the secret",
		},
	}
	for _, tc := range testCases {
		cmd := newSecretInspectCommand(
			test.NewFakeCli(&fakeClient{
				secretInspectFunc: tc.secretInspectFunc,
			}),
		)
		cmd.SetArgs(tc.args)
		for key, value := range tc.flags {
			assert.Check(t, cmd.Flags().Set(key, value))
		}
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestSecretInspectWithoutFormat(t *testing.T) {
	testCases := []struct {
		name              string
		args              []string
		secretInspectFunc func(ctx context.Context, secretID string) (swarm.Secret, []byte, error)
	}{
		{
			name: "single-secret",
			args: []string{"foo"},
			secretInspectFunc: func(_ context.Context, name string) (swarm.Secret, []byte, error) {
				if name != "foo" {
					return swarm.Secret{}, nil, errors.Errorf("Invalid name, expected %s, got %s", "foo", name)
				}
				return *builders.Secret(builders.SecretID("ID-foo"), builders.SecretName("foo")), nil, nil
			},
		},
		{
			name: "multiple-secrets-with-labels",
			args: []string{"foo", "bar"},
			secretInspectFunc: func(_ context.Context, name string) (swarm.Secret, []byte, error) {
				return *builders.Secret(builders.SecretID("ID-"+name), builders.SecretName(name), builders.SecretLabels(map[string]string{
					"label1": "label-foo",
				})), nil, nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				secretInspectFunc: tc.secretInspectFunc,
			})
			cmd := newSecretInspectCommand(cli)
			cmd.SetArgs(tc.args)
			assert.NilError(t, cmd.Execute())
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("secret-inspect-without-format.%s.golden", tc.name))
		})
	}
}

func TestSecretInspectWithFormat(t *testing.T) {
	secretInspectFunc := func(_ context.Context, name string) (swarm.Secret, []byte, error) {
		return *builders.Secret(builders.SecretName("foo"), builders.SecretLabels(map[string]string{
			"label1": "label-foo",
		})), nil, nil
	}
	testCases := []struct {
		name              string
		format            string
		args              []string
		secretInspectFunc func(_ context.Context, name string) (swarm.Secret, []byte, error)
	}{
		{
			name:              "simple-template",
			format:            "{{.Spec.Name}}",
			args:              []string{"foo"},
			secretInspectFunc: secretInspectFunc,
		},
		{
			name:              "json-template",
			format:            "{{json .Spec.Labels}}",
			args:              []string{"foo"},
			secretInspectFunc: secretInspectFunc,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				secretInspectFunc: tc.secretInspectFunc,
			})
			cmd := newSecretInspectCommand(cli)
			cmd.SetArgs(tc.args)
			assert.Check(t, cmd.Flags().Set("format", tc.format))
			assert.NilError(t, cmd.Execute())
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("secret-inspect-with-format.%s.golden", tc.name))
		})
	}
}

func TestSecretInspectPretty(t *testing.T) {
	testCases := []struct {
		name              string
		secretInspectFunc func(context.Context, string) (swarm.Secret, []byte, error)
	}{
		{
			name: "simple",
			secretInspectFunc: func(_ context.Context, id string) (swarm.Secret, []byte, error) {
				return *builders.Secret(
					builders.SecretLabels(map[string]string{
						"lbl1": "value1",
					}),
					builders.SecretID("secretID"),
					builders.SecretName("secretName"),
					builders.SecretDriver("driver"),
					builders.SecretCreatedAt(time.Time{}),
					builders.SecretUpdatedAt(time.Time{}),
				), []byte{}, nil
			},
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			secretInspectFunc: tc.secretInspectFunc,
		})
		cmd := newSecretInspectCommand(cli)
		cmd.SetArgs([]string{"secretID"})
		assert.Check(t, cmd.Flags().Set("pretty", "true"))
		assert.NilError(t, cmd.Execute())
		golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("secret-inspect-pretty.%s.golden", tc.name))
	}
}
