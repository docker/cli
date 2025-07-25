package swarm

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/api/types/swarm"
	"github.com/pkg/errors"
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

func newJoinCommand(dockerCli command.Cli) *cobra.Command {
	opts := joinOptions{
		listenAddr: NewListenAddrOption(),
	}

	cmd := &cobra.Command{
		Use:   "join [OPTIONS] HOST:PORT",
		Short: "Join a swarm as a node and/or manager",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.remote = args[0]
			return runJoin(cmd.Context(), dockerCli, cmd.Flags(), opts)
		},
		Annotations: map[string]string{
			"version": "1.24",
			"swarm":   "", // swarm join does not require swarm to be active, and is always available on API 1.24 and up
		},
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

func runJoin(ctx context.Context, dockerCli command.Cli, flags *pflag.FlagSet, opts joinOptions) error {
	client := dockerCli.Client()

	req := swarm.JoinRequest{
		JoinToken:     opts.token,
		ListenAddr:    opts.listenAddr.String(),
		AdvertiseAddr: opts.advertiseAddr,
		DataPathAddr:  opts.dataPathAddr,
		RemoteAddrs:   []string{opts.remote},
	}
	if flags.Changed(flagAvailability) {
		availability := swarm.NodeAvailability(strings.ToLower(opts.availability))
		switch availability {
		case swarm.NodeAvailabilityActive, swarm.NodeAvailabilityPause, swarm.NodeAvailabilityDrain:
			req.Availability = availability
		default:
			return errors.Errorf("invalid availability %q, only active, pause and drain are supported", opts.availability)
		}
	}

	err := client.SwarmJoin(ctx, req)
	if err != nil {
		return err
	}

	info, err := client.Info(ctx)
	if err != nil {
		return err
	}

	if info.Swarm.ControlAvailable {
		fmt.Fprintln(dockerCli.Out(), "This node joined a swarm as a manager.")
	} else {
		fmt.Fprintln(dockerCli.Out(), "This node joined a swarm as a worker.")
	}
	return nil
}
