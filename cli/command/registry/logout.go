package registry

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/internal/oauth/manager"
	"github.com/docker/docker/registry"
	"github.com/spf13/cobra"
)

// NewLogoutCommand creates a new `docker logout` command
func NewLogoutCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout [SERVER]",
		Short: "Log out from a registry",
		Long:  "Log out from a registry.\nIf no server is specified, the default is defined by the daemon.",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var serverAddress string
			if len(args) > 0 {
				serverAddress = args[0]
			}
			return runLogout(cmd.Context(), dockerCli, serverAddress)
		},
		Annotations: map[string]string{
			"category-top": "9",
		},
		// TODO (thaJeztah) add completion for registries we have authentication stored for
	}

	return cmd
}

func runLogout(ctx context.Context, dockerCLI command.Cli, serverAddress string) error {
	maybePrintEnvAuthWarning(dockerCLI)

	var isDefaultRegistry bool

	if serverAddress == "" {
		serverAddress = registry.IndexServer
		isDefaultRegistry = true
	}

	var (
		regsToLogout    = []string{serverAddress}
		hostnameAddress = serverAddress
	)
	if !isDefaultRegistry {
		hostnameAddress = credentials.ConvertToHostname(serverAddress)
		// the tries below are kept for backward compatibility where a user could have
		// saved the registry in one of the following format.
		regsToLogout = append(regsToLogout, hostnameAddress, "http://"+hostnameAddress, "https://"+hostnameAddress)
	}

	if isDefaultRegistry {
		store := dockerCLI.ConfigFile().GetCredentialsStore(registry.IndexServer)
		if err := manager.NewManager(store).Logout(ctx); err != nil {
			_, _ = fmt.Fprintln(dockerCLI.Err(), "WARNING:", err)
		}
	}

	_, _ = fmt.Fprintln(dockerCLI.Out(), "Removing login credentials for", hostnameAddress)
	errs := make(map[string]error)
	for _, r := range regsToLogout {
		if err := dockerCLI.ConfigFile().GetCredentialsStore(r).Erase(r); err != nil {
			errs[r] = err
		}
	}

	// if at least one removal succeeded, report success. Otherwise report errors
	if len(errs) == len(regsToLogout) {
		_, _ = fmt.Fprintln(dockerCLI.Err(), "WARNING: could not erase credentials:")
		for k, v := range errs {
			_, _ = fmt.Fprintf(dockerCLI.Err(), "%s: %s\n", k, v)
		}
	}

	return nil
}
