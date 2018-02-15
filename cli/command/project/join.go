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

type joinOptions struct {
	projectDir string
}

// NewJoinCommand creates a new cobra.Command for `docker project join`
func NewJoinCommand(dockerCli *command.DockerCli) *cobra.Command {
	var opts joinOptions

	cmd := &cobra.Command{
		Use:   "join ID",
		Short: "Join a Docker project",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJoin(dockerCli, &opts, args[0])
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.projectDir, "dir", "d", "", "Target directory (default is current directory)")

	return cmd
}

func runJoin(dockerCli *command.DockerCli, opts *joinOptions, ID string) error {

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

	proj, err := project.Init(dir, ID, false)
	if err != nil {
		return err
	}

	err = projectutil.SaveInRecentProjects(proj)
	if err != nil {
		return err
	}

	fmt.Fprintf(dockerCli.Out(), "joined project %s!\n", proj.ID())

	return nil
}
