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

type initOptions struct {
	projectDir string
	force      bool
}

// NewInitCommand creates a new cobra.Command for `docker project init`
func NewInitCommand(dockerCli *command.DockerCli) *cobra.Command {
	var opts initOptions

	cmd := &cobra.Command{
		Use:   "init [ID]",
		Short: "Initiate Docker project",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ID := ""
			if len(args) == 1 {
				ID = args[0]
			}
			return runInit(dockerCli, &opts, ID)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.projectDir, "dir", "d", "", "Target directory (default is current directory)")
	flags.BoolVarP(&opts.force, "force", "f", false, "Force initialization over existing project")

	return cmd
}

func runInit(dockerCli *command.DockerCli, opts *initOptions, ID string) error {

	// directory where project should be initiated
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

	proj, err := project.Init(dir, ID, opts.force)
	if err != nil {
		return err
	}

	err = projectutil.SaveInRecentProjects(proj)
	if err != nil {
		return err
	}

	fmt.Fprintf(dockerCli.Out(), "project created at %s\n", dir)
	fmt.Fprintf(dockerCli.Out(), "to join the project: `docker project join %s`\n", proj.ID())

	return nil
}
