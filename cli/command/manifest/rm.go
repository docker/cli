package manifest // import "docker.com/cli/v28/cli/command/manifest"

import (
	"context"
	"errors"

	"github.com/docker/cli/v28/cli"
	"github.com/docker/cli/v28/cli/command"
	manifeststore "github.com/docker/cli/v28/cli/manifest/store"
	"github.com/spf13/cobra"
)

func newRmManifestListCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm MANIFEST_LIST [MANIFEST_LIST...]",
		Short: "Delete one or more manifest lists from local storage",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd.Context(), dockerCLI.ManifestStore(), args)
		},
	}

	return cmd
}

func runRemove(ctx context.Context, store manifeststore.Store, targets []string) error {
	var errs []error
	for _, target := range targets {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		targetRef, err := normalizeReference(target)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		_, err = store.GetList(targetRef)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		err = store.Remove(targetRef)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
