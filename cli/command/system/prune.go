package system

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/prune"
	"github.com/docker/cli/opts"
	units "github.com/docker/go-units"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

type pruneOptions struct {
	force        bool
	all          bool
	pruneVolumes bool
	filter       opts.FilterOpt
}

// NewPruneCommand creates a new cobra.Command for `docker prune`
func NewPruneCommand(dockerCli command.Cli) *cobra.Command {
	options := pruneOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "prune [OPTIONS]",
		Short: "Remove unused data",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPrune(dockerCli, options)
		},
		Tags: map[string]string{"version": "1.25"},
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

func runPrune(dockerCli command.Cli, options pruneOptions) error {
	if !options.force && !command.PromptForConfirmation(dockerCli.In(), dockerCli.Out(), confirmationMessage(options)) {
		return nil
	}

	var spaceReclaimed uint64
	pruneFuncs := []func(dockerCli command.Cli, filter opts.FilterOpt) (uint64, string, error){
		prune.RunContainerPrune,
		prune.RunNetworkPrune,
	}
	if options.pruneVolumes {
		pruneFuncs = append(pruneFuncs, prune.RunVolumePrune)
	}

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

	spc, output, err := prune.RunImagePrune(dockerCli, options.all, options.filter)
	if err != nil {
		return err
	}
	if spc > 0 {
		spaceReclaimed += spc
		fmt.Fprintln(dockerCli.Out(), output)
	}

	report, err := dockerCli.Client().BuildCachePrune(context.Background())
	if err != nil {
		return err
	}
	spaceReclaimed += report.SpaceReclaimed

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
	warnings = append(warnings, "all build cache")

	var buffer bytes.Buffer
	t.Execute(&buffer, &warnings)
	return buffer.String()
}
