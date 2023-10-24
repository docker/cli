package secret

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestSecretListErrors(t *testing.T) {
	testCases := []struct {
		args           []string
		secretListFunc func(context.Context, types.SecretListOptions) ([]swarm.Secret, error)
		expectedError  string
	}{
		{
			args:          []string{"foo"},
			expectedError: "accepts no argument",
		},
		{
			secretListFunc: func(_ context.Context, options types.SecretListOptions) ([]swarm.Secret, error) {
				return []swarm.Secret{}, errors.Errorf("error listing secrets")
			},
			expectedError: "error listing secrets",
		},
	}
	for _, tc := range testCases {
		cmd := newSecretListCommand(
			test.NewFakeCli(&fakeClient{
				secretListFunc: tc.secretListFunc,
			}),
		)
		cmd.SetArgs(tc.args)
		cmd.SetOut(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestSecretList(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		secretListFunc: func(_ context.Context, options types.SecretListOptions) ([]swarm.Secret, error) {
			return []swarm.Secret{
				*builders.Secret(builders.SecretID("ID-1-foo"),
					builders.SecretName("1-foo"),
					builders.SecretVersion(swarm.Version{Index: 10}),
					builders.SecretCreatedAt(time.Now().Add(-2*time.Hour)),
					builders.SecretUpdatedAt(time.Now().Add(-1*time.Hour)),
				),
				*builders.Secret(builders.SecretID("ID-10-foo"),
					builders.SecretName("10-foo"),
					builders.SecretVersion(swarm.Version{Index: 11}),
					builders.SecretCreatedAt(time.Now().Add(-2*time.Hour)),
					builders.SecretUpdatedAt(time.Now().Add(-1*time.Hour)),
					builders.SecretDriver("driver"),
				),
				*builders.Secret(builders.SecretID("ID-2-foo"),
					builders.SecretName("2-foo"),
					builders.SecretVersion(swarm.Version{Index: 11}),
					builders.SecretCreatedAt(time.Now().Add(-2*time.Hour)),
					builders.SecretUpdatedAt(time.Now().Add(-1*time.Hour)),
					builders.SecretDriver("driver"),
				),
			}, nil
		},
	})
	cmd := newSecretListCommand(cli)
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "secret-list-sort.golden")
}

func TestSecretListWithQuietOption(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		secretListFunc: func(_ context.Context, options types.SecretListOptions) ([]swarm.Secret, error) {
			return []swarm.Secret{
				*builders.Secret(builders.SecretID("ID-foo"), builders.SecretName("foo")),
				*builders.Secret(builders.SecretID("ID-bar"), builders.SecretName("bar"), builders.SecretLabels(map[string]string{
					"label": "label-bar",
				})),
			}, nil
		},
	})
	cmd := newSecretListCommand(cli)
	assert.Check(t, cmd.Flags().Set("quiet", "true"))
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "secret-list-with-quiet-option.golden")
}

func TestSecretListWithConfigFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		secretListFunc: func(_ context.Context, options types.SecretListOptions) ([]swarm.Secret, error) {
			return []swarm.Secret{
				*builders.Secret(builders.SecretID("ID-foo"), builders.SecretName("foo")),
				*builders.Secret(builders.SecretID("ID-bar"), builders.SecretName("bar"), builders.SecretLabels(map[string]string{
					"label": "label-bar",
				})),
			}, nil
		},
	})
	cli.SetConfigFile(&configfile.ConfigFile{
		SecretFormat: "{{ .Name }} {{ .Labels }}",
	})
	cmd := newSecretListCommand(cli)
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "secret-list-with-config-format.golden")
}

func TestSecretListWithFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		secretListFunc: func(_ context.Context, options types.SecretListOptions) ([]swarm.Secret, error) {
			return []swarm.Secret{
				*builders.Secret(builders.SecretID("ID-foo"), builders.SecretName("foo")),
				*builders.Secret(builders.SecretID("ID-bar"), builders.SecretName("bar"), builders.SecretLabels(map[string]string{
					"label": "label-bar",
				})),
			}, nil
		},
	})
	cmd := newSecretListCommand(cli)
	assert.Check(t, cmd.Flags().Set("format", "{{ .Name }} {{ .Labels }}"))
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "secret-list-with-format.golden")
}

func TestSecretListWithFilter(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		secretListFunc: func(_ context.Context, options types.SecretListOptions) ([]swarm.Secret, error) {
			assert.Check(t, is.Equal("foo", options.Filters.Get("name")[0]), "foo")
			assert.Check(t, is.Equal("lbl1=Label-bar", options.Filters.Get("label")[0]))
			return []swarm.Secret{
				*builders.Secret(builders.SecretID("ID-foo"),
					builders.SecretName("foo"),
					builders.SecretVersion(swarm.Version{Index: 10}),
					builders.SecretCreatedAt(time.Now().Add(-2*time.Hour)),
					builders.SecretUpdatedAt(time.Now().Add(-1*time.Hour)),
				),
				*builders.Secret(builders.SecretID("ID-bar"),
					builders.SecretName("bar"),
					builders.SecretVersion(swarm.Version{Index: 11}),
					builders.SecretCreatedAt(time.Now().Add(-2*time.Hour)),
					builders.SecretUpdatedAt(time.Now().Add(-1*time.Hour)),
				),
			}, nil
		},
	})
	cmd := newSecretListCommand(cli)
	assert.Check(t, cmd.Flags().Set("filter", "name=foo"))
	assert.Check(t, cmd.Flags().Set("filter", "label=lbl1=Label-bar"))
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "secret-list-with-filter.golden")
}
