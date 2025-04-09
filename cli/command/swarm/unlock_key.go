package swarm

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/api/types/swarm"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type unlockKeyOptions struct {
	rotate bool
	quiet  bool
}

func newUnlockKeyCommand(dockerCli command.Cli) *cobra.Command {
	opts := unlockKeyOptions{}

	cmd := &cobra.Command{
		Use:   "unlock-key [OPTIONS]",
		Short: "Manage the unlock key",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUnlockKey(cmd.Context(), dockerCli, opts)
		},
		Annotations: map[string]string{
			"version": "1.24",
			"swarm":   "manager",
		},
		ValidArgsFunction: completion.NoComplete,
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.rotate, flagRotate, false, "Rotate unlock key")
	flags.BoolVarP(&opts.quiet, flagQuiet, "q", false, "Only display token")

	return cmd
}

func runUnlockKey(ctx context.Context, dockerCLI command.Cli, opts unlockKeyOptions) error {
	apiClient := dockerCLI.Client()

	if opts.rotate {
		flags := swarm.UpdateFlags{RotateManagerUnlockKey: true}

		sw, err := apiClient.SwarmInspect(ctx)
		if err != nil {
			return err
		}

		if !sw.Spec.EncryptionConfig.AutoLockManagers {
			return errors.New("cannot rotate because autolock is not turned on")
		}

		if err := apiClient.SwarmUpdate(ctx, sw.Version, sw.Spec, flags); err != nil {
			return err
		}

		if !opts.quiet {
			_, _ = fmt.Fprintln(dockerCLI.Out(), "Successfully rotated manager unlock key.")
		}
	}

	unlockKeyResp, err := apiClient.SwarmGetUnlockKey(ctx)
	if err != nil {
		return errors.Wrap(err, "could not fetch unlock key")
	}

	if unlockKeyResp.UnlockKey == "" {
		return errors.New("no unlock key is set")
	}

	if opts.quiet {
		_, _ = fmt.Fprintln(dockerCLI.Out(), unlockKeyResp.UnlockKey)
		return nil
	}

	printUnlockCommand(dockerCLI.Out(), unlockKeyResp.UnlockKey)
	return nil
}

func printUnlockCommand(out io.Writer, unlockKey string) {
	if len(unlockKey) > 0 {
		_, _ = fmt.Fprintf(out, "To unlock a swarm manager after it restarts, "+
			"run the `docker swarm unlock`\ncommand and provide the following key:\n\n    %s\n\n"+
			"Remember to store this key in a password manager, since without it you\n"+
			"will not be able to restart the manager.\n", unlockKey)
	}
}
