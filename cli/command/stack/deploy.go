package stack

import (
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/compose/convert"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/versions"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

const (
	defaultNetworkDriver = "overlay"
	resolveImageAlways   = "always"
	resolveImageChanged  = "changed"
	resolveImageNever    = "never"
)

type deployOptions struct {
	bundlefile       string
	composefile      string
	namespace        string
	resolveImage     string
	sendRegistryAuth bool
	prune            bool
}

func newDeployCommand(dockerCli command.Cli) *cobra.Command {
	var opts deployOptions

	cmd := &cobra.Command{
		Use:     "deploy [OPTIONS] STACK",
		Aliases: []string{"up"},
		Short:   "Deploy a new stack or update an existing stack",
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.namespace = args[0]
			return runDeploy(dockerCli, opts)
		},
	}

	flags := cmd.Flags()
	addBundlefileFlag(&opts.bundlefile, flags)
	addComposefileFlag(&opts.composefile, flags)
	addRegistryAuthFlag(&opts.sendRegistryAuth, flags)
	flags.BoolVar(&opts.prune, "prune", false, "Prune services that are no longer referenced")
	flags.SetAnnotation("prune", "version", []string{"1.27"})
	flags.StringVar(&opts.resolveImage, "resolve-image", resolveImageAlways,
		`Query the registry to resolve image digest and supported platforms ("`+resolveImageAlways+`"|"`+resolveImageChanged+`"|"`+resolveImageNever+`")`)
	flags.SetAnnotation("resolve-image", "version", []string{"1.30"})
	return cmd
}

func runDeploy(dockerCli command.Cli, opts deployOptions) error {
	ctx := context.Background()

	if err := validateResolveImageFlag(dockerCli, &opts); err != nil {
		return err
	}

	switch {
	case opts.bundlefile == "" && opts.composefile == "":
		return errors.Errorf("Please specify either a bundle file (with --bundle-file) or a Compose file (with --compose-file).")
	case opts.bundlefile != "" && opts.composefile != "":
		return errors.Errorf("You cannot specify both a bundle file and a Compose file.")
	case opts.bundlefile != "":
		return deployBundle(ctx, dockerCli, opts)
	default:
		return deployCompose(ctx, dockerCli, opts)
	}
}

// validateResolveImageFlag validates the opts.resolveImage command line option
// and also turns image resolution off if the version is older than 1.30
func validateResolveImageFlag(dockerCli command.Cli, opts *deployOptions) error {
	if opts.resolveImage != resolveImageAlways && opts.resolveImage != resolveImageChanged && opts.resolveImage != resolveImageNever {
		return errors.Errorf("Invalid option %s for flag --resolve-image", opts.resolveImage)
	}
	// client side image resolution should not be done when the supported
	// server version is older than 1.30
	if versions.LessThan(dockerCli.Client().ClientVersion(), "1.30") {
		opts.resolveImage = resolveImageNever
	}
	return nil
}

// checkDaemonIsSwarmManager does an Info API call to verify that the daemon is
// a swarm manager. This is necessary because we must create networks before we
// create services, but the API call for creating a network does not return a
// proper status code when it can't create a network in the "global" scope.
func checkDaemonIsSwarmManager(ctx context.Context, dockerCli command.Cli) error {
	info, err := dockerCli.Client().Info(ctx)
	if err != nil {
		return err
	}
	if !info.Swarm.ControlAvailable {
		return errors.New("this node is not a swarm manager. Use \"docker swarm init\" or \"docker swarm join\" to connect this node to swarm and try again")
	}
	return nil
}

// pruneServices removes services that are no longer referenced in the source
func pruneServices(ctx context.Context, dockerCli command.Cli, namespace convert.Namespace, services map[string]struct{}) {
	client := dockerCli.Client()

	oldServices, err := getServices(ctx, client, namespace.Name())
	if err != nil {
		fmt.Fprintf(dockerCli.Err(), "Failed to list services: %s", err)
	}

	pruneServices := []swarm.Service{}
	for _, service := range oldServices {
		if _, exists := services[namespace.Descope(service.Spec.Name)]; !exists {
			pruneServices = append(pruneServices, service)
		}
	}
	removeServices(ctx, dockerCli, pruneServices)
}
