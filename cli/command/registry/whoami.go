package registry

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/internal/commands"
	"github.com/docker/cli/internal/registry"
	"github.com/spf13/cobra"
)

func init() {
	commands.Register(newWhoamiCommand)
}

type whoamiOptions struct {
	serverAddress string
	all           bool
}

// newWhoamiCommand creates a new `docker whoami` command
func newWhoamiCommand(dockerCLI command.Cli) *cobra.Command {
	var opts whoamiOptions

	cmd := &cobra.Command{
		Use:   "whoami [SERVER]",
		Short: "Display the username of the currently logged in user",
		Long:  "Display the username of the currently logged in user.\nDefaults to Docker Hub if no server is specified.",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.serverAddress = args[0]
			}
			return runWhoami(cmd.Context(), dockerCLI, opts)
		},
		Annotations: map[string]string{
			"category-top": "10",
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.all, "all", false, "Display usernames for all authenticated registries")

	return cmd
}

func runWhoami(_ context.Context, dockerCLI command.Cli, opts whoamiOptions) error {
	maybePrintEnvAuthWarning(dockerCLI)

	if opts.all {
		return runWhoamiAll(dockerCLI)
	}
	return runWhoamiSingle(dockerCLI, opts)
}

func runWhoamiSingle(dockerCLI command.Cli, opts whoamiOptions) error {
	serverAddress := opts.serverAddress
	if serverAddress == "" || serverAddress == registry.DefaultNamespace {
		serverAddress = registry.IndexServer
	} else {
		serverAddress = credentials.ConvertToHostname(serverAddress)
	}

	authConfig, err := dockerCLI.ConfigFile().GetAuthConfig(serverAddress)
	if err != nil {
		return err
	}

	if authConfig.Username == "" {
		registryName := "Docker Hub"
		if opts.serverAddress != "" && opts.serverAddress != registry.DefaultNamespace {
			registryName = serverAddress
		}
		return fmt.Errorf("not logged in to %s", registryName)
	}

	fmt.Fprintln(dockerCLI.Out(), authConfig.Username)
	return nil
}

func runWhoamiAll(dockerCLI command.Cli) error {
	creds, err := dockerCLI.ConfigFile().GetAllCredentials()
	if err != nil {
		return err
	}

	if len(creds) == 0 {
		return errors.New("not logged in to any registries")
	}

	for serverAddress, authConfig := range creds {
		if authConfig.Username != "" {
			fmt.Fprintf(dockerCLI.Out(), "%s: %s\n", serverAddress, authConfig.Username)
		}
	}

	return nil
}
