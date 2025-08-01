package plugin

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

var pluginFoo = &types.Plugin{
	ID:   "id-foo",
	Name: "name-foo",
	Config: types.PluginConfig{
		Description:   "plugin foo description",
		DockerVersion: "17.12.1-ce",
		Documentation: "plugin foo documentation",
		Entrypoint:    []string{"/foo"},
		Interface: types.PluginConfigInterface{
			Socket: "plugin-foo.sock",
		},
		Linux: types.PluginConfigLinux{
			Capabilities: []string{"CAP_SYS_ADMIN"},
		},
		WorkDir: "workdir-foo",
		Rootfs: &types.PluginConfigRootfs{
			DiffIds: []string{"sha256:deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"},
			Type:    "layers",
		},
	},
}

func TestInspectErrors(t *testing.T) {
	testCases := []struct {
		description   string
		args          []string
		flags         map[string]string
		expectedError string
		inspectFunc   func(name string) (*types.Plugin, []byte, error)
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
			inspectFunc: func(name string) (*types.Plugin, []byte, error) {
				return nil, nil, errors.New("error inspecting plugin")
			},
		},
		{
			description: "invalid format",
			args:        []string{"foo"},
			flags: map[string]string{
				"format": "{{invalid format}}",
			},
			expectedError: "template parsing error",
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
		inspectFunc func(name string) (*types.Plugin, []byte, error)
	}{
		{
			description: "inspect single plugin with format",
			args:        []string{"foo"},
			flags: map[string]string{
				"format": "{{ .Name }}",
			},
			golden: "plugin-inspect-single-with-format.golden",
			inspectFunc: func(name string) (*types.Plugin, []byte, error) {
				return &types.Plugin{
					ID:   "id-foo",
					Name: "name-foo",
				}, []byte{}, nil
			},
		},
		{
			description: "inspect single plugin without format",
			args:        []string{"foo"},
			golden:      "plugin-inspect-single-without-format.golden",
			inspectFunc: func(name string) (*types.Plugin, []byte, error) {
				return pluginFoo, nil, nil
			},
		},
		{
			description: "inspect multiple plugins with format",
			args:        []string{"foo", "bar"},
			flags: map[string]string{
				"format": "{{ .Name }}",
			},
			golden: "plugin-inspect-multiple-with-format.golden",
			inspectFunc: func(name string) (*types.Plugin, []byte, error) {
				switch name {
				case "foo":
					return &types.Plugin{
						ID:   "id-foo",
						Name: "name-foo",
					}, []byte{}, nil
				case "bar":
					return &types.Plugin{
						ID:   "id-bar",
						Name: "name-bar",
					}, []byte{}, nil
				default:
					return nil, nil, fmt.Errorf("unexpected plugin name: %s", name)
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
