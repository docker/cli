package swarm

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types/swarm"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type initOptions struct {
	swarmOptions
	listenAddr NodeAddrOption
	// Not a NodeAddrOption because it has no default port.
	advertiseAddr             string
	dataPathAddr              string
	dataPathPort              uint32
	forceNewCluster           bool
	availability              string
	defaultAddrPools          []net.IPNet
	DefaultAddrPoolMaskLength uint32
}

func newInitCommand(dockerCli command.Cli) *cobra.Command {
	opts := initOptions{
		listenAddr: NewListenAddrOption(),
	}

	cmd := &cobra.Command{
		Use:   "init [OPTIONS]",
		Short: "Initialize a swarm",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(dockerCli, cmd.Flags(), opts)
		},
	}

	flags := cmd.Flags()
	flags.Var(&opts.listenAddr, flagListenAddr, "Listen address (format: <ip|interface>[:port])")
	flags.StringVar(&opts.advertiseAddr, flagAdvertiseAddr, "", "Advertised address (format: <ip|interface>[:port])")
	flags.StringVar(&opts.dataPathAddr, flagDataPathAddr, "", "Address or interface to use for data path traffic (format: <ip|interface>)")
	flags.SetAnnotation(flagDataPathAddr, "version", []string{"1.31"})
	flags.Uint32Var(&opts.dataPathPort, flagDataPathPort, 0, "Port number to use for data path traffic (1024 - 49151). If no value is set or is set to 0, the default port (4789) is used.")
	flags.SetAnnotation(flagDataPathPort, "version", []string{"1.40"})
	flags.BoolVar(&opts.forceNewCluster, "force-new-cluster", false, "Force create a new cluster from current state")
	flags.BoolVar(&opts.autolock, flagAutolock, false, "Enable manager autolocking (requiring an unlock key to start a stopped manager)")
	flags.StringVar(&opts.availability, flagAvailability, "active", `Availability of the node ("active"|"pause"|"drain")`)
	flags.IPNetSliceVar(&opts.defaultAddrPools, flagDefaultAddrPool, []net.IPNet{}, "default address pool in CIDR format")
	flags.SetAnnotation(flagDefaultAddrPool, "version", []string{"1.39"})
	flags.Uint32Var(&opts.DefaultAddrPoolMaskLength, flagDefaultAddrPoolMaskLength, 24, "default address pool subnet mask length")
	flags.SetAnnotation(flagDefaultAddrPoolMaskLength, "version", []string{"1.39"})
	addSwarmFlags(flags, &opts.swarmOptions)
	return cmd
}

func runInit(dockerCli command.Cli, flags *pflag.FlagSet, opts initOptions) error {
	var defaultAddrPool []string

	client := dockerCli.Client()
	ctx := context.Background()

	for _, p := range opts.defaultAddrPools {
		defaultAddrPool = append(defaultAddrPool, p.String())
	}
	req := swarm.InitRequest{
		ListenAddr:       opts.listenAddr.String(),
		AdvertiseAddr:    opts.advertiseAddr,
		DataPathAddr:     opts.dataPathAddr,
		DataPathPort:     opts.dataPathPort,
		DefaultAddrPool:  defaultAddrPool,
		ForceNewCluster:  opts.forceNewCluster,
		Spec:             opts.swarmOptions.ToSpec(flags),
		AutoLockManagers: opts.swarmOptions.autolock,
		SubnetSize:       opts.DefaultAddrPoolMaskLength,
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

	nodeID, err := client.SwarmInit(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "could not choose an IP address to advertise") || strings.Contains(err.Error(), "could not find the system's IP address") {
			return errors.New(err.Error() + " - specify one with --advertise-addr")
		}
		return err
	}

	fmt.Fprintf(dockerCli.Out(), "Swarm initialized: current node (%s) is now a manager.\n\n", nodeID)

	if err := printJoinCommand(ctx, dockerCli, nodeID, true, false); err != nil {
		return err
	}

	fmt.Fprint(dockerCli.Out(), "To add a manager to this swarm, run 'docker swarm join-token manager' and follow the instructions.\n\n")

	if req.AutoLockManagers {
		unlockKeyResp, err := client.SwarmGetUnlockKey(ctx)
		if err != nil {
			return errors.Wrap(err, "could not fetch unlock key")
		}
		printUnlockCommand(dockerCli.Out(), unlockKeyResp.UnlockKey)
	}

	return nil
}
