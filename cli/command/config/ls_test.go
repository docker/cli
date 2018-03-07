package config

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

func TestConfigListErrors(t *testing.T) {
	testCases := []struct {
		args           []string
		configListFunc func(types.ConfigListOptions) ([]swarm.Config, error)
		expectedError  string
	}{
		{
			args:          []string{"foo"},
			expectedError: "accepts no argument",
		},
		{
			configListFunc: func(options types.ConfigListOptions) ([]swarm.Config, error) {
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
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestConfigList(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		configListFunc: func(options types.ConfigListOptions) ([]swarm.Config, error) {
			return []swarm.Config{
				*Config(ConfigID("ID-1-foo"),
					ConfigName("1-foo"),
					ConfigVersion(swarm.Version{Index: 10}),
					ConfigCreatedAt(time.Now().Add(-2*time.Hour)),
					ConfigUpdatedAt(time.Now().Add(-1*time.Hour)),
				),
				*Config(ConfigID("ID-10-foo"),
					ConfigName("10-foo"),
					ConfigVersion(swarm.Version{Index: 11}),
					ConfigCreatedAt(time.Now().Add(-2*time.Hour)),
					ConfigUpdatedAt(time.Now().Add(-1*time.Hour)),
				),
				*Config(ConfigID("ID-2-foo"),
					ConfigName("2-foo"),
					ConfigVersion(swarm.Version{Index: 11}),
					ConfigCreatedAt(time.Now().Add(-2*time.Hour)),
					ConfigUpdatedAt(time.Now().Add(-1*time.Hour)),
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
		configListFunc: func(options types.ConfigListOptions) ([]swarm.Config, error) {
			return []swarm.Config{
				*Config(ConfigID("ID-foo"), ConfigName("foo")),
				*Config(ConfigID("ID-bar"), ConfigName("bar"), ConfigLabels(map[string]string{
					"label": "label-bar",
				})),
			}, nil
		},
	})
	cmd := newConfigListCommand(cli)
	cmd.Flags().Set("quiet", "true")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "config-list-with-quiet-option.golden")
}

func TestConfigListWithConfigFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		configListFunc: func(options types.ConfigListOptions) ([]swarm.Config, error) {
			return []swarm.Config{
				*Config(ConfigID("ID-foo"), ConfigName("foo")),
				*Config(ConfigID("ID-bar"), ConfigName("bar"), ConfigLabels(map[string]string{
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
		configListFunc: func(options types.ConfigListOptions) ([]swarm.Config, error) {
			return []swarm.Config{
				*Config(ConfigID("ID-foo"), ConfigName("foo")),
				*Config(ConfigID("ID-bar"), ConfigName("bar"), ConfigLabels(map[string]string{
					"label": "label-bar",
				})),
			}, nil
		},
	})
	cmd := newConfigListCommand(cli)
	cmd.Flags().Set("format", "{{ .Name }} {{ .Labels }}")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "config-list-with-format.golden")
}

func TestConfigListWithFilter(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		configListFunc: func(options types.ConfigListOptions) ([]swarm.Config, error) {
			assert.Check(t, is.Equal("foo", options.Filters.Get("name")[0]))
			assert.Check(t, is.Equal("lbl1=Label-bar", options.Filters.Get("label")[0]))
			return []swarm.Config{
				*Config(ConfigID("ID-foo"),
					ConfigName("foo"),
					ConfigVersion(swarm.Version{Index: 10}),
					ConfigCreatedAt(time.Now().Add(-2*time.Hour)),
					ConfigUpdatedAt(time.Now().Add(-1*time.Hour)),
				),
				*Config(ConfigID("ID-bar"),
					ConfigName("bar"),
					ConfigVersion(swarm.Version{Index: 11}),
					ConfigCreatedAt(time.Now().Add(-2*time.Hour)),
					ConfigUpdatedAt(time.Now().Add(-1*time.Hour)),
				),
			}, nil
		},
	})
	cmd := newConfigListCommand(cli)
	cmd.Flags().Set("filter", "name=foo")
	cmd.Flags().Set("filter", "label=lbl1=Label-bar")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "config-list-with-filter.golden")
}
