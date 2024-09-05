package config

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

func TestConfigInspectErrors(t *testing.T) {
	testCases := []struct {
		args              []string
		flags             map[string]string
		configInspectFunc func(_ context.Context, configID string) (swarm.Config, []byte, error)
		expectedError     string
	}{
		{
			expectedError: "requires at least 1 argument",
		},
		{
			args: []string{"foo"},
			configInspectFunc: func(_ context.Context, configID string) (swarm.Config, []byte, error) {
				return swarm.Config{}, nil, errors.Errorf("error while inspecting the config")
			},
			expectedError: "error while inspecting the config",
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
			configInspectFunc: func(_ context.Context, configID string) (swarm.Config, []byte, error) {
				if configID == "foo" {
					return *builders.Config(builders.ConfigName("foo")), nil, nil
				}
				return swarm.Config{}, nil, errors.Errorf("error while inspecting the config")
			},
			expectedError: "error while inspecting the config",
		},
	}
	for _, tc := range testCases {
		cmd := newConfigInspectCommand(
			test.NewFakeCli(&fakeClient{
				configInspectFunc: tc.configInspectFunc,
			}),
		)
		cmd.SetArgs(tc.args)
		for key, value := range tc.flags {
			assert.Check(t, cmd.Flags().Set(key, value))
		}
		cmd.SetOut(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestConfigInspectWithoutFormat(t *testing.T) {
	testCases := []struct {
		name              string
		args              []string
		configInspectFunc func(_ context.Context, configID string) (swarm.Config, []byte, error)
	}{
		{
			name: "single-config",
			args: []string{"foo"},
			configInspectFunc: func(_ context.Context, name string) (swarm.Config, []byte, error) {
				if name != "foo" {
					return swarm.Config{}, nil, errors.Errorf("Invalid name, expected %s, got %s", "foo", name)
				}
				return *builders.Config(builders.ConfigID("ID-foo"), builders.ConfigName("foo")), nil, nil
			},
		},
		{
			name: "multiple-configs-with-labels",
			args: []string{"foo", "bar"},
			configInspectFunc: func(_ context.Context, name string) (swarm.Config, []byte, error) {
				return *builders.Config(builders.ConfigID("ID-"+name), builders.ConfigName(name), builders.ConfigLabels(map[string]string{
					"label1": "label-foo",
				})), nil, nil
			},
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{configInspectFunc: tc.configInspectFunc})
		cmd := newConfigInspectCommand(cli)
		cmd.SetArgs(tc.args)
		assert.NilError(t, cmd.Execute())
		golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("config-inspect-without-format.%s.golden", tc.name))
	}
}

func TestConfigInspectWithFormat(t *testing.T) {
	configInspectFunc := func(_ context.Context, name string) (swarm.Config, []byte, error) {
		return *builders.Config(builders.ConfigName("foo"), builders.ConfigLabels(map[string]string{
			"label1": "label-foo",
		})), nil, nil
	}
	testCases := []struct {
		name              string
		format            string
		args              []string
		configInspectFunc func(_ context.Context, name string) (swarm.Config, []byte, error)
	}{
		{
			name:              "simple-template",
			format:            "{{.Spec.Name}}",
			args:              []string{"foo"},
			configInspectFunc: configInspectFunc,
		},
		{
			name:              "json-template",
			format:            "{{json .Spec.Labels}}",
			args:              []string{"foo"},
			configInspectFunc: configInspectFunc,
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			configInspectFunc: tc.configInspectFunc,
		})
		cmd := newConfigInspectCommand(cli)
		cmd.SetArgs(tc.args)
		assert.Check(t, cmd.Flags().Set("format", tc.format))
		assert.NilError(t, cmd.Execute())
		golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("config-inspect-with-format.%s.golden", tc.name))
	}
}

func TestConfigInspectPretty(t *testing.T) {
	testCases := []struct {
		name              string
		configInspectFunc func(context.Context, string) (swarm.Config, []byte, error)
	}{
		{
			name: "simple",
			configInspectFunc: func(_ context.Context, id string) (swarm.Config, []byte, error) {
				return *builders.Config(
					builders.ConfigLabels(map[string]string{
						"lbl1": "value1",
					}),
					builders.ConfigID("configID"),
					builders.ConfigName("configName"),
					builders.ConfigCreatedAt(time.Time{}),
					builders.ConfigUpdatedAt(time.Time{}),
					builders.ConfigData([]byte("payload here")),
				), []byte{}, nil
			},
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			configInspectFunc: tc.configInspectFunc,
		})
		cmd := newConfigInspectCommand(cli)

		cmd.SetArgs([]string{"configID"})
		assert.Check(t, cmd.Flags().Set("pretty", "true"))
		assert.NilError(t, cmd.Execute())
		golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("config-inspect-pretty.%s.golden", tc.name))
	}
}
