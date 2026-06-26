package swarm

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type joinOptions struct {
	remote     string
	listenAddr NodeAddrOption
	// Not a NodeAddrOption because it has no default port.
	advertiseAddr string
	dataPathAddr  string
	token         string
	availability  string
}

func newJoinCommand(dockerCLI command.Cli) *cobra.Command {
	opts := joinOptions{
		listenAddr: NewListenAddrOption(),
	}

	cmd := &cobra.Command{
		Use:   "join [OPTIONS] HOST:PORT",
		Short: "Join a swarm as a node and/or manager",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.remote = args[0]
			return runJoin(cmd.Context(), dockerCLI, cmd.Flags(), opts)
		},
		Annotations: map[string]string{
			"version": "1.24",
			"swarm":   "", // swarm join does not require swarm to be active, and is always available on API 1.24 and up
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.Var(&opts.listenAddr, flagListenAddr, `Listen address (format: "<ip|interface>[:port]")`)
	flags.StringVar(&opts.advertiseAddr, flagAdvertiseAddr, "", `Advertised address (format: "<ip|interface>[:port]")`)
	flags.StringVar(&opts.dataPathAddr, flagDataPathAddr, "", `Address or interface to use for data path traffic (format: "<ip|interface>")`)
	flags.SetAnnotation(flagDataPathAddr, "version", []string{"1.31"})
	flags.StringVar(&opts.token, flagToken, "", "Token for entry into the swarm")
	flags.StringVar(&opts.availability, flagAvailability, "active", `Availability of the node ("active", "pause", "drain")`)
	return cmd
}

func runJoin(ctx context.Context, dockerCLI command.Cli, flags *pflag.FlagSet, opts joinOptions) error {
	apiClient := dockerCLI.Client()

	var availability swarm.NodeAvailability
	if flags.Changed(flagAvailability) {
		switch a := swarm.NodeAvailability(strings.ToLower(opts.availability)); a {
		case swarm.NodeAvailabilityActive, swarm.NodeAvailabilityPause, swarm.NodeAvailabilityDrain:
			availability = a
		default:
			return fmt.Errorf("invalid availability %q, only active, pause and drain are supported", opts.availability)
		}
	}

	_, err := apiClient.SwarmJoin(ctx, client.SwarmJoinOptions{
		JoinToken:     opts.token,
		ListenAddr:    opts.listenAddr.String(),
		AdvertiseAddr: opts.advertiseAddr,
		DataPathAddr:  opts.dataPathAddr,
		RemoteAddrs:   []string{opts.remote},
		Availability:  availability,
	})
	if err != nil {
		return err
	}

	res, err := apiClient.Info(ctx, client.InfoOptions{})
	if err != nil {
		return err
	}

	if res.Info.Swarm.ControlAvailable {
		_, _ = fmt.Fprintln(dockerCLI.Out(), "This node joined a swarm as a manager.")
	} else {
		_, _ = fmt.Fprintln(dockerCLI.Out(), "This node joined a swarm as a worker.")
	}
	return nil
}
