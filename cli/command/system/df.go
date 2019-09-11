package system

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/spf13/cobra"
)

type diskUsageOptions struct {
	verbose bool
	format  string
}

// newDiskUsageCommand creates a new cobra.Command for `docker df`
func newDiskUsageCommand(dockerCli command.Cli) *cobra.Command {
	var opts diskUsageOptions

	cmd := &cobra.Command{
		Use:   "df [OPTIONS]",
		Short: "Show docker disk usage",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiskUsage(dockerCli, opts)
		},
		Annotations: map[string]string{"version": "1.25"},
	}

	flags := cmd.Flags()

	flags.BoolVarP(&opts.verbose, "verbose", "v", false, "Show detailed information on space usage")
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

	var bsz int64
	for _, bc := range du.BuildCache {
		if !bc.Shared {
			bsz += bc.Size
		}
	}

	duCtx := formatter.DiskUsageContext{
		Context: formatter.Context{
			Output: dockerCli.Out(),
			Format: formatter.NewDiskUsageFormat(format, opts.verbose),
		},
		LayersSize:  du.LayersSize,
		BuilderSize: bsz,
		BuildCache:  du.BuildCache,
		Images:      du.Images,
		Containers:  du.Containers,
		Volumes:     du.Volumes,
		Verbose:     opts.verbose,
	}

	return duCtx.Write()
}
