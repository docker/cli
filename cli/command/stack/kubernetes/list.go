package kubernetes

import (
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"vbom.ml/util/sortorder"
)

type listOptions struct {
	format string
}

func newListCommand(dockerCli command.Cli, kubeCli *kubeCli) *cobra.Command {
	opts := listOptions{}
	cmd := &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "List stacks",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(dockerCli, kubeCli, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.format, "format", "", "Pretty-print stacks using a Go template")

	return cmd
}

func runList(dockerCli command.Cli, kubeCli *kubeCli, opts listOptions) error {
	stacks, err := getStacks(kubeCli)
	if err != nil {
		return err
	}
	format := opts.format
	if len(format) == 0 {
		format = formatter.TableFormatKey
	}
	stackCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: formatter.NewStackFormat(format),
	}
	sort.Sort(byName(stacks))
	return formatter.StackWrite(stackCtx, stacks)
}

type byName []*formatter.Stack

func (n byName) Len() int           { return len(n) }
func (n byName) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n byName) Less(i, j int) bool { return sortorder.NaturalLess(n[i].Name, n[j].Name) }

func getStacks(kubeCli *kubeCli) ([]*formatter.Stack, error) {
	stackSvc, err := kubeCli.Stacks()
	if err != nil {
		return nil, err
	}

	stacks, err := stackSvc.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var formattedStacks []*formatter.Stack
	for _, stack := range stacks.Items {
		services, err := getServices(stack.Spec.ComposeFile)
		if err != nil {
			return nil, err
		}
		formattedStacks = append(formattedStacks, &formatter.Stack{
			Name:     stack.Name,
			Services: len(services),
		})
	}
	return formattedStacks, nil
}
