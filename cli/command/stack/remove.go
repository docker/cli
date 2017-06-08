package stack

import (
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

type removeOptions struct {
	namespaces []string
	force      bool
}

func newRemoveCommand(dockerCli command.Cli) *cobra.Command {
	var opts removeOptions

	cmd := &cobra.Command{
		Use:     "rm STACK [STACK...]",
		Aliases: []string{"remove", "down"},
		Short:   "Remove one or more stacks",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.namespaces = args
			return runRemove(dockerCli, opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.force, "force", "f", false, "Force the removal of the stack")
	return cmd
}

const warning = `WARNING! This will remove:
Services: %s
Secrets: %s
Networks: %s
Are you sure you want to continue?`

func formatWarning(warning string, svcs []swarm.Service, scrts []swarm.Secret, ntwks []types.NetworkResource) string {
	services := []string{}
	secrets := []string{}
	networks := []string{}
	for _, svc := range svcs {
		services = append(services, svc.Spec.Annotations.Name)
	}
	for _, scrt := range scrts {
		secrets = append(secrets, scrt.Spec.Annotations.Name)
	}
	for _, ntwk := range ntwks {
		networks = append(networks, ntwk.Name)
	}
	return fmt.Sprintf(warning, strings.Join(services, ","), strings.Join(secrets, ","), strings.Join(networks, ","))
}

func runRemove(dockerCli command.Cli, opts removeOptions) error {
	namespaces := opts.namespaces
	client := dockerCli.Client()
	ctx := context.Background()

	var errs []string
	for _, namespace := range namespaces {
		services, err := getServices(ctx, client, namespace)
		if err != nil {
			return err
		}

		networks, err := getStackNetworks(ctx, client, namespace)
		if err != nil {
			return err
		}

		secrets, err := getStackSecrets(ctx, client, namespace)
		if err != nil {
			return err
		}

		configs, err := getStackConfigs(ctx, client, namespace)
		if err != nil {
			return err
		}

		if len(services)+len(networks)+len(secrets)+len(configs) == 0 {
			fmt.Fprintf(dockerCli.Out(), "Nothing found in stack: %s\n", namespace)
			continue
		}

		if !opts.force && !command.PromptForConfirmation(dockerCli.In(), dockerCli.Out(), formatWarning(warning, services, secrets, networks)) {
			continue
		}

		hasError := removeServices(ctx, dockerCli, services)
		hasError = removeSecrets(ctx, dockerCli, secrets) || hasError
		hasError = removeConfigs(ctx, dockerCli, configs) || hasError
		hasError = removeNetworks(ctx, dockerCli, networks) || hasError

		if hasError {
			errs = append(errs, fmt.Sprintf("Failed to remove some resources from stack: %s", namespace))
		}
	}

	if len(errs) > 0 {
		return errors.Errorf(strings.Join(errs, "\n"))
	}
	return nil
}

func removeServices(
	ctx context.Context,
	dockerCli command.Cli,
	services []swarm.Service,
) bool {
	var err error
	for _, service := range services {
		fmt.Fprintf(dockerCli.Err(), "Removing service %s\n", service.Spec.Name)
		if err = dockerCli.Client().ServiceRemove(ctx, service.ID); err != nil {
			fmt.Fprintf(dockerCli.Err(), "Failed to remove service %s: %s", service.ID, err)
		}
	}
	return err != nil
}

func removeNetworks(
	ctx context.Context,
	dockerCli command.Cli,
	networks []types.NetworkResource,
) bool {
	var err error
	for _, network := range networks {
		fmt.Fprintf(dockerCli.Err(), "Removing network %s\n", network.Name)
		if err = dockerCli.Client().NetworkRemove(ctx, network.ID); err != nil {
			fmt.Fprintf(dockerCli.Err(), "Failed to remove network %s: %s", network.ID, err)
		}
	}
	return err != nil
}

func removeSecrets(
	ctx context.Context,
	dockerCli command.Cli,
	secrets []swarm.Secret,
) bool {
	var err error
	for _, secret := range secrets {
		fmt.Fprintf(dockerCli.Err(), "Removing secret %s\n", secret.Spec.Name)
		if err = dockerCli.Client().SecretRemove(ctx, secret.ID); err != nil {
			fmt.Fprintf(dockerCli.Err(), "Failed to remove secret %s: %s", secret.ID, err)
		}
	}
	return err != nil
}

func removeConfigs(
	ctx context.Context,
	dockerCli command.Cli,
	configs []swarm.Config,
) bool {
	var err error
	for _, config := range configs {
		fmt.Fprintf(dockerCli.Err(), "Removing config %s\n", config.Spec.Name)
		if err = dockerCli.Client().ConfigRemove(ctx, config.ID); err != nil {
			fmt.Fprintf(dockerCli.Err(), "Failed to remove config %s: %s", config.ID, err)
		}
	}
	return err != nil
}
