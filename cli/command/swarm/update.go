package swarm

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/moby/moby/api/types/swarm"
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
		ValidArgsFunction: completion.NoComplete,
	}

	cmd.Flags().BoolVar(&opts.autolock, flagAutolock, false, "Change manager autolocking setting (true|false)")
	addSwarmFlags(cmd.Flags(), &opts)
	return cmd
}

func runUpdate(ctx context.Context, dockerCli command.Cli, flags *pflag.FlagSet, opts swarmOptions) error {
	client := dockerCli.Client()

	var updateFlags swarm.UpdateFlags

	swarmInspect, err := client.SwarmInspect(ctx)
	if err != nil {
		return err
	}

	prevAutoLock := swarmInspect.Spec.EncryptionConfig.AutoLockManagers

	opts.mergeSwarmSpec(&swarmInspect.Spec, flags, &swarmInspect.ClusterInfo.TLSInfo.TrustRoot)

	curAutoLock := swarmInspect.Spec.EncryptionConfig.AutoLockManagers

	err = client.SwarmUpdate(ctx, swarmInspect.Version, swarmInspect.Spec, updateFlags)
	if err != nil {
		return err
	}

	fmt.Fprintln(dockerCli.Out(), "Swarm updated.")

	if curAutoLock && !prevAutoLock {
		unlockKeyResp, err := client.SwarmGetUnlockKey(ctx)
		if err != nil {
			return errors.Wrap(err, "could not fetch unlock key")
		}
		printUnlockCommand(dockerCli.Out(), unlockKeyResp.UnlockKey)
	}

	return nil
}
