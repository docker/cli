package plugin

import (
	"context"
	"errors"
	"fmt"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/jsonstream"
	"github.com/docker/cli/internal/prompt"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

func newUpgradeCommand(dockerCLI command.Cli) *cobra.Command {
	var options pluginOptions
	cmd := &cobra.Command{
		Use:   "upgrade [OPTIONS] PLUGIN [REMOTE]",
		Short: "Upgrade an existing plugin",
		Args:  cli.RequiresRangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.localName = args[0]
			if len(args) == 2 {
				options.remote = args[1]
			}
			return runUpgrade(cmd.Context(), dockerCLI, options)
		},
		Annotations:           map[string]string{"version": "1.26"},
		ValidArgsFunction:     completeNames(dockerCLI, stateAny), // TODO(thaJeztah): should only complete for the first arg
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVar(&options.grantPerms, "grant-all-permissions", false, "Grant all permissions necessary to run the plugin")
	// TODO(thaJeztah): DEPRECATED: remove in v29.1 or v30
	flags.Bool("disable-content-trust", true, "Skip image verification (deprecated)")
	_ = flags.MarkDeprecated("disable-content-trust", "support for docker content trust was removed")
	flags.BoolVar(&options.skipRemoteCheck, "skip-remote-check", false, "Do not check if specified remote plugin matches existing plugin image")
	return cmd
}

func runUpgrade(ctx context.Context, dockerCLI command.Cli, opts pluginOptions) error {
	res, err := dockerCLI.Client().PluginInspect(ctx, opts.localName, client.PluginInspectOptions{})
	if err != nil {
		return fmt.Errorf("error reading plugin data: %w", err)
	}

	if res.Plugin.Enabled {
		return errors.New("the plugin must be disabled before upgrading")
	}

	opts.localName = res.Plugin.Name
	if opts.remote == "" {
		opts.remote = res.Plugin.PluginReference
	}
	remote, err := reference.ParseNormalizedNamed(opts.remote)
	if err != nil {
		return fmt.Errorf("error parsing remote upgrade image reference: %w", err)
	}
	remote = reference.TagNameOnly(remote)

	old, err := reference.ParseNormalizedNamed(res.Plugin.PluginReference)
	if err != nil {
		return fmt.Errorf("error parsing current image reference: %w", err)
	}
	old = reference.TagNameOnly(old)

	_, _ = fmt.Fprintf(dockerCLI.Out(), "Upgrading plugin %s from %s to %s\n", res.Plugin.Name, reference.FamiliarString(old), reference.FamiliarString(remote))
	if !opts.skipRemoteCheck && remote.String() != old.String() {
		r, err := prompt.Confirm(ctx, dockerCLI.In(), dockerCLI.Out(), "Plugin images do not match, are you sure?")
		if err != nil {
			return err
		}
		if !r {
			return cancelledErr{errors.New("plugin upgrade has been cancelled")}
		}
	}

	options, err := buildPullConfig(dockerCLI, opts)
	if err != nil {
		return err
	}

	responseBody, err := dockerCLI.Client().PluginUpgrade(ctx, opts.localName, client.PluginUpgradeOptions(options))
	if err != nil {
		return err
	}
	defer func() {
		_ = responseBody.Close()
	}()
	if err := jsonstream.Display(ctx, responseBody, dockerCLI.Out()); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(dockerCLI.Out(), "Upgraded plugin %s to %s\n", opts.localName, opts.remote) // todo: return proper values from the API for this result
	return nil
}

type cancelledErr struct{ error }

func (cancelledErr) Cancelled() {}
