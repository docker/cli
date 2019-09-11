package system

import (
	"bytes"
	"fmt"
	"sort"
	"text/template"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/builder"
	"github.com/docker/cli/cli/command/container"
	"github.com/docker/cli/cli/command/image"
	"github.com/docker/cli/cli/command/network"
	"github.com/docker/cli/cli/command/volume"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/go-units"
	"github.com/spf13/cobra"
	"vbom.ml/util/sortorder"
)

type pruneOptions struct {
	force           bool
	all             bool
	pruneVolumes    bool
	pruneBuildCache bool
	filter          opts.FilterOpt
}

// newPruneCommand creates a new cobra.Command for `docker prune`
func newPruneCommand(dockerCli command.Cli) *cobra.Command {
	options := pruneOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "prune [OPTIONS]",
		Short: "Remove unused data",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			options.pruneBuildCache = versions.GreaterThanOrEqualTo(dockerCli.Client().ClientVersion(), "1.31")
			return runPrune(dockerCli, options)
		},
		Annotations: map[string]string{"version": "1.25"},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.force, "force", "f", false, "Do not prompt for confirmation")
	flags.BoolVarP(&options.all, "all", "a", false, "Remove all unused images not just dangling ones")
	flags.BoolVar(&options.pruneVolumes, "volumes", false, "Prune volumes")
	flags.Var(&options.filter, "filter", "Provide filter values (e.g. 'label=<key>=<value>')")
	// "filter" flag is available in 1.28 (docker 17.04) and up
	flags.SetAnnotation("filter", "version", []string{"1.28"})

	return cmd
}

const confirmationTemplate = `WARNING! This will remove:
{{- range $_, $warning := .warnings }}
  - {{ $warning }}
{{- end }}
{{if .filters}}
  Items to be pruned will be filtered with:
{{- range $_, $filters := .filters }}
  - {{ $filters }}
{{- end }}
{{end}}
Are you sure you want to continue?`

func runPrune(dockerCli command.Cli, options pruneOptions) error {
	// TODO version this once "until" filter is supported for volumes
	if options.pruneVolumes && options.filter.Value().Contains("until") {
		return fmt.Errorf(`ERROR: The "until" filter is not supported with "--volumes"`)
	}
	if !options.force && !command.PromptForConfirmation(dockerCli.In(), dockerCli.Out(), confirmationMessage(dockerCli, options)) {
		return nil
	}
	pruneFuncs := []func(dockerCli command.Cli, all bool, filter opts.FilterOpt) (uint64, string, error){
		container.RunPrune,
		network.RunPrune,
	}
	if options.pruneVolumes {
		pruneFuncs = append(pruneFuncs, volume.RunPrune)
	}
	pruneFuncs = append(pruneFuncs, image.RunPrune)
	if options.pruneBuildCache {
		pruneFuncs = append(pruneFuncs, builder.CachePrune)
	}

	var spaceReclaimed uint64
	for _, pruneFn := range pruneFuncs {
		spc, output, err := pruneFn(dockerCli, options.all, options.filter)
		if err != nil {
			return err
		}
		spaceReclaimed += spc
		if output != "" {
			fmt.Fprintln(dockerCli.Out(), output)
		}
	}

	fmt.Fprintln(dockerCli.Out(), "Total reclaimed space:", units.HumanSize(float64(spaceReclaimed)))

	return nil
}

// confirmationMessage constructs a confirmation message that depends on the cli options.
func confirmationMessage(dockerCli command.Cli, options pruneOptions) string {
	t := template.Must(template.New("confirmation message").Parse(confirmationTemplate))

	warnings := []string{
		"all stopped containers",
		"all networks not used by at least one container",
	}
	if options.pruneVolumes {
		warnings = append(warnings, "all volumes not used by at least one container")
	}
	if options.all {
		warnings = append(warnings, "all images without at least one container associated to them")
	} else {
		warnings = append(warnings, "all dangling images")
	}
	if options.pruneBuildCache {
		if options.all {
			warnings = append(warnings, "all build cache")
		} else {
			warnings = append(warnings, "all dangling build cache")
		}
	}

	var filters []string
	pruneFilters := command.PruneFilters(dockerCli, options.filter.Value())
	if pruneFilters.Len() > 0 {
		// TODO remove fixed list of filters, and print all filters instead,
		// because the list of filters that is supported by the engine may evolve over time.
		for _, name := range []string{"label", "label!", "until"} {
			for _, v := range pruneFilters.Get(name) {
				filters = append(filters, name+"="+v)
			}
		}
		sort.Slice(filters, func(i, j int) bool {
			return sortorder.NaturalLess(filters[i], filters[j])
		})
	}

	var buffer bytes.Buffer
	t.Execute(&buffer, map[string][]string{"warnings": warnings, "filters": filters})
	return buffer.String()
}
