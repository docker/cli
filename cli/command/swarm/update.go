package swarm

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newUpdateCommand(dockerCli command.Cli) *cobra.Command {
	opts := swarmOptions{}

	cmd := &cobra.Command{
		Use:   "update [OPTIONS]",
		Short: "Update the swarm",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd.Context(), dockerCli, cmd.Flags(), opts)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().NFlag() == 0 {
				return pflag.ErrHelp
			}
			return nil
		},
		Annotations: map[string]string{
			"version": "1.24",
			"swarm":   "manager",
		},
		ValidArgsFunction: cobra.NoFileCompletions,
	}

	cmd.Flags().BoolVar(&opts.autolock, flagAutolock, false, "Change manager autolocking setting (true|false)")
	addSwarmFlags(cmd.Flags(), &opts)
	return cmd
}

func runUpdate(ctx context.Context, dockerCLI command.Cli, flags *pflag.FlagSet, opts swarmOptions) error {
	apiClient := dockerCLI.Client()

	swarmInspect, err := apiClient.SwarmInspect(ctx)
	if err != nil {
		return err
	}

	prevAutoLock := swarmInspect.Spec.EncryptionConfig.AutoLockManagers

	opts.mergeSwarmSpec(&swarmInspect.Spec, flags, &swarmInspect.ClusterInfo.TLSInfo.TrustRoot)

	curAutoLock := swarmInspect.Spec.EncryptionConfig.AutoLockManagers

	err = apiClient.SwarmUpdate(ctx, swarmInspect.Version, swarmInspect.Spec, client.SwarmUpdateFlags{})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(dockerCLI.Out(), "Swarm updated.")

	if curAutoLock && !prevAutoLock {
		unlockKeyResp, err := apiClient.SwarmGetUnlockKey(ctx)
		if err != nil {
			return errors.Wrap(err, "could not fetch unlock key")
		}
		printUnlockCommand(dockerCLI.Out(), unlockKeyResp.UnlockKey)
	}

	return nil
}
