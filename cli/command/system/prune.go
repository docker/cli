// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.24

package system

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"text/template"

	"github.com/containerd/errdefs"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/system/pruner"
	"github.com/docker/cli/internal/prompt"
	"github.com/docker/cli/opts"
	"github.com/docker/go-units"
	"github.com/fvbommel/sortorder"
	"github.com/spf13/cobra"
)

type pruneOptions struct {
	force        bool
	all          bool
	pruneVolumes bool
	filter       opts.FilterOpt
}

// newPruneCommand creates a new cobra.Command for `docker prune`
func newPruneCommand(dockerCLI command.Cli) *cobra.Command {
	options := pruneOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "prune [OPTIONS]",
		Short: "Remove unused data",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPrune(cmd.Context(), dockerCLI, options)
		},
		Annotations:           map[string]string{"version": "1.25"},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.force, "force", "f", false, "Do not prompt for confirmation")
	flags.BoolVarP(&options.all, "all", "a", false, "Remove all unused images not just dangling ones")
	flags.BoolVar(&options.pruneVolumes, "volumes", false, "Prune anonymous volumes")
	flags.Var(&options.filter, "filter", `Provide filter values (e.g. "label=<key>=<value>")`)
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

func runPrune(ctx context.Context, dockerCli command.Cli, options pruneOptions) error {
	// prune requires either force, or a user to confirm after prompting.
	confirmed := options.force

	// Validate the given options for each pruner and construct a confirmation-message.
	confirmationMessage, err := dryRun(ctx, dockerCli, options)
	if err != nil {
		return err
	}
	if !confirmed {
		var err error
		confirmed, err = prompt.Confirm(ctx, dockerCli.In(), dockerCli.Out(), confirmationMessage)
		if err != nil {
			return err
		}
		if !confirmed {
			return cancelledErr{errors.New("system prune has been cancelled")}
		}
	}

	var spaceReclaimed uint64
	for contentType, pruneFn := range pruner.List() {
		switch contentType {
		case pruner.TypeVolume:
			if !options.pruneVolumes {
				continue
			}
		case pruner.TypeContainer, pruner.TypeNetwork, pruner.TypeImage, pruner.TypeBuildCache:
			// no special handling; keeping the "exhaustive" linter happy.
		default:
			// other pruners; no special handling; keeping the "exhaustive" linter happy.
		}

		spc, output, err := pruneFn(ctx, dockerCli, pruner.PruneOptions{
			Confirmed: confirmed,
			All:       options.all,
			Filter:    options.filter,
		})
		if err != nil && !errdefs.IsNotImplemented(err) {
			return err
		}
		spaceReclaimed += spc
		if output != "" {
			_, _ = fmt.Fprintln(dockerCli.Out(), output)
		}
	}

	_, _ = fmt.Fprintln(dockerCli.Out(), "Total reclaimed space:", units.HumanSize(float64(spaceReclaimed)))

	return nil
}

type cancelledErr struct{ error }

func (cancelledErr) Cancelled() {}

// dryRun validates the given options for each prune-function and constructs
// a confirmation message that depends on the cli options.
func dryRun(ctx context.Context, dockerCli command.Cli, options pruneOptions) (string, error) {
	var (
		errs     []error
		warnings []string
	)
	for contentType, pruneFn := range pruner.List() {
		switch contentType {
		case pruner.TypeVolume:
			if !options.pruneVolumes {
				continue
			}
		case pruner.TypeContainer, pruner.TypeNetwork, pruner.TypeImage, pruner.TypeBuildCache:
			// no special handling; keeping the "exhaustive" linter happy.
		default:
			// other pruners; no special handling; keeping the "exhaustive" linter happy.
		}
		// Always run with "[pruner.PruneOptions.Confirmed] = false"
		// to perform validation of the given options and produce
		// a confirmation message for the pruner.
		_, confirmMsg, err := pruneFn(ctx, dockerCli, pruner.PruneOptions{
			All:    options.all,
			Filter: options.filter,
		})
		// A "canceled" error is expected in dry-run mode; any other error
		// must be returned as a "fatal" error.
		if err != nil && !errdefs.IsCanceled(err) && !errdefs.IsNotImplemented(err) {
			errs = append(errs, err)
		}
		if confirmMsg != "" {
			warnings = append(warnings, confirmMsg)
		}
	}
	if len(errs) > 0 {
		return "", errors.Join(errs...)
	}

	var filters []string
	pruneFilters := command.PruneFilters(dockerCli, options.filter.Value())
	if len(pruneFilters) > 0 {
		// TODO remove fixed list of filters, and print all filters instead,
		// because the list of filters that is supported by the engine may evolve over time.
		for _, name := range []string{"label", "label!", "until"} {
			for v := range pruneFilters[name] {
				filters = append(filters, name+"="+v)
			}
		}
		sort.Slice(filters, func(i, j int) bool {
			return sortorder.NaturalLess(filters[i], filters[j])
		})
	}

	var buffer bytes.Buffer
	t := template.Must(template.New("confirmation message").Parse(confirmationTemplate))
	_ = t.Execute(&buffer, map[string][]string{"warnings": warnings, "filters": filters})
	return buffer.String(), nil
}
