package stack

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/compose/convert"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// deployOptions holds docker stack deploy options
type deployOptions struct {
	composefiles     []string
	namespace        string
	resolveImage     string
	sendRegistryAuth bool
	prune            bool
	detach           bool
	quiet            bool
}

func newDeployCommand(dockerCLI command.Cli) *cobra.Command {
	var opts deployOptions

	cmd := &cobra.Command{
		Use:     "deploy [OPTIONS] STACK",
		Aliases: []string{"up"},
		Short:   "Deploy a new stack or update an existing stack",
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.namespace = args[0]
			if err := validateStackName(opts.namespace); err != nil {
				return err
			}
			config, err := loadComposeFile(dockerCLI, opts)
			if err != nil {
				return err
			}
			return runDeploy(cmd.Context(), dockerCLI, cmd.Flags(), &opts, config)
		},
		ValidArgsFunction:     completeNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringSliceVarP(&opts.composefiles, "compose-file", "c", []string{}, `Path to a Compose file, or "-" to read from stdin`)
	flags.SetAnnotation("compose-file", "version", []string{"1.25"})
	flags.BoolVar(&opts.sendRegistryAuth, "with-registry-auth", false, "Send registry authentication details to Swarm agents")
	flags.BoolVar(&opts.prune, "prune", false, "Prune services that are no longer referenced")
	flags.SetAnnotation("prune", "version", []string{"1.27"})
	flags.StringVar(&opts.resolveImage, "resolve-image", resolveImageAlways,
		`Query the registry to resolve image digest and supported platforms ("`+resolveImageAlways+`", "`+resolveImageChanged+`", "`+resolveImageNever+`")`)
	flags.SetAnnotation("resolve-image", "version", []string{"1.30"})
	flags.BoolVarP(&opts.detach, "detach", "d", true, "Exit immediately instead of waiting for the stack services to converge")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Suppress progress output")
	return cmd
}

// Resolve image constants
const (
	resolveImageAlways  = "always"
	resolveImageChanged = "changed"
	resolveImageNever   = "never"
)

const defaultNetworkDriver = "overlay"

// runDeploy is the swarm implementation of docker stack deploy
func runDeploy(ctx context.Context, dockerCLI command.Cli, flags *pflag.FlagSet, opts *deployOptions, cfg *composetypes.Config) error {
	switch opts.resolveImage {
	case resolveImageAlways, resolveImageChanged, resolveImageNever:
		// valid options.
	default:
		return fmt.Errorf("invalid option %s for flag --resolve-image", opts.resolveImage)
	}

	if opts.detach && !flags.Changed("detach") {
		_, _ = fmt.Fprintln(dockerCLI.Err(), "Since --detach=false was not specified, tasks will be created in the background.\n"+
			"In a future release, --detach=false will become the default.")
	}

	return deployCompose(ctx, dockerCLI, opts, cfg)
}

// checkDaemonIsSwarmManager does an Info API call to verify that the daemon is
// a swarm manager. This is necessary because we must create networks before we
// create services, but the API call for creating a network does not return a
// proper status code when it can't create a network in the "global" scope.
func checkDaemonIsSwarmManager(ctx context.Context, dockerCli command.Cli) error {
	res, err := dockerCli.Client().Info(ctx, client.InfoOptions{})
	if err != nil {
		return err
	}
	if !res.Info.Swarm.ControlAvailable {
		return errors.New(`this node is not a swarm manager. Use "docker swarm init" or "docker swarm join" to connect this node to swarm and try again`)
	}
	return nil
}

// pruneServices removes services that are no longer referenced in the source
func pruneServices(ctx context.Context, dockerCLI command.Cli, namespace convert.Namespace, services map[string]struct{}) {
	apiClient := dockerCLI.Client()

	oldServices, err := getStackServices(ctx, apiClient, namespace.Name())
	if err != nil {
		_, _ = fmt.Fprintln(dockerCLI.Err(), "Failed to list services:", err)
	}

	toRemove := make([]swarm.Service, 0, len(oldServices.Items))
	for _, service := range oldServices.Items {
		if _, exists := services[namespace.Descope(service.Spec.Name)]; !exists {
			toRemove = append(toRemove, service)
		}
	}
	removeServices(ctx, dockerCLI, toRemove)
}
