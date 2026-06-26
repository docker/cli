package swarm

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/streams"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newUnlockCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock",
		Short: "Unlock swarm",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUnlock(cmd.Context(), dockerCLI)
		},
		Annotations: map[string]string{
			"version": "1.24",
			"swarm":   "manager",
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	return cmd
}

func runUnlock(ctx context.Context, dockerCLI command.Cli) error {
	apiClient := dockerCLI.Client()

	// First see if the node is actually part of a swarm, and if it is actually locked first.
	// If it's in any other state than locked, don't ask for the key.
	res, err := apiClient.Info(ctx, client.InfoOptions{})
	if err != nil {
		return err
	}

	switch res.Info.Swarm.LocalNodeState {
	case swarm.LocalNodeStateInactive:
		return errors.New("error: this node is not part of a swarm")
	case swarm.LocalNodeStateLocked:
		break
	case swarm.LocalNodeStatePending, swarm.LocalNodeStateActive, swarm.LocalNodeStateError:
		return errors.New("error: swarm is not locked")
	}

	key, err := readKey(dockerCLI.In(), "Enter unlock key: ")
	if err != nil {
		return err
	}

	_, err = apiClient.SwarmUnlock(ctx, client.SwarmUnlockOptions{
		Key: key,
	})
	return err
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
