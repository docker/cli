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
	var isDefaultRegistry bool

	if serverAddress == "" {
		serverAddress = command.ElectAuthServer(ctx, dockerCli)
		isDefaultRegistry = true
	}

	var (
		regsToLogout    = []string{serverAddress}
		hostnameAddress = serverAddress
	)
	if !isDefaultRegistry {
		hostnameAddress = registry.ConvertToHostname(serverAddress)
		// the tries below are kept for backward compatibility where a user could have
		// saved the registry in one of the following format.
		regsToLogout = append(regsToLogout, hostnameAddress, "http://"+hostnameAddress, "https://"+hostnameAddress)
	}

	fmt.Fprintf(dockerCli.Out(), "Removing login credentials for %s\n", hostnameAddress)
	errs := make(map[string]error)
	for _, r := range regsToLogout {
		if err := dockerCli.ConfigFile().GetCredentialsStore(r).Erase(r); err != nil {
			errs[r] = err
		}
	}

	// if at least one removal succeeded, report success. Otherwise report errors
	if len(errs) == len(regsToLogout) {
		fmt.Fprintln(dockerCli.Err(), "WARNING: could not erase credentials:")
		for k, v := range errs {
			fmt.Fprintf(dockerCli.Err(), "%s: %s\n", k, v)
		}
	}

	return nil
}
