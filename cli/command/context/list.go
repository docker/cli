package context

import (
	"fmt"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/context/docker"
	kubecontext "github.com/docker/cli/cli/context/kubernetes"
	"github.com/spf13/cobra"
	"vbom.ml/util/sortorder"
)

type listOptions struct {
	format string
	quiet  bool
}

func newListCommand(dockerCli command.Cli) *cobra.Command {
	opts := &listOptions{}
	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Aliases: []string{"list"},
		Short:   "List contexts",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(dockerCli, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.format, "format", "", "Pretty-print contexts using a Go template")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Only show context names")
	return cmd
}

func runList(dockerCli command.Cli, opts *listOptions) error {
	if opts.format == "" {
		opts.format = formatter.TableFormatKey
	}
	curContext := dockerCli.CurrentContext()
	contextMap, err := dockerCli.ContextStore().ListContexts()
	if err != nil {
		return err
	}
	var contexts []*formatter.ClientContext
	for _, rawMeta := range contextMap {
		meta, err := command.GetDockerContext(rawMeta)
		if err != nil {
			return err
		}
		dockerEndpoint, err := docker.EndpointFromContext(rawMeta)
		if err != nil {
			return err
		}
		kubernetesEndpoint := kubecontext.EndpointFromContext(rawMeta)
		kubEndpointText := ""
		if kubernetesEndpoint != nil {
			kubEndpointText = fmt.Sprintf("%s (%s)", kubernetesEndpoint.Host, kubernetesEndpoint.DefaultNamespace)
		}
		if rawMeta.Name == command.DefaultContextName {
			meta.Description = "Current DOCKER_HOST based configuration"
		}
		desc := formatter.ClientContext{
			Name:               rawMeta.Name,
			Current:            rawMeta.Name == curContext,
			Description:        meta.Description,
			StackOrchestrator:  string(meta.StackOrchestrator),
			DockerEndpoint:     dockerEndpoint.Host,
			KubernetesEndpoint: kubEndpointText,
		}
		contexts = append(contexts, &desc)
	}
	sort.Slice(contexts, func(i, j int) bool {
		return sortorder.NaturalLess(contexts[i].Name, contexts[j].Name)
	})
	return format(dockerCli, opts, contexts)
}

func format(dockerCli command.Cli, opts *listOptions, contexts []*formatter.ClientContext) error {
	contextCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: formatter.NewClientContextFormat(opts.format, opts.quiet),
	}
	return formatter.ClientContextWrite(contextCtx, contexts)
}
