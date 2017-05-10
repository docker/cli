package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	project "github.com/docker/cli/project/impl"
	projectutil "github.com/docker/cli/project/util"
	"github.com/spf13/cobra"
)

type leaveOptions struct {
	projectDir string
}

// NewLeaveCommand removes the project id
func NewLeaveCommand(dockerCli *command.DockerCli) *cobra.Command {
	var opts leaveOptions

	cmd := &cobra.Command{
		Use:   "leave",
		Short: "Leave a Docker project",
		Args:  cli.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLeave(dockerCli, &opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.projectDir, "dir", "d", "", "Target directory (default is current directory)")

	return cmd
}

func runLeave(dockerCli *command.DockerCli, opts *leaveOptions) error {
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
		// still try to remove it from recent projects considering given
		// dir as project root directory
		err = projectutil.RemoveFromRecentProjects(dir)
		if err == nil {
			fmt.Fprintf(dockerCli.Out(), "(removed from recent projects though)\n")
		}
		// don't do anything is err != nil
		return nil
	}

	_ = projectutil.RemoveFromRecentProjects(proj.RootDir())

	projectID := proj.ID()

	err = proj.Leave()
	if err != nil {
		return err
	}

	fmt.Fprintf(dockerCli.Out(), "left project %s\n", projectID)

	return nil
}
