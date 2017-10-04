package secret

import (
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"vbom.ml/util/sortorder"
)

type bySecretName []swarm.Secret

func (r bySecretName) Len() int      { return len(r) }
func (r bySecretName) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r bySecretName) Less(i, j int) bool {
	return sortorder.NaturalLess(r[i].Spec.Name, r[j].Spec.Name)
}

type listOptions struct {
	quiet  bool
	format string
	filter opts.FilterOpt
}

func newSecretListCommand(dockerCli command.Cli) *cobra.Command {
	options := listOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Aliases: []string{"list"},
		Short:   "List secrets",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSecretList(dockerCli, options)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only display IDs")
	flags.StringVarP(&options.format, "format", "", "", "Pretty-print secrets using a Go template")
	flags.VarP(&options.filter, "filter", "f", "Filter output based on conditions provided")

	return cmd
}

func runSecretList(dockerCli command.Cli, options listOptions) error {
	client := dockerCli.Client()
	ctx := context.Background()

	secrets, err := client.SecretList(ctx, types.SecretListOptions{Filters: options.filter.Value()})
	if err != nil {
		return err
	}
	format := options.format
	if len(format) == 0 {
		if len(dockerCli.ConfigFile().SecretFormat) > 0 && !options.quiet {
			format = dockerCli.ConfigFile().SecretFormat
		} else {
			format = formatter.TableFormatKey
		}
	}

	sort.Sort(bySecretName(secrets))

	secretCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: formatter.NewSecretFormat(format, options.quiet),
	}
	return formatter.SecretWrite(secretCtx, secrets)
}
