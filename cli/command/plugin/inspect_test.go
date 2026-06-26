package plugin

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/plugin"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

var pluginFoo = client.PluginInspectResult{
	Plugin: plugin.Plugin{
		ID:   "id-foo",
		Name: "name-foo",
		Config: plugin.Config{
			Description:   "plugin foo description",
			Documentation: "plugin foo documentation",
			Entrypoint:    []string{"/foo"},
			Interface: plugin.Interface{
				Socket: "plugin-foo.sock",
			},
			Linux: plugin.LinuxConfig{
				Capabilities: []string{"CAP_SYS_ADMIN"},
			},
			WorkDir: "workdir-foo",
			Rootfs: &plugin.RootFS{
				DiffIds: []string{"sha256:deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"},
				Type:    "layers",
			},
		},
	},
}

func TestInspectErrors(t *testing.T) {
	testCases := []struct {
		description   string
		args          []string
		flags         map[string]string
		expectedError string
		inspectFunc   func(name string) (client.PluginInspectResult, error)
	}{
		{
			description:   "too few arguments",
			args:          []string{},
			expectedError: "requires at least 1 argument",
		},
		{
			description:   "error inspecting plugin",
			args:          []string{"foo"},
			expectedError: "error inspecting plugin",
			inspectFunc: func(string) (client.PluginInspectResult, error) {
				return client.PluginInspectResult{}, errors.New("error inspecting plugin")
			},
		},
		{
			description: "invalid format",
			args:        []string{"foo"},
			flags: map[string]string{
				"format": "{{invalid format}}",
			},
			expectedError: "template parsing error",
			inspectFunc: func(string) (client.PluginInspectResult, error) {
				return client.PluginInspectResult{}, errors.New("this function should not be called in this test")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{pluginInspectFunc: tc.inspectFunc})
			cmd := newInspectCommand(cli)
			cmd.SetArgs(tc.args)
			for key, value := range tc.flags {
				assert.NilError(t, cmd.Flags().Set(key, value))
			}
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestInspect(t *testing.T) {
	testCases := []struct {
		description string
		args        []string
		flags       map[string]string
		golden      string
		inspectFunc func(name string) (client.PluginInspectResult, error)
	}{
		{
			description: "inspect single plugin with format",
			args:        []string{"foo"},
			flags: map[string]string{
				"format": "{{ .Name }}",
			},
			golden: "plugin-inspect-single-with-format.golden",
			inspectFunc: func(name string) (client.PluginInspectResult, error) {
				return client.PluginInspectResult{
					Plugin: plugin.Plugin{
						ID:   "id-foo",
						Name: "name-foo",
					},
				}, nil
			},
		},
		{
			description: "inspect single plugin without format",
			args:        []string{"foo"},
			golden:      "plugin-inspect-single-without-format.golden",
			inspectFunc: func(name string) (client.PluginInspectResult, error) {
				return pluginFoo, nil
			},
		},
		{
			description: "inspect multiple plugins with format",
			args:        []string{"foo", "bar"},
			flags: map[string]string{
				"format": "{{ .Name }}",
			},
			golden: "plugin-inspect-multiple-with-format.golden",
			inspectFunc: func(name string) (client.PluginInspectResult, error) {
				switch name {
				case "foo":
					return client.PluginInspectResult{
						Plugin: plugin.Plugin{
							ID:   "id-foo",
							Name: "name-foo",
						},
					}, nil
				case "bar":
					return client.PluginInspectResult{
						Plugin: plugin.Plugin{
							ID:   "id-bar",
							Name: "name-bar",
						},
					}, nil
				default:
					return client.PluginInspectResult{}, fmt.Errorf("unexpected plugin name: %s", name)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{pluginInspectFunc: tc.inspectFunc})
			cmd := newInspectCommand(cli)
			cmd.SetArgs(tc.args)
			for key, value := range tc.flags {
				assert.NilError(t, cmd.Flags().Set(key, value))
			}
			assert.NilError(t, cmd.Execute())
			golden.Assert(t, cli.OutBuffer().String(), tc.golden)
		})
	}
}
