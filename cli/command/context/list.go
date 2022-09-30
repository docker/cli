package context

import (
	"io"
	"os"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/context/docker"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/docker/docker/client"
	"github.com/fvbommel/sortorder"
	"github.com/spf13/cobra"
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
			return runList(cmd, dockerCli, opts)
		},
		ValidArgsFunction: completion.NoComplete,
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.format, "format", "", flagsHelper.FormatHelp)
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Only show context names")
	return cmd
}

func runList(cmd *cobra.Command, dockerCli command.Cli, opts *listOptions) error {
	if opts.format == "" {
		opts.format = formatter.TableFormatKey
	}
	contextMap, err := dockerCli.ContextStore().List()
	if err != nil {
		return err
	}
	var (
		curContext = dockerCli.CurrentContext()
		curFound   bool
		contexts   []*formatter.ClientContext
	)
	for _, rawMeta := range contextMap {
		isCurrent := rawMeta.Name == curContext
		if isCurrent {
			curFound = true
		}
		meta, err := command.GetDockerContext(rawMeta)
		if err != nil {
			// Add a stub-entry to the list, including the error-message
			// indicating that the context couldn't be loaded.
			contexts = append(contexts, &formatter.ClientContext{
				Name:    rawMeta.Name,
				Current: isCurrent,
				Error:   err.Error(),
			})
			continue
		}
		var errMsg string
		dockerEndpoint, err := docker.EndpointFromContext(rawMeta)
		if err != nil {
			errMsg = err.Error()
		}
		desc := formatter.ClientContext{
			Name:           rawMeta.Name,
			Current:        isCurrent,
			Description:    meta.Description,
			DockerEndpoint: dockerEndpoint.Host,
			Error:          errMsg,
		}
		contexts = append(contexts, &desc)
	}
	if !curFound {
		// The currently specified context wasn't found. We add a stub-entry
		// to the list, including the error-message indicating that the context
		// wasn't found.
		var errMsg string
		_, err := dockerCli.ContextStore().GetMetadata(curContext)
		if err != nil {
			errMsg = err.Error()
		}
		contexts = append(contexts, &formatter.ClientContext{
			Name:    curContext,
			Current: true,
			Error:   errMsg,
		})
	}
	sort.Slice(contexts, func(i, j int) bool {
		return sortorder.NaturalLess(contexts[i].Name, contexts[j].Name)
	})
	if err := format(cmd.OutOrStdout(), opts, contexts); err != nil {
		return err
	}
	if os.Getenv(client.EnvOverrideHost) != "" {
		cmd.PrintErrf("Warning: %[1]s environment variable overrides the active context. "+
			"To use a context, either set the global --context flag, or unset %[1]s environment variable.\n", client.EnvOverrideHost)

		// Alternatives:
		// - dockerCli.PrintErrf(.....)
		// - dockerCli.Err().Printf(.....)
	}
	return nil
}

func format(output io.Writer, opts *listOptions, contexts []*formatter.ClientContext) error {
	contextCtx := formatter.Context{
		Output: output,
		Format: formatter.NewClientContextFormat(opts.format, opts.quiet),
	}
	return formatter.ClientContextWrite(contextCtx, contexts)
}
