package project

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	projectutil "github.com/docker/cli/project/util"
	"github.com/spf13/cobra"
)

type lsOptions struct {
	json   bool
	quiet  bool
	format string
}

// NewLsCommand creates a new cobra.Command for `docker project ls`
func NewLsCommand(dockerCli *command.DockerCli) *cobra.Command {
	var opts lsOptions

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List recent projects",
		Args:  cli.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLs(dockerCli, &opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Only display volume names")
	flags.StringVar(&opts.format, "format", "", "Pretty-print volumes using a Go template")

	return cmd
}

func runLs(dockerCli *command.DockerCli, opts *lsOptions) error {
	projects := projectutil.GetRecentProjects()

	format := opts.format
	if len(format) == 0 {
		// TODO: allow project ls format to be defined in config

		// if len(dockerCli.ConfigFile().VolumesFormat) > 0 && !opts.quiet {
		// 	format = dockerCli.ConfigFile().VolumesFormat
		// } else {
		format = formatter.TableFormatKey
		// }
	}

	projectCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: formatter.NewProjectFormat(format, opts.quiet),
	}

	return formatter.ProjectWrite(projectCtx, projects)
}
