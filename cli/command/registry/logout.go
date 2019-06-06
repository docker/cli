package registry

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/registry"
	"github.com/spf13/cobra"
)

// NewLogoutCommand creates a new `docker logout` command
func NewLogoutCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout [SERVER]",
		Short: "Log out from a Docker registry",
		Long:  "Log out from a Docker registry.\nIf no server is specified, the default is defined by the daemon.",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var serverAddress string
			if len(args) > 0 {
				serverAddress = args[0]
			}
			return runLogout(dockerCli, serverAddress)
		},
	}

	return cmd
}

func runLogout(dockerCli command.Cli, serverAddress string) error {
	ctx := context.Background()

	var (
		hostnameAddress = serverAddress
		loggedIn        bool     // is set later, when checking for credentials
		regsToTry       []string // is set based on the type of registry
	)

	// differentiate between default und private registry
	if serverAddress == "" {
		// if no server address given, handle case for default registry
		serverAddress = command.ElectAuthServer(ctx, dockerCli)
		regsToTry = []string{serverAddress}
	} else {
		// if server address given, handle case for private registry
		hostnameAddress = registry.ConvertToHostname(serverAddress)

		// the tries below are kept for backward compatibility where a user could have
		// saved the registry in one of the following format.
		regsToTry = []string{hostnameAddress, "http://" + hostnameAddress, "https://" + hostnameAddress}
	}

	// check if we're logged in based on the records in the config file
	// which means it couldn't have user/pass cause they may be in the creds store
	for _, s := range regsToTry {
		if _, ok := dockerCli.ConfigFile().AuthConfigs[s]; ok {
			loggedIn = true

			// remove credentials, for found auth config
			fmt.Fprintf(dockerCli.Out(), "Removing login credentials for %s\n", hostnameAddress)
			if err := dockerCli.ConfigFile().GetCredentialsStore(s).Erase(s); err != nil {
				fmt.Fprintf(dockerCli.Err(), "WARNING: could not erase credentials: %v\n", err)
			}
		}
	}

	if !loggedIn {
		fmt.Fprintf(dockerCli.Out(), "Not logged in to %s\n", hostnameAddress)
		return nil
	}

	return nil
}
