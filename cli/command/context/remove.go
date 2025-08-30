package context

import (
	"errors"
	"fmt"
	"os"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// RemoveOptions are the options used to remove contexts
//
// Deprecated: this type was for internal use and will be removed in the next release.
type RemoveOptions struct {
	Force bool
}

// removeOptions are the options used to remove contexts.
type removeOptions struct {
	force bool
}

func newRemoveCommand(dockerCLI command.Cli) *cobra.Command {
	var opts removeOptions
	cmd := &cobra.Command{
		Use:     "rm CONTEXT [CONTEXT...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more contexts",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(dockerCLI, opts, args)
		},
		ValidArgsFunction: completeContextNames(dockerCLI, -1, false),
	}
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Force the removal of a context in use")
	return cmd
}

// RunRemove removes one or more contexts
//
// Deprecated: this function was for internal use and will be removed in the next release.
func RunRemove(dockerCLI command.Cli, opts removeOptions, names []string) error {
	return runRemove(dockerCLI, opts, names)
}

// runRemove removes one or more contexts.
func runRemove(dockerCLI command.Cli, opts removeOptions, names []string) error {
	var errs []error
	currentCtx := dockerCLI.CurrentContext()
	for _, name := range names {
		if name == "default" {
			errs = append(errs, errors.New(`context "default" cannot be removed`))
		} else if err := doRemove(dockerCLI, name, name == currentCtx, opts.force); err != nil {
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
		return notFoundErr{fmt.Errorf("context %q does not exist", name)}
	}
	// Ignore other errors; if relevant, they will produce an error when
	// performing the actual delete.
	return nil
}

type notFoundErr struct{ error }

func (notFoundErr) NotFound() {}
