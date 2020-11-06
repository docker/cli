package manifest

import (
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newRmManifestListCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm MANIFEST_LIST [MANIFEST_LIST...]",
		Short: "Delete one or more manifest lists from local storage",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRm(dockerCli, args)
		},
	}

	return cmd
}

func runRm(dockerCli command.Cli, targets []string) error {
	var errs []string
	for _, target := range targets {
		targetRef, refErr := normalizeReference(target)
		if refErr != nil {
			errs = append(errs, refErr.Error())
			continue
		}
		_, searchErr := dockerCli.ManifestStore().GetList(targetRef)
		if searchErr != nil {
			errs = append(errs, searchErr.Error())
			continue
		}
		rmErr := dockerCli.ManifestStore().Remove(targetRef)
		if rmErr != nil {
			errs = append(errs, rmErr.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
