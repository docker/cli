package context

import (
	"errors"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// RemoveOptions are the options used to remove contexts
type RemoveOptions struct {
	Force bool
}

func newRemoveCommand(dockerCli command.Cli) *cobra.Command {
	var opts RemoveOptions
	cmd := &cobra.Command{
		Use:     "rm CONTEXT [CONTEXT...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more contexts",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunRemove(dockerCli, opts, args)
		},
	}
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Force the removal of a context in use")
	return cmd
}

// RunRemove removes one or more contexts
func RunRemove(dockerCli command.Cli, opts RemoveOptions, names []string) error {
	var errs []string
	currentCtx := dockerCli.CurrentContext()
	for _, name := range names {
		if name == "default" {
			errs = append(errs, `default: context "default" cannot be removed`)
		} else if err := doRemove(dockerCli, name, name == currentCtx, opts.Force); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %s", name, err))
		} else {
			fmt.Fprintln(dockerCli.Out(), name)
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func doRemove(dockerCli command.Cli, name string, isCurrent, force bool) error {
	if _, err := dockerCli.ContextStore().GetMetadata(name); err != nil {
		return err
	}
	if isCurrent {
		if !force {
			return errors.New("context is in use, set -f flag to force remove")
		}
		// fallback to DOCKER_HOST
		cfg := dockerCli.ConfigFile()
		cfg.CurrentContext = ""
		if err := cfg.Save(); err != nil {
			return err
		}
	}
	return dockerCli.ContextStore().Remove(name)
}
