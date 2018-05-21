package system

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/container"
	"github.com/docker/cli/cli/command/image"
	"github.com/docker/cli/cli/command/network"
	"github.com/docker/cli/cli/command/volume"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/versions"
	units "github.com/docker/go-units"
	"github.com/spf13/cobra"
)

type pruneOptions struct {
	force           bool
	all             bool
	pruneBuildCache bool
	pruneVolumes    bool
	filter          opts.FilterOpt
}

// newPruneCommand creates a new cobra.Command for `docker prune`
func newPruneCommand(dockerCli command.Cli) *cobra.Command {
	options := pruneOptions{filter: opts.NewFilterOpt(), pruneBuildCache: true}

	cmd := &cobra.Command{
		Use:   "prune [OPTIONS]",
		Short: "Remove unused data",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
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
{{- range $_, $warning := . }}
        - {{ $warning }}
{{- end }}
Are you sure you want to continue?`

// runBuildCachePrune executes a prune command for build cache
func runBuildCachePrune(dockerCli command.Cli, _ opts.FilterOpt) (uint64, string, error) {
	report, err := dockerCli.Client().BuildCachePrune(context.Background())
	if err != nil {
		return 0, "", err
	}
	return report.SpaceReclaimed, "", nil
}

func runPrune(dockerCli command.Cli, options pruneOptions) error {
	// TODO version this once "until" filter is supported for volumes
	if options.pruneVolumes && options.filter.Value().Include("until") {
		return fmt.Errorf(`ERROR: The "until" filter is not supported with "--volumes"`)
	}
	if versions.LessThan(dockerCli.Client().ClientVersion(), "1.31") {
		options.pruneBuildCache = false
	}
	if !options.force && !command.PromptForConfirmation(dockerCli.In(), dockerCli.Out(), confirmationMessage(options)) {
		return nil
	}
	imagePrune := func(dockerCli command.Cli, filter opts.FilterOpt) (uint64, string, error) {
		return image.RunPrune(dockerCli, options.all, options.filter)
	}
	pruneFuncs := []func(dockerCli command.Cli, filter opts.FilterOpt) (uint64, string, error){
		container.RunPrune,
		network.RunPrune,
	}
	if options.pruneVolumes {
		pruneFuncs = append(pruneFuncs, volume.RunPrune)
	}
	pruneFuncs = append(pruneFuncs, imagePrune)
	if options.pruneBuildCache {
		pruneFuncs = append(pruneFuncs, runBuildCachePrune)
	}

	var spaceReclaimed uint64
	for _, pruneFn := range pruneFuncs {
		spc, output, err := pruneFn(dockerCli, options.filter)
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
func confirmationMessage(options pruneOptions) string {
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
		warnings = append(warnings, "all build cache")
	}
	if len(options.filter.String()) > 0 {
		warnings = append(warnings, "Elements to be pruned will be filtered with:")
		warnings = append(warnings, "label="+options.filter.String())
	}

	var buffer bytes.Buffer
	t.Execute(&buffer, &warnings)
	return buffer.String()
}
