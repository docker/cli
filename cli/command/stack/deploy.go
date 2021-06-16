package stack

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/loader"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	composetypes "github.com/docker/cli/cli/compose/types"
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
			config, err := loader.LoadComposefile(dockerCli, opts)
			if err != nil {
				return err
			}
			return RunDeploy(dockerCli, cmd.Flags(), config, common.Orchestrator(), opts)
		},
	}

	flags := cmd.Flags()
	flags.StringSliceVarP(&opts.Composefiles, "compose-file", "c", []string{}, `Path to a Compose file, or "-" to read from stdin`)
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
	return cmd
}

// RunDeploy performs a stack deploy against the specified orchestrator
func RunDeploy(dockerCli command.Cli, flags *pflag.FlagSet, config *composetypes.Config, commonOrchestrator command.Orchestrator, opts options.Deploy) error {
	return swarm.RunDeploy(dockerCli, opts, config)
}
