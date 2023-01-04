package swarm

import (
	"net"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/spf13/cobra"
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
			return command.RunSwarm(dockerCli)
		},
		Annotations: map[string]string{
			"version": "1.24",
			"swarm":   "", // swarm init does not require swarm to be active, and is always available on API 1.24 and up
		},
		ValidArgsFunction: completion.NoComplete,
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
	flags.Var(newIPNetSliceValue([]net.IPNet{}, &opts.defaultAddrPools), flagDefaultAddrPool, "default address pool in CIDR format")
	flags.SetAnnotation(flagDefaultAddrPool, "version", []string{"1.39"})
	flags.Uint32Var(&opts.DefaultAddrPoolMaskLength, flagDefaultAddrPoolMaskLength, 24, "default address pool subnet mask length")
	flags.SetAnnotation(flagDefaultAddrPoolMaskLength, "version", []string{"1.39"})
	addSwarmFlags(flags, &opts.swarmOptions)
	return cmd
}
