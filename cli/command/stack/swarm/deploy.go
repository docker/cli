package swarm

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/compose/convert"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/versions"
	"github.com/pkg/errors"
)

// Resolve image constants
const (
	defaultNetworkDriver = "overlay"
	ResolveImageAlways   = "always"
	ResolveImageChanged  = "changed"
	ResolveImageNever    = "never"
)

// RunDeploy is the swarm implementation of docker stack deploy
func RunDeploy(dockerCli command.Cli, opts options.Deploy) error {
	ctx := context.Background()

	if err := validateStackName(opts.Namespace); err != nil {
		return err
	}
	if err := validateResolveImageFlag(dockerCli, &opts); err != nil {
		return err
	}

	switch {
	case opts.Bundlefile == "" && len(opts.Composefiles) == 0:
		return errors.Errorf("Please specify either a bundle file (with --bundle-file) or a Compose file (with --compose-file).")
	case opts.Bundlefile != "" && len(opts.Composefiles) != 0:
		return errors.Errorf("You cannot specify both a bundle file and a Compose file.")
	case opts.Bundlefile != "":
		return deployBundle(ctx, dockerCli, opts)
	default:
		return deployCompose(ctx, dockerCli, opts)
	}
}

// validateResolveImageFlag validates the opts.resolveImage command line option
// and also turns image resolution off if the version is older than 1.30
func validateResolveImageFlag(dockerCli command.Cli, opts *options.Deploy) error {
	if opts.ResolveImage != ResolveImageAlways && opts.ResolveImage != ResolveImageChanged && opts.ResolveImage != ResolveImageNever {
		return errors.Errorf("Invalid option %s for flag --resolve-image", opts.ResolveImage)
	}
	// client side image resolution should not be done when the supported
	// server version is older than 1.30
	if versions.LessThan(dockerCli.Client().ClientVersion(), "1.30") {
		opts.ResolveImage = ResolveImageNever
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

	oldServices, err := getStackServices(ctx, client, namespace.Name())
	if err != nil {
		fmt.Fprintf(dockerCli.Err(), "Failed to list services: %s\n", err)
	}

	pruneServices := []swarm.Service{}
	for _, service := range oldServices {
		if _, exists := services[namespace.Descope(service.Spec.Name)]; !exists {
			pruneServices = append(pruneServices, service)
		}
	}
	removeServices(ctx, dockerCli, pruneServices)
}
