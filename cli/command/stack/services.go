package stack

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/kubernetes"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	cliopts "github.com/docker/cli/opts"
	"github.com/spf13/cobra"
)

func newServicesCommand(dockerCli command.Cli) *cobra.Command {
	opts := options.Services{Filter: cliopts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "services [OPTIONS] STACK",
		Short: "List the services in the stack",
		Args:  cli.ExactArgs(1),
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
				return kubernetes.RunServices(kli, opts)
			default:
				return swarm.RunServices(dockerCli, opts)
			}
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&opts.Quiet, "quiet", "q", false, "Only display IDs")
	flags.StringVar(&opts.Format, "format", "", "Pretty-print services using a Go template")
	flags.VarP(&opts.Filter, "filter", "f", "Filter output based on conditions provided")
	kubernetes.AddNamespaceFlag(flags)
	return cmd
}
