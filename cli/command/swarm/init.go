package swarm

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
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

func newInitCommand(dockerCLI command.Cli) *cobra.Command {
	opts := initOptions{
		listenAddr: NewListenAddrOption(),
	}

	cmd := &cobra.Command{
		Use:   "init [OPTIONS]",
		Short: "Initialize a swarm",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd.Context(), dockerCLI, cmd.Flags(), opts)
		},
		Annotations: map[string]string{
			"version": "1.24",
			"swarm":   "", // swarm init does not require swarm to be active, and is always available on API 1.24 and up
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.Var(&opts.listenAddr, flagListenAddr, `Listen address (format: "<ip|interface>[:port]")`)
	flags.StringVar(&opts.advertiseAddr, flagAdvertiseAddr, "", `Advertised address (format: "<ip|interface>[:port]")`)
	flags.StringVar(&opts.dataPathAddr, flagDataPathAddr, "", `Address or interface to use for data path traffic (format: "<ip|interface>")`)
	_ = flags.SetAnnotation(flagDataPathAddr, "version", []string{"1.31"})
	flags.Uint32Var(&opts.dataPathPort, flagDataPathPort, 0, "Port number to use for data path traffic (1024 - 49151). If no value is set or is set to 0, the default port (4789) is used.")
	_ = flags.SetAnnotation(flagDataPathPort, "version", []string{"1.40"})
	flags.BoolVar(&opts.forceNewCluster, "force-new-cluster", false, "Force create a new cluster from current state")
	flags.BoolVar(&opts.autolock, flagAutolock, false, "Enable manager autolocking (requiring an unlock key to start a stopped manager)")
	flags.StringVar(&opts.availability, flagAvailability, "active", `Availability of the node ("active", "pause", "drain")`)
	flags.IPNetSliceVar(&opts.defaultAddrPools, flagDefaultAddrPool, []net.IPNet{}, "default address pool in CIDR format")
	_ = flags.SetAnnotation(flagDefaultAddrPool, "version", []string{"1.39"})
	flags.Uint32Var(&opts.DefaultAddrPoolMaskLength, flagDefaultAddrPoolMaskLength, 24, "default address pool subnet mask length")
	_ = flags.SetAnnotation(flagDefaultAddrPoolMaskLength, "version", []string{"1.39"})
	addSwarmFlags(flags, &opts.swarmOptions)
	return cmd
}

func runInit(ctx context.Context, dockerCLI command.Cli, flags *pflag.FlagSet, opts initOptions) error {
	apiClient := dockerCLI.Client()

	// TODO(thaJeztah): change opts.defaultAddrPools a []netip.Prefix; see https://github.com/docker/cli/pull/6545#discussion_r2420361609
	defaultAddrPool := make([]netip.Prefix, 0, len(opts.defaultAddrPools))
	for _, p := range opts.defaultAddrPools {
		if len(p.IP) == 0 {
			continue
		}
		ip := p.IP.To4()
		if ip == nil {
			ip = p.IP.To16()
		}
		addr, ok := netip.AddrFromSlice(ip)
		if !ok {
			return fmt.Errorf("invalid IP address: %s", p.IP)
		}
		ones, _ := p.Mask.Size()
		defaultAddrPool = append(defaultAddrPool, netip.PrefixFrom(addr, ones))
	}
	var availability swarm.NodeAvailability
	if flags.Changed(flagAvailability) {
		switch a := swarm.NodeAvailability(strings.ToLower(opts.availability)); a {
		case swarm.NodeAvailabilityActive, swarm.NodeAvailabilityPause, swarm.NodeAvailabilityDrain:
			availability = a
		default:
			return fmt.Errorf("invalid availability %q, only active, pause and drain are supported", opts.availability)
		}
	}

	res, err := apiClient.SwarmInit(ctx, client.SwarmInitOptions{
		ListenAddr:       opts.listenAddr.String(),
		AdvertiseAddr:    opts.advertiseAddr,
		DataPathAddr:     opts.dataPathAddr,
		DataPathPort:     opts.dataPathPort,
		DefaultAddrPool:  defaultAddrPool,
		ForceNewCluster:  opts.forceNewCluster,
		Spec:             opts.swarmOptions.ToSpec(flags),
		AutoLockManagers: opts.swarmOptions.autolock,
		Availability:     availability,
		SubnetSize:       opts.DefaultAddrPoolMaskLength,
	})
	if err != nil {
		if strings.Contains(err.Error(), "could not choose an IP address to advertise") || strings.Contains(err.Error(), "could not find the system's IP address") {
			return fmt.Errorf("%w - specify one with --advertise-addr", err)
		}
		return err
	}

	_, _ = fmt.Fprintf(dockerCLI.Out(), "Swarm initialized: current node (%s) is now a manager.\n\n", res.NodeID)

	if err := printJoinCommand(ctx, dockerCLI, res.NodeID, true, false); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(dockerCLI.Out(), "To add a manager to this swarm, run 'docker swarm join-token manager' and follow the instructions.")

	if opts.swarmOptions.autolock {
		resp, err := apiClient.SwarmGetUnlockKey(ctx)
		if err != nil {
			return fmt.Errorf("could not fetch unlock key: %w", err)
		}
		printUnlockCommand(dockerCLI.Out(), resp.Key)
	}

	return nil
}
