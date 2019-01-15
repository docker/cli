package context

import (
	"errors"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

type removeOptions struct {
	force bool
}

func newRemoveCommand(dockerCli command.Cli) *cobra.Command {
	var opts removeOptions
	cmd := &cobra.Command{
		Use:     "rm CONTEXT [CONTEXT...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more contexts",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(dockerCli, opts, args)
		},
	}
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Force the removal of a context in use")
	return cmd
}

func runRemove(dockerCli command.Cli, opts removeOptions, names []string) error {
	var errs []string
	currentCtx := dockerCli.CurrentContext()
	for _, name := range names {
		if name == "default" {
			errs = append(errs, `default: context "default" cannot be removed`)
		} else if err := doRemove(dockerCli, name, name == currentCtx, opts.force); err != nil {
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
	if _, err := dockerCli.ContextStore().GetContextMetadata(name); err != nil {
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
	return dockerCli.ContextStore().RemoveContext(name)
}
