package stack

import (
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/stack/kubernetes"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	"github.com/spf13/cobra"
	"vbom.ml/util/sortorder"
)

func newListCommand(dockerCli command.Cli) *cobra.Command {
	opts := options.List{}

	cmd := &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "List stacks",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, dockerCli, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.Format, "format", "", "Pretty-print stacks using a Go template")
	flags.StringSliceVar(&opts.Namespaces, "namespace", []string{}, "Kubernetes namespaces to use")
	flags.SetAnnotation("namespace", "kubernetes", nil)
	flags.BoolVarP(&opts.AllNamespaces, "all-namespaces", "", false, "List stacks from all Kubernetes namespaces")
	flags.SetAnnotation("all-namespaces", "kubernetes", nil)
	return cmd
}

func runList(cmd *cobra.Command, dockerCli command.Cli, opts options.List) error {
	stacks := []*formatter.Stack{}
	if dockerCli.ClientInfo().HasSwarm() {
		ss, err := swarm.GetStacks(dockerCli)
		if err != nil {
			return err
		}
		stacks = append(stacks, ss...)
	}
	if dockerCli.ClientInfo().HasKubernetes() {
		kubeCli, err := kubernetes.WrapCli(dockerCli, kubernetes.NewOptions(cmd.Flags()))
		if err != nil {
			return err
		}
		ss, err := kubernetes.GetStacks(kubeCli, opts)
		if err != nil {
			return err
		}
		stacks = append(stacks, ss...)
	}
	return format(dockerCli, opts, stacks)
}

func format(dockerCli command.Cli, opts options.List, stacks []*formatter.Stack) error {
	format := opts.Format
	if format == "" || format == formatter.TableFormatKey {
		format = formatter.SwarmStackTableFormat
		if dockerCli.ClientInfo().HasKubernetes() {
			format = formatter.KubernetesStackTableFormat
		}
	}
	stackCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: formatter.Format(format),
	}
	sort.Slice(stacks, func(i, j int) bool {
		return sortorder.NaturalLess(stacks[i].Name, stacks[j].Name) ||
			!sortorder.NaturalLess(stacks[j].Name, stacks[i].Name) &&
				sortorder.NaturalLess(stacks[j].Namespace, stacks[i].Namespace)
	})
	return formatter.StackWrite(stackCtx, stacks)
}
