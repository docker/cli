package stack

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/kubernetes"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	"github.com/spf13/cobra"
)

func newDeployCommand(dockerCli command.Cli) *cobra.Command {
	var opts options.Deploy

	cmd := &cobra.Command{
		Use:     "deploy [OPTIONS] STACK",
		Aliases: []string{"up"},
		Short:   "Deploy a new stack or update an existing stack",
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Namespace = args[0]
			switch {
			case dockerCli.ClientInfo().HasAll():
				return errUnsupportedAllOrchestrator
			case dockerCli.ClientInfo().HasKubernetes():
				kli, err := kubernetes.WrapCli(dockerCli, kubernetes.NewOptions(cmd.Flags()))
				if err != nil {
					return err
				}
				return kubernetes.RunDeploy(kli, opts)
			default:
				return swarm.RunDeploy(dockerCli, opts)
			}
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
