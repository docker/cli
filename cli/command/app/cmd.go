package app

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/container"
	"github.com/docker/cli/cli/command/image"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// cliAdapter adds convenience methods to the command.Cli interface
// for building images, running containers, and copying files from containers
type cliAdapter interface {
	command.Cli
	//
	RunBuild(context.Context, *image.BuildOptions) error
	RunRun(context.Context, *pflag.FlagSet, *container.RunOptions, *container.ContainerOptions) error
	RunCopy(context.Context, *container.CopyOptions) error
}

type dockerCliAdapter struct {
	command.Cli
}

func newDockerCliAdapter(c command.Cli) cliAdapter {
	return &dockerCliAdapter{
		c,
	}
}

func (r *dockerCliAdapter) RunBuild(ctx context.Context, buildOpts *image.BuildOptions) error {
	return image.RunBuild(ctx, r, buildOpts)
}

func (r *dockerCliAdapter) RunRun(ctx context.Context, flags *pflag.FlagSet, runOpts *container.RunOptions, containerOpts *container.ContainerOptions) error {
	return container.RunRun(ctx, r, flags, runOpts, containerOpts)
}

func (r *dockerCliAdapter) RunCopy(ctx context.Context, copyOpts *container.CopyOptions) error {
	return container.RunCopy(ctx, r, copyOpts)
}

// NewAppCommand returns a cobra command for `app` subcommands
func NewAppCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app",
		Short: "Manage application with Docker",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
	}
	cmd.AddCommand(
		NewInstallCommand(dockerCli),
		NewLaunchCommand(dockerCli),
		NewRemoveCommand(dockerCli),
	)
	return cmd
}

func markFlagsHiddenExcept(cmd *cobra.Command, unhidden ...string) {
	contains := func(n string) bool {
		for _, v := range unhidden {
			if v == n {
				return true
			}
		}
		return false
	}
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		name := flag.Name
		if !contains(name) {
			flag.Hidden = true
		}
	})
}
