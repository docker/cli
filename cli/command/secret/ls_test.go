package secret

import (
	"bytes"
	"io/ioutil"
	"testing"
	"time"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/pkg/errors"
	// Import builders to get the builder function as package function
	. "github.com/docker/cli/cli/internal/test/builders"
	"github.com/docker/docker/pkg/testutil"
	"github.com/docker/docker/pkg/testutil/golden"
	"github.com/stretchr/testify/assert"
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
		buf := new(bytes.Buffer)
		cmd := newSecretListCommand(
			test.NewFakeCliWithOutput(&fakeClient{
				secretListFunc: tc.secretListFunc,
			}, buf),
		)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestSecretList(t *testing.T) {
	buf := new(bytes.Buffer)
	cli := test.NewFakeCliWithOutput(&fakeClient{
		secretListFunc: func(options types.SecretListOptions) ([]swarm.Secret, error) {
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
	}, buf)
	cmd := newSecretListCommand(cli)
	cmd.SetOutput(buf)
	assert.NoError(t, cmd.Execute())
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "secret-list.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}

func TestSecretListWithQuietOption(t *testing.T) {
	buf := new(bytes.Buffer)
	cli := test.NewFakeCliWithOutput(&fakeClient{
		secretListFunc: func(options types.SecretListOptions) ([]swarm.Secret, error) {
			return []swarm.Secret{
				*Secret(SecretID("ID-foo"), SecretName("foo")),
				*Secret(SecretID("ID-bar"), SecretName("bar"), SecretLabels(map[string]string{
					"label": "label-bar",
				})),
			}, nil
		},
	}, buf)
	cmd := newSecretListCommand(cli)
	cmd.Flags().Set("quiet", "true")
	assert.NoError(t, cmd.Execute())
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "secret-list-with-quiet-option.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}

func TestSecretListWithConfigFormat(t *testing.T) {
	buf := new(bytes.Buffer)
	cli := test.NewFakeCliWithOutput(&fakeClient{
		secretListFunc: func(options types.SecretListOptions) ([]swarm.Secret, error) {
			return []swarm.Secret{
				*Secret(SecretID("ID-foo"), SecretName("foo")),
				*Secret(SecretID("ID-bar"), SecretName("bar"), SecretLabels(map[string]string{
					"label": "label-bar",
				})),
			}, nil
		},
	}, buf)
	cli.SetConfigFile(&configfile.ConfigFile{
		SecretFormat: "{{ .Name }} {{ .Labels }}",
	})
	cmd := newSecretListCommand(cli)
	assert.NoError(t, cmd.Execute())
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "secret-list-with-config-format.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}

func TestSecretListWithFormat(t *testing.T) {
	buf := new(bytes.Buffer)
	cli := test.NewFakeCliWithOutput(&fakeClient{
		secretListFunc: func(options types.SecretListOptions) ([]swarm.Secret, error) {
			return []swarm.Secret{
				*Secret(SecretID("ID-foo"), SecretName("foo")),
				*Secret(SecretID("ID-bar"), SecretName("bar"), SecretLabels(map[string]string{
					"label": "label-bar",
				})),
			}, nil
		},
	}, buf)
	cmd := newSecretListCommand(cli)
	cmd.Flags().Set("format", "{{ .Name }} {{ .Labels }}")
	assert.NoError(t, cmd.Execute())
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "secret-list-with-format.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}

func TestSecretListWithFilter(t *testing.T) {
	buf := new(bytes.Buffer)
	cli := test.NewFakeCliWithOutput(&fakeClient{
		secretListFunc: func(options types.SecretListOptions) ([]swarm.Secret, error) {
			assert.Equal(t, "foo", options.Filters.Get("name")[0], "foo")
			assert.Equal(t, "lbl1=Label-bar", options.Filters.Get("label")[0])
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
	}, buf)
	cmd := newSecretListCommand(cli)
	cmd.Flags().Set("filter", "name=foo")
	cmd.Flags().Set("filter", "label=lbl1=Label-bar")
	assert.NoError(t, cmd.Execute())
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "secret-list-with-filter.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}
