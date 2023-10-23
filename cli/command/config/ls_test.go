package config

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

func TestConfigListErrors(t *testing.T) {
	testCases := []struct {
		args           []string
		configListFunc func(context.Context, types.ConfigListOptions) ([]swarm.Config, error)
		expectedError  string
	}{
		{
			args:          []string{"foo"},
			expectedError: "accepts no argument",
		},
		{
			configListFunc: func(_ context.Context, options types.ConfigListOptions) ([]swarm.Config, error) {
				return []swarm.Config{}, errors.Errorf("error listing configs")
			},
			expectedError: "error listing configs",
		},
	}
	for _, tc := range testCases {
		cmd := newConfigListCommand(
			test.NewFakeCli(&fakeClient{
				configListFunc: tc.configListFunc,
			}),
		)
		cmd.SetArgs(tc.args)
		cmd.SetOut(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestConfigList(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		configListFunc: func(_ context.Context, options types.ConfigListOptions) ([]swarm.Config, error) {
			return []swarm.Config{
				*builders.Config(builders.ConfigID("ID-1-foo"),
					builders.ConfigName("1-foo"),
					builders.ConfigVersion(swarm.Version{Index: 10}),
					builders.ConfigCreatedAt(time.Now().Add(-2*time.Hour)),
					builders.ConfigUpdatedAt(time.Now().Add(-1*time.Hour)),
				),
				*builders.Config(builders.ConfigID("ID-10-foo"),
					builders.ConfigName("10-foo"),
					builders.ConfigVersion(swarm.Version{Index: 11}),
					builders.ConfigCreatedAt(time.Now().Add(-2*time.Hour)),
					builders.ConfigUpdatedAt(time.Now().Add(-1*time.Hour)),
				),
				*builders.Config(builders.ConfigID("ID-2-foo"),
					builders.ConfigName("2-foo"),
					builders.ConfigVersion(swarm.Version{Index: 11}),
					builders.ConfigCreatedAt(time.Now().Add(-2*time.Hour)),
					builders.ConfigUpdatedAt(time.Now().Add(-1*time.Hour)),
				),
			}, nil
		},
	})
	cmd := newConfigListCommand(cli)
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "config-list-sort.golden")
}

func TestConfigListWithQuietOption(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		configListFunc: func(_ context.Context, options types.ConfigListOptions) ([]swarm.Config, error) {
			return []swarm.Config{
				*builders.Config(builders.ConfigID("ID-foo"), builders.ConfigName("foo")),
				*builders.Config(builders.ConfigID("ID-bar"), builders.ConfigName("bar"), builders.ConfigLabels(map[string]string{
					"label": "label-bar",
				})),
			}, nil
		},
	})
	cmd := newConfigListCommand(cli)
	assert.Check(t, cmd.Flags().Set("quiet", "true"))
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "config-list-with-quiet-option.golden")
}

func TestConfigListWithConfigFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		configListFunc: func(_ context.Context, options types.ConfigListOptions) ([]swarm.Config, error) {
			return []swarm.Config{
				*builders.Config(builders.ConfigID("ID-foo"), builders.ConfigName("foo")),
				*builders.Config(builders.ConfigID("ID-bar"), builders.ConfigName("bar"), builders.ConfigLabels(map[string]string{
					"label": "label-bar",
				})),
			}, nil
		},
	})
	cli.SetConfigFile(&configfile.ConfigFile{
		ConfigFormat: "{{ .Name }} {{ .Labels }}",
	})
	cmd := newConfigListCommand(cli)
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "config-list-with-config-format.golden")
}

func TestConfigListWithFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		configListFunc: func(_ context.Context, options types.ConfigListOptions) ([]swarm.Config, error) {
			return []swarm.Config{
				*builders.Config(builders.ConfigID("ID-foo"), builders.ConfigName("foo")),
				*builders.Config(builders.ConfigID("ID-bar"), builders.ConfigName("bar"), builders.ConfigLabels(map[string]string{
					"label": "label-bar",
				})),
			}, nil
		},
	})
	cmd := newConfigListCommand(cli)
	assert.Check(t, cmd.Flags().Set("format", "{{ .Name }} {{ .Labels }}"))
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "config-list-with-format.golden")
}

func TestConfigListWithFilter(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		configListFunc: func(_ context.Context, options types.ConfigListOptions) ([]swarm.Config, error) {
			assert.Check(t, is.Equal("foo", options.Filters.Get("name")[0]))
			assert.Check(t, is.Equal("lbl1=Label-bar", options.Filters.Get("label")[0]))
			return []swarm.Config{
				*builders.Config(builders.ConfigID("ID-foo"),
					builders.ConfigName("foo"),
					builders.ConfigVersion(swarm.Version{Index: 10}),
					builders.ConfigCreatedAt(time.Now().Add(-2*time.Hour)),
					builders.ConfigUpdatedAt(time.Now().Add(-1*time.Hour)),
				),
				*builders.Config(builders.ConfigID("ID-bar"),
					builders.ConfigName("bar"),
					builders.ConfigVersion(swarm.Version{Index: 11}),
					builders.ConfigCreatedAt(time.Now().Add(-2*time.Hour)),
					builders.ConfigUpdatedAt(time.Now().Add(-1*time.Hour)),
				),
			}, nil
		},
	})
	cmd := newConfigListCommand(cli)
	assert.Check(t, cmd.Flags().Set("filter", "name=foo"))
	assert.Check(t, cmd.Flags().Set("filter", "label=lbl1=Label-bar"))
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "config-list-with-filter.golden")
}
