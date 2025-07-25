package swarm

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/streams"
	"github.com/moby/moby/api/types/swarm"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newUnlockCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock",
		Short: "Unlock swarm",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUnlock(cmd.Context(), dockerCli)
		},
		Annotations: map[string]string{
			"version": "1.24",
			"swarm":   "manager",
		},
		ValidArgsFunction: completion.NoComplete,
	}

	return cmd
}

func runUnlock(ctx context.Context, dockerCli command.Cli) error {
	client := dockerCli.Client()

	// First see if the node is actually part of a swarm, and if it is actually locked first.
	// If it's in any other state than locked, don't ask for the key.
	info, err := client.Info(ctx)
	if err != nil {
		return err
	}

	switch info.Swarm.LocalNodeState {
	case swarm.LocalNodeStateInactive:
		return errors.New("Error: This node is not part of a swarm")
	case swarm.LocalNodeStateLocked:
		break
	case swarm.LocalNodeStatePending, swarm.LocalNodeStateActive, swarm.LocalNodeStateError:
		return errors.New("Error: swarm is not locked")
	}

	key, err := readKey(dockerCli.In(), "Enter unlock key: ")
	if err != nil {
		return err
	}

	return client.SwarmUnlock(ctx, swarm.UnlockRequest{
		UnlockKey: key,
	})
}

func readKey(in *streams.In, prompt string) (string, error) {
	if in.IsTerminal() {
		fmt.Print(prompt)
		dt, err := term.ReadPassword(int(in.FD()))
		fmt.Println()
		return string(dt), err
	}
	key, err := bufio.NewReader(in).ReadString('\n')
	if err == io.EOF {
		err = nil
	}
	return strings.TrimSpace(key), err
}
