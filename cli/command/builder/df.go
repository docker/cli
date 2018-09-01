package builder

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/spf13/cobra"
)

type diskUsageOptions struct {
	format string
}

// NewDiskUsageCommand creates a new cobra.Command for `docker df`
func NewDiskUsageCommand(dockerCli command.Cli) *cobra.Command {
	var opts diskUsageOptions

	cmd := &cobra.Command{
		Use:   "df [OPTIONS]",
		Short: "Show build cache disk usage",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiskUsage(dockerCli, opts)
		},
		Annotations: map[string]string{"version": "1.39"},
	}

	flags := cmd.Flags()

	flags.StringVar(&opts.format, "format", "", "Pretty-print images using a Go template")

	return cmd
}

func runDiskUsage(dockerCli command.Cli, opts diskUsageOptions) error {
	du, err := dockerCli.Client().DiskUsage(context.Background())
	if err != nil {
		return err
	}

	format := opts.format
	if len(format) == 0 {
		format = formatter.TableFormatKey
	}

	ctx := formatter.Context{
		Output: dockerCli.Out(),
		Format: formatter.NewBuildCacheFormat(format, false),
		Trunc:  format == formatter.TableFormatKey,
	}
	return formatter.BuildCacheWrite(ctx, du.BuildCache)
}
