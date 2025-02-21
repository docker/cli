package context // import "docker.com/cli/v28/cli/command/context"

import (
	"errors"
	"fmt"
	"os"

	"github.com/docker/cli/v28/cli"
	"github.com/docker/cli/v28/cli/command"
	"github.com/docker/docker/errdefs"
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
func RunRemove(dockerCLI command.Cli, opts RemoveOptions, names []string) error {
	var errs []error
	currentCtx := dockerCLI.CurrentContext()
	for _, name := range names {
		if name == "default" {
			errs = append(errs, errors.New(`context "default" cannot be removed`))
		} else if err := doRemove(dockerCLI, name, name == currentCtx, opts.Force); err != nil {
			errs = append(errs, err)
		} else {
			_, _ = fmt.Fprintln(dockerCLI.Out(), name)
		}
	}
	return errors.Join(errs...)
}

func doRemove(dockerCli command.Cli, name string, isCurrent, force bool) error {
	if isCurrent {
		if !force {
			return fmt.Errorf("context %q is in use, set -f flag to force remove", name)
		}
		// fallback to DOCKER_HOST
		cfg := dockerCli.ConfigFile()
		cfg.CurrentContext = ""
		if err := cfg.Save(); err != nil {
			return err
		}
	}

	if !force {
		// TODO(thaJeztah): instead of checking before removing, can we make ContextStore().Remove() return a proper errdef and ignore "not found" errors?
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
		return errdefs.NotFound(fmt.Errorf("context %q does not exist", name))
	}
	// Ignore other errors; if relevant, they will produce an error when
	// performing the actual delete.
	return nil
}
