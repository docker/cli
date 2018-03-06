package secret

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/pkg/errors"
	// Import builders to get the builder function as package function
	. "github.com/docker/cli/internal/test/builders"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/golden"
)

func TestSecretListErrors(t *testing.T) {
	testCases := []struct {
		args           []string
		secretListFunc func(types.SecretListOptions) ([]swarm.Secret, error)
		expectedError  string
	}{
		{
			args:          []string{"foo"},
			expectedError: "accepts no argument",
		},
		{
			secretListFunc: func(options types.SecretListOptions) ([]swarm.Secret, error) {
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
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestSecretList(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		secretListFunc: func(options types.SecretListOptions) ([]swarm.Secret, error) {
			return []swarm.Secret{
				*Secret(SecretID("ID-1-foo"),
					SecretName("1-foo"),
					SecretVersion(swarm.Version{Index: 10}),
					SecretCreatedAt(time.Now().Add(-2*time.Hour)),
					SecretUpdatedAt(time.Now().Add(-1*time.Hour)),
				),
				*Secret(SecretID("ID-10-foo"),
					SecretName("10-foo"),
					SecretVersion(swarm.Version{Index: 11}),
					SecretCreatedAt(time.Now().Add(-2*time.Hour)),
					SecretUpdatedAt(time.Now().Add(-1*time.Hour)),
					SecretDriver("driver"),
				),
				*Secret(SecretID("ID-2-foo"),
					SecretName("2-foo"),
					SecretVersion(swarm.Version{Index: 11}),
					SecretCreatedAt(time.Now().Add(-2*time.Hour)),
					SecretUpdatedAt(time.Now().Add(-1*time.Hour)),
					SecretDriver("driver"),
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
		secretListFunc: func(options types.SecretListOptions) ([]swarm.Secret, error) {
			return []swarm.Secret{
				*Secret(SecretID("ID-foo"), SecretName("foo")),
				*Secret(SecretID("ID-bar"), SecretName("bar"), SecretLabels(map[string]string{
					"label": "label-bar",
				})),
			}, nil
		},
	})
	cmd := newSecretListCommand(cli)
	cmd.Flags().Set("quiet", "true")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "secret-list-with-quiet-option.golden")
}

func TestSecretListWithConfigFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		secretListFunc: func(options types.SecretListOptions) ([]swarm.Secret, error) {
			return []swarm.Secret{
				*Secret(SecretID("ID-foo"), SecretName("foo")),
				*Secret(SecretID("ID-bar"), SecretName("bar"), SecretLabels(map[string]string{
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
		secretListFunc: func(options types.SecretListOptions) ([]swarm.Secret, error) {
			return []swarm.Secret{
				*Secret(SecretID("ID-foo"), SecretName("foo")),
				*Secret(SecretID("ID-bar"), SecretName("bar"), SecretLabels(map[string]string{
					"label": "label-bar",
				})),
			}, nil
		},
	})
	cmd := newSecretListCommand(cli)
	cmd.Flags().Set("format", "{{ .Name }} {{ .Labels }}")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "secret-list-with-format.golden")
}

func TestSecretListWithFilter(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		secretListFunc: func(options types.SecretListOptions) ([]swarm.Secret, error) {
			assert.Check(t, is.Equal("foo", options.Filters.Get("name")[0]), "foo")
			assert.Check(t, is.Equal("lbl1=Label-bar", options.Filters.Get("label")[0]))
			return []swarm.Secret{
				*Secret(SecretID("ID-foo"),
					SecretName("foo"),
					SecretVersion(swarm.Version{Index: 10}),
					SecretCreatedAt(time.Now().Add(-2*time.Hour)),
					SecretUpdatedAt(time.Now().Add(-1*time.Hour)),
				),
				*Secret(SecretID("ID-bar"),
					SecretName("bar"),
					SecretVersion(swarm.Version{Index: 11}),
					SecretCreatedAt(time.Now().Add(-2*time.Hour)),
					SecretUpdatedAt(time.Now().Add(-1*time.Hour)),
				),
			}, nil
		},
	})
	cmd := newSecretListCommand(cli)
	cmd.Flags().Set("filter", "name=foo")
	cmd.Flags().Set("filter", "label=lbl1=Label-bar")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "secret-list-with-filter.golden")
}
