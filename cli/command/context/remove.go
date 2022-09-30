package context

import (
	"fmt"
	"os"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/errdefs"
	"github.com/pkg/errors"
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
			errs = append(errs, err.Error())
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
	if isCurrent {
		if !force {
			return errors.Errorf("context %q is in use, set -f flag to force remove", name)
		}
		// fallback to DOCKER_HOST
		cfg := dockerCli.ConfigFile()
		cfg.CurrentContext = ""
		if err := cfg.Save(); err != nil {
			return err
		}
	}

	if !force {
		if err := checkContextExists(dockerCli, name); err != nil {
			return err
		}
	}
	return dockerCli.ContextStore().Remove(name)
}

// checkContextExists returns an error if the context directory does not exist.
func checkContextExists(dockerCli command.Cli, name string) error {
	contextDir := dockerCli.ContextStore().GetStorageInfo(name).MetadataPath
	_, err := os.Stat(contextDir)
	if os.IsNotExist(err) {
		return errdefs.NotFound(errors.Errorf("context %q does not exist", name))
	}
	// Ignore other errors; if relevant, they will produce an error when
	// performing the actual delete.
	return nil
}
