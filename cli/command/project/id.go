package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	project "github.com/docker/cli/project/impl"
	"github.com/spf13/cobra"
)

type idOptions struct {
	projectDir string
}

// NewIDCommand returns a command that can be used to display current project id
func NewIDCommand(dockerCli *command.DockerCli) *cobra.Command {
	var opts idOptions

	cmd := &cobra.Command{
		Use:   "id",
		Short: "Display Docker project ID",
		Args:  cli.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runID(dockerCli, &opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.projectDir, "dir", "d", "", "Target directory (default is current directory)")

	return cmd
}

func runID(dockerCli *command.DockerCli, opts *idOptions) error {
	// directory from where project should be left
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	if opts.projectDir != "" {
		if filepath.IsAbs(opts.projectDir) {
			dir = opts.projectDir
		} else {
			dir = filepath.Clean(filepath.Join(dir, opts.projectDir))
		}
	}
	proj, err := project.Load(dir)
	if err != nil {
		return err
	}
	if proj == nil {
		fmt.Fprintf(dockerCli.Out(), "no project found at %s\n", dir)
		return nil
	}
	fmt.Fprintf(dockerCli.Out(), "%s\n", proj.ID())
	return nil
}
