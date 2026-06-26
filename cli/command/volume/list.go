package volume

import (
	"context"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/docker/cli/opts"
	"github.com/fvbommel/sortorder"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

const (
	clusterTableFormat = "table {{.Name}}\t{{.Group}}\t{{.Driver}}\t{{.Availability}}\t{{.Status}}"
)

type listOptions struct {
	quiet   bool
	format  string
	cluster bool
	filter  opts.FilterOpt
}

func newListCommand(dockerCLI command.Cli) *cobra.Command {
	options := listOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Aliases: []string{"list"},
		Short:   "List volumes",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), dockerCLI, options)
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only display volume names")
	flags.StringVar(&options.format, "format", "", flagsHelper.FormatHelp)
	flags.VarP(&options.filter, "filter", "f", `Provide filter values (e.g. "dangling=true")`)
	flags.BoolVar(&options.cluster, "cluster", false, "Display only cluster volumes, and use cluster volume list formatting")
	_ = flags.SetAnnotation("cluster", "version", []string{"1.42"})
	_ = flags.SetAnnotation("cluster", "swarm", []string{"manager"})

	return cmd
}

func runList(ctx context.Context, dockerCLI command.Cli, options listOptions) error {
	apiClient := dockerCLI.Client()
	res, err := apiClient.VolumeList(ctx, client.VolumeListOptions{Filters: options.filter.Value()})
	if err != nil {
		return err
	}

	format := options.format
	if len(format) == 0 && !options.cluster {
		if len(dockerCLI.ConfigFile().VolumesFormat) > 0 && !options.quiet {
			format = dockerCLI.ConfigFile().VolumesFormat
		} else {
			format = formatter.TableFormatKey
		}
	} else if options.cluster {
		// TODO(dperny): write server-side filter for cluster volumes. For this
		// proof of concept, we'll just filter out non-cluster volumes here

		// trick for filtering in place
		n := 0
		for _, vol := range res.Items {
			if vol.ClusterVolume != nil {
				res.Items[n] = vol
				n++
			}
		}
		res.Items = res.Items[:n]
		if !options.quiet {
			format = clusterTableFormat
		} else {
			format = formatter.TableFormatKey
		}
	}

	sort.Slice(res.Items, func(i, j int) bool {
		return sortorder.NaturalLess(res.Items[i].Name, res.Items[j].Name)
	})

	volumeCtx := formatter.Context{
		Output: dockerCLI.Out(),
		Format: formatter.NewVolumeFormat(format, options.quiet),
	}
	return formatter.VolumeWrite(volumeCtx, res.Items)
}
