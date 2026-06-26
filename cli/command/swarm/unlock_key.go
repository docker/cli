package swarm

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type unlockKeyOptions struct {
	rotate bool
	quiet  bool
}

func newUnlockKeyCommand(dockerCLI command.Cli) *cobra.Command {
	opts := unlockKeyOptions{}

	cmd := &cobra.Command{
		Use:   "unlock-key [OPTIONS]",
		Short: "Manage the unlock key",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUnlockKey(cmd.Context(), dockerCLI, opts)
		},
		Annotations: map[string]string{
			"version": "1.24",
			"swarm":   "manager",
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.rotate, flagRotate, false, "Rotate unlock key")
	flags.BoolVarP(&opts.quiet, flagQuiet, "q", false, "Only display token")

	return cmd
}

func runUnlockKey(ctx context.Context, dockerCLI command.Cli, opts unlockKeyOptions) error {
	apiClient := dockerCLI.Client()

	if opts.rotate {
		res, err := apiClient.SwarmInspect(ctx, client.SwarmInspectOptions{})
		if err != nil {
			return err
		}

		if !res.Swarm.Spec.EncryptionConfig.AutoLockManagers {
			return errors.New("cannot rotate because autolock is not turned on")
		}

		_, err = apiClient.SwarmUpdate(ctx, client.SwarmUpdateOptions{
			Version: res.Swarm.Version,
			Spec:    res.Swarm.Spec,

			RotateManagerUnlockKey: true,
		})
		if err != nil {
			return err
		}

		if !opts.quiet {
			_, _ = fmt.Fprintln(dockerCLI.Out(), "Successfully rotated manager unlock key.")
		}
	}

	resp, err := apiClient.SwarmGetUnlockKey(ctx)
	if err != nil {
		return fmt.Errorf("could not fetch unlock key: %w", err)
	}

	if resp.Key == "" {
		return errors.New("no unlock key is set")
	}

	if opts.quiet {
		_, _ = fmt.Fprintln(dockerCLI.Out(), resp.Key)
		return nil
	}

	printUnlockCommand(dockerCLI.Out(), resp.Key)
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
