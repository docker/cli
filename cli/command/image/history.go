package image

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/spf13/cobra"
)

type historyOptions struct {
	image string

	human   bool
	quiet   bool
	noTrunc bool
	format  string
}

// NewHistoryCommand creates a new `docker history` command
func NewHistoryCommand(dockerCli command.Cli) *cobra.Command {
	var opts historyOptions

	cmd := &cobra.Command{
		Use:   "history [OPTIONS] IMAGE",
		Short: "Show the history of an image",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.image = args[0]
			return runHistory(cmd.Context(), dockerCli, opts)
		},
		Annotations: map[string]string{
			"aliases": "docker image history, docker history",
		},
	}

	flags := cmd.Flags()

	flags.BoolVarP(&opts.human, "human", "H", true, "Print sizes and dates in human readable format")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Only show image IDs")
	flags.BoolVar(&opts.noTrunc, "no-trunc", false, "Don't truncate output")
	flags.StringVar(&opts.format, "format", "", flagsHelper.FormatHelp)

	return cmd
}

func runHistory(ctx context.Context, dockerCli command.Cli, opts historyOptions) error {
	history, err := dockerCli.Client().ImageHistory(ctx, opts.image)
	if err != nil {
		return err
	}

	format := opts.format
	if len(format) == 0 {
		format = formatter.TableFormatKey
	}

	historyCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: NewHistoryFormat(format, opts.quiet, opts.human),
		Trunc:  !opts.noTrunc,
	}
	return HistoryWrite(historyCtx, opts.human, history)
}
