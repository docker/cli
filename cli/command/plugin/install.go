package plugin

import (
	"context"
	"fmt"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/jsonstream"
	"github.com/docker/cli/internal/prompt"
	"github.com/moby/moby/api/types/plugin"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type pluginOptions struct {
	remote          string
	localName       string
	grantPerms      bool
	disable         bool
	args            []string
	skipRemoteCheck bool
}

func newInstallCommand(dockerCLI command.Cli) *cobra.Command {
	var options pluginOptions
	cmd := &cobra.Command{
		Use:   "install [OPTIONS] PLUGIN [KEY=VALUE...]",
		Short: "Install a plugin",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.remote = args[0]
			if len(args) > 1 {
				options.args = args[1:]
			}
			return runInstall(cmd.Context(), dockerCLI, options)
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVar(&options.grantPerms, "grant-all-permissions", false, "Grant all permissions necessary to run the plugin")
	flags.BoolVar(&options.disable, "disable", false, "Do not enable the plugin on install")
	flags.StringVar(&options.localName, "alias", "", "Local name for plugin")

	// TODO(thaJeztah): DEPRECATED: remove in v29.1 or v30
	flags.Bool("disable-content-trust", true, "Skip image verification (deprecated)")
	_ = flags.MarkDeprecated("disable-content-trust", "support for docker content trust was removed")
	return cmd
}

func buildPullConfig(dockerCLI command.Cli, opts pluginOptions) (client.PluginInstallOptions, error) {
	// Names with both tag and digest will be treated by the daemon
	// as a pull by digest with a local name for the tag
	// (if no local name is provided).
	ref, err := reference.ParseNormalizedNamed(opts.remote)
	if err != nil {
		return client.PluginInstallOptions{}, err
	}

	encodedAuth, err := command.RetrieveAuthTokenFromImage(dockerCLI.ConfigFile(), ref.String())
	if err != nil {
		return client.PluginInstallOptions{}, err
	}

	return client.PluginInstallOptions{
		RegistryAuth:          encodedAuth,
		RemoteRef:             ref.String(),
		Disabled:              opts.disable,
		AcceptAllPermissions:  opts.grantPerms,
		AcceptPermissionsFunc: acceptPrivileges(dockerCLI, opts.remote),
		PrivilegeFunc:         nil,
		Args:                  opts.args,
	}, nil
}

func runInstall(ctx context.Context, dockerCLI command.Cli, opts pluginOptions) error {
	var localName string
	if opts.localName != "" {
		aref, err := reference.ParseNormalizedNamed(opts.localName)
		if err != nil {
			return err
		}
		if _, ok := aref.(reference.Canonical); ok {
			return fmt.Errorf("invalid name: %s", opts.localName)
		}
		localName = reference.FamiliarString(reference.TagNameOnly(aref))
	}

	options, err := buildPullConfig(dockerCLI, opts)
	if err != nil {
		return err
	}
	responseBody, err := dockerCLI.Client().PluginInstall(ctx, localName, options)
	if err != nil {
		return err
	}
	defer func() {
		_ = responseBody.Close()
	}()
	if err := jsonstream.Display(ctx, responseBody, dockerCLI.Out()); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(dockerCLI.Out(), "Installed plugin", opts.remote) // todo: return proper values from the API for this result
	return nil
}

func acceptPrivileges(dockerCLI command.Streams, name string) func(ctx context.Context, privileges plugin.Privileges) (bool, error) {
	return func(ctx context.Context, privileges plugin.Privileges) (bool, error) {
		_, _ = fmt.Fprintf(dockerCLI.Out(), "Plugin %q is requesting the following privileges:\n", name)
		for _, privilege := range privileges {
			_, _ = fmt.Fprintf(dockerCLI.Out(), " - %s: %v\n", privilege.Name, privilege.Value)
		}
		return prompt.Confirm(ctx, dockerCLI.In(), dockerCLI.Out(), "Do you grant the above permissions?")
	}
}
