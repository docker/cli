package swarm

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newUpdateCommand(dockerCLI command.Cli) *cobra.Command {
	opts := swarmOptions{}

	cmd := &cobra.Command{
		Use:   "update [OPTIONS]",
		Short: "Update the swarm",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd.Context(), dockerCLI, cmd.Flags(), opts)
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
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	cmd.Flags().BoolVar(&opts.autolock, flagAutolock, false, "Change manager autolocking setting (true|false)")
	addSwarmFlags(cmd.Flags(), &opts)
	return cmd
}

func runUpdate(ctx context.Context, dockerCLI command.Cli, flags *pflag.FlagSet, opts swarmOptions) error {
	apiClient := dockerCLI.Client()

	sw, err := apiClient.SwarmInspect(ctx, client.SwarmInspectOptions{})
	if err != nil {
		return err
	}

	prevAutoLock := sw.Swarm.Spec.EncryptionConfig.AutoLockManagers

	opts.mergeSwarmSpec(&sw.Swarm.Spec, flags, &sw.Swarm.ClusterInfo.TLSInfo.TrustRoot)

	curAutoLock := sw.Swarm.Spec.EncryptionConfig.AutoLockManagers

	_, err = apiClient.SwarmUpdate(ctx, client.SwarmUpdateOptions{
		Version: sw.Swarm.Version,
		Spec:    sw.Swarm.Spec,
	})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(dockerCLI.Out(), "Swarm updated.")

	if curAutoLock && !prevAutoLock {
		resp, err := apiClient.SwarmGetUnlockKey(ctx)
		if err != nil {
			return fmt.Errorf("could not fetch unlock key: %w", err)
		}
		printUnlockCommand(dockerCLI.Out(), resp.Key)
	}

	return nil
}
