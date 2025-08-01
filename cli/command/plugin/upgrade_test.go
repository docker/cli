package plugin

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types"
	"github.com/moby/moby/client"
	"gotest.tools/v3/golden"
)

func TestUpgradePromptTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cli := test.NewFakeCli(&fakeClient{
		pluginUpgradeFunc: func(name string, options client.PluginInstallOptions) (io.ReadCloser, error) {
			return nil, errors.New("should not be called")
		},
		pluginInspectFunc: func(name string) (*types.Plugin, []byte, error) {
			return &types.Plugin{
				ID:              "5724e2c8652da337ab2eedd19fc6fc0ec908e4bd907c7421bf6a8dfc70c4c078",
				Name:            "foo/bar",
				Enabled:         false,
				PluginReference: "localhost:5000/foo/bar:v0.1.0",
			}, []byte{}, nil
		},
	})
	cmd := newUpgradeCommand(cli)
	// need to set a remote address that does not match the plugin
	// reference sent by the `pluginInspectFunc`
	cmd.SetArgs([]string{"foo/bar", "localhost:5000/foo/bar:v1.0.0"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	test.TerminatePrompt(ctx, t, cmd, cli)
	golden.Assert(t, cli.OutBuffer().String(), "plugin-upgrade-terminate.golden")
}
