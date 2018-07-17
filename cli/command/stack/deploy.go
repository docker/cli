package stack

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/kubernetes"
	"github.com/docker/cli/cli/command/stack/loader"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newDeployCommand(dockerCli command.Cli, common *commonOptions) *cobra.Command {
	var opts options.Deploy

	cmd := &cobra.Command{
		Use:     "deploy [OPTIONS] STACK",
		Aliases: []string{"up"},
		Short:   "Deploy a new stack or update an existing stack",
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Namespace = args[0]
			if err := validateStackName(opts.Namespace); err != nil {
				return err
			}

			commonOrchestrator := command.OrchestratorSwarm // default for top-level deploy command
			if common != nil {
				commonOrchestrator = common.orchestrator
			}

			switch {
			case opts.Bundlefile == "" && len(opts.Composefiles) == 0:
				return errors.Errorf("Please specify either a bundle file (with --bundle-file) or a Compose file (with --compose-file).")
			case opts.Bundlefile != "" && len(opts.Composefiles) != 0:
				return errors.Errorf("You cannot specify both a bundle file and a Compose file.")
			case opts.Bundlefile != "":
				if commonOrchestrator != command.OrchestratorSwarm {
					return errors.Errorf("bundle files are not supported on another orchestrator than swarm.")
				}
				return swarm.DeployBundle(context.Background(), dockerCli, opts)
			}

			config, err := loader.LoadComposefile(dockerCli, opts)
			if err != nil {
				return err
			}
			return RunDeploy(dockerCli, cmd.Flags(), config, commonOrchestrator, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.Bundlefile, "bundle-file", "", "Path to a Distributed Application Bundle file")
	flags.SetAnnotation("bundle-file", "experimental", nil)
	flags.SetAnnotation("bundle-file", "swarm", nil)
	flags.StringSliceVarP(&opts.Composefiles, "compose-file", "c", []string{}, "Path to a Compose file")
	flags.SetAnnotation("compose-file", "version", []string{"1.25"})
	flags.BoolVar(&opts.SendRegistryAuth, "with-registry-auth", false, "Send registry authentication details to Swarm agents")
	flags.SetAnnotation("with-registry-auth", "swarm", nil)
	flags.BoolVar(&opts.Prune, "prune", false, "Prune services that are no longer referenced")
	flags.SetAnnotation("prune", "version", []string{"1.27"})
	flags.SetAnnotation("prune", "swarm", nil)
	flags.StringVar(&opts.ResolveImage, "resolve-image", swarm.ResolveImageAlways,
		`Query the registry to resolve image digest and supported platforms ("`+swarm.ResolveImageAlways+`"|"`+swarm.ResolveImageChanged+`"|"`+swarm.ResolveImageNever+`")`)
	flags.SetAnnotation("resolve-image", "version", []string{"1.30"})
	flags.SetAnnotation("resolve-image", "swarm", nil)
	kubernetes.AddNamespaceFlag(flags)
	return cmd
}

// RunDeploy performs a stack deploy against the specified orchestrator
func RunDeploy(dockerCli command.Cli, flags *pflag.FlagSet, config *composetypes.Config, commonOrchestrator command.Orchestrator, opts options.Deploy) error {
	switch {
	case commonOrchestrator.HasAll():
		return errUnsupportedAllOrchestrator
	case commonOrchestrator.HasKubernetes():
		kli, err := kubernetes.WrapCli(dockerCli, kubernetes.NewOptions(flags, commonOrchestrator))
		if err != nil {
			return errors.Wrap(err, "unable to deploy to Kubernetes")
		}
		return kubernetes.RunDeploy(kli, opts, config)
	default:
		return swarm.RunDeploy(dockerCli, opts, config)
	}
}
